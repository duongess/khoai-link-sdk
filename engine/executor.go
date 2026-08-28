package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"khoai-link-sdk/client"
	"khoai-link-sdk/types"

	"github.com/khoai-link-protocol/core"
)

type Executor struct {
	nodeID       string
	handlers     map[string]types.TaskHandler
	p2pClient    *client.P2pClient
	store        *BufferStore
	resolver     *Resolver
	singleFlight *Group
}

func NewExecutor(nodeID string, p2pClient *client.P2pClient) *Executor {
	store := NewBufferStore(5 * time.Minute)
	return &Executor{
		nodeID:       nodeID,
		handlers:     make(map[string]types.TaskHandler),
		p2pClient:    p2pClient,
		store:        store,
		resolver:     NewResolver(store),
		singleFlight: NewSingleFlightGroup(),
	}
}

func (e *Executor) RegisterTaskHandler(taskName string, handler types.TaskHandler) {
	e.handlers[taskName] = handler
}

// ExecuteAndDispatch chay node hien tai theo StepID va tim cac Node con tiep theo de dispatch
func (e *Executor) ExecuteAndDispatch(ctx context.Context, reqID string, plan *core.ExecutionPlan, currentStepID string, runtimeOutputs map[string]map[string]any) (any, error) {
	if plan == nil {
		return nil, errors.New("execution plan cannot be nil")
	}

	// 1. Tim ExecutionNode hien tai trong Flat Nodes array
	var currentNode *core.ExecutionNode
	var currentIndex int
	for i := range plan.Nodes {
		if plan.Nodes[i].StepID == currentStepID {
			currentNode = &plan.Nodes[i]
			currentIndex = i
			break
		}
	}
	if currentNode == nil {
		return nil, fmt.Errorf("step_id '%s' not found in execution plan", currentStepID)
	}

	// 2. Kiem tra Timeout quy dinh rieng cua Step hoac dung Timeout cua toan Plan
	timeout := time.Duration(plan.TimeoutMs) * time.Millisecond
	if currentNode.TimeoutMs > 0 {
		timeout = time.Duration(currentNode.TimeoutMs) * time.Millisecond
	}
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 3. Resolve InputBindings
	resolvedInput, err := e.resolver.ResolveInputs(plan.PlanID, currentNode.Inputs, runtimeOutputs)
	if err != nil {
		return nil, fmt.Errorf("resolve inputs failed at step '%s': %w", currentStepID, err)
	}

	// 4. Tim handler va thuc thi
	handler, exists := e.handlers[currentNode.TaskName]
	if !exists {
		return nil, fmt.Errorf("handler for task '%s' not registered on node '%s'", currentNode.TaskName, e.nodeID)
	}

	flightKey := fmt.Sprintf("%s:%s", plan.PlanID, currentStepID)
	outputRaw, err := e.singleFlight.Do(flightKey, func() (any, error) {
		return handler(stepCtx, resolvedInput)
	})
	if err != nil {
		if currentNode.OnFailure == core.FailFast {
			return nil, fmt.Errorf("task '%s' failed (fail_fast policy triggered): %w", currentNode.TaskName, err)
		}
		// Neu la FailPartial: ghi nhan loi va tiep tuc
		outputRaw = map[string]any{"error": err.Error(), "partial": true}
	}

	// 5. Luu ket qua buoc vao BufferStore va Runtime Context
	stepOutput, ok := outputRaw.(map[string]any)
	if !ok {
		stepOutput = map[string]any{"result": outputRaw}
	}

	if runtimeOutputs == nil {
		runtimeOutputs = make(map[string]map[string]any)
	}
	runtimeOutputs[currentStepID] = stepOutput
	e.store.Set(e.store.MakeKey(plan.PlanID, currentStepID), stepOutput, 10*time.Minute)

	// 6. Kiem tra Node tiep theo trong chuoi Flat DAG (hoac chot ket qua neu la node cuoi)
	nextIndex := currentIndex + 1
	if nextIndex >= len(plan.Nodes) || currentNode.MergePolicy == core.MergeNone {
		return runtimeOutputs, nil
	}

	nextNode := plan.Nodes[nextIndex]

	// 7. Chuyen tiep (Dispatch P2P) sang Node tiep theo dung dia chi NodeIP trong Plan
	forwardPayload := map[string]any{
		"execution_plan":  plan,
		"current_step_id": nextNode.StepID,
		"runtime_outputs": runtimeOutputs,
	}

	resp, err := e.p2pClient.ForwardTask(ctx, nextNode.NodeIP, reqID, forwardPayload)
	if err != nil {
		return nil, fmt.Errorf("forwarding to next node '%s' (%s) failed: %w", nextNode.NodeID, nextNode.NodeIP, err)
	}

	return resp.Result, nil
}
