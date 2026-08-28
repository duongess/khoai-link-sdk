package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// findRootNodes loc ra tat ca cac node khong phu thuoc vao bat ky step nao khac
func (e *Executor) findRootNodes(plan *core.ExecutionPlan) []core.ExecutionNode {
	var roots []core.ExecutionNode
	for _, node := range plan.Nodes {
		isRoot := true
		for _, binding := range node.Inputs {
			if binding.FromStepID != "" {
				isRoot = false
				break
			}
		}
		if isRoot {
			roots = append(roots, node)
		}
	}
	return roots
}

// ExecuteAndDispatch xu ly ca 2 truong hop: Khoi dong Plan (currentStepID == "") va chay tiep 1 Step cu the
func (e *Executor) ExecuteAndDispatch(ctx context.Context, reqID string, plan *core.ExecutionPlan, currentStepID string, runtimeOutputs map[string]map[string]any) (any, error) {
	if plan == nil {
		return nil, errors.New("execution plan cannot be nil")
	}

	if runtimeOutputs == nil {
		runtimeOutputs = make(map[string]map[string]any)
	}

	// TRUONG HOP 1: Khoi dong Plan tong tu Gateway (currentStepID rong) -> Tim va chay cac Root Nodes
	if currentStepID == "" {
		roots := e.findRootNodes(plan)
		if len(roots) == 0 {
			return nil, errors.New("invalid execution plan: no root nodes found (possible circular dependency)")
		}

		// Neu chi co 1 root node va nam ngay tren Node hien tai -> Chay luon
		if len(roots) == 1 && roots[0].NodeID == e.nodeID {
			return e.ExecuteAndDispatch(ctx, reqID, plan, roots[0].StepID, runtimeOutputs)
		}

		// Neu co nhieu Root Nodes hoac Root Node nam o may khac -> Ban song song (Fan-out)
		var wg sync.WaitGroup
		errChan := make(chan error, len(roots))

		for _, root := range roots {
			wg.Add(1)
			go func(targetRoot core.ExecutionNode) {
				defer wg.Done()
				if targetRoot.NodeID == e.nodeID {
					_, err := e.ExecuteAndDispatch(ctx, reqID, plan, targetRoot.StepID, runtimeOutputs)
					if err != nil {
						errChan <- err
					}
				} else {
					// Root node nam o may khac -> Dispatch P2P sang may do
					payload := map[string]any{
						"execution_plan":  plan,
						"current_step_id": targetRoot.StepID,
						"runtime_outputs": runtimeOutputs,
					}
					_, err := e.p2pClient.ForwardTask(ctx, targetRoot.NodeIP, reqID, payload)
					if err != nil {
						errChan <- fmt.Errorf("failed to trigger root node '%s' at %s: %w", targetRoot.StepID, targetRoot.NodeIP, err)
					}
				}
			}(root)
		}

		wg.Wait()
		close(errChan)

		if len(errChan) > 0 {
			var combinedErr error
			for err := range errChan {
				combinedErr = errors.Join(combinedErr, err)
			}
			return nil, combinedErr
		}
		return map[string]string{"status": "plan_started"}, nil
	}

	// TRUONG HOP 2: Thuc thi Step cu the tren Node hien tai
	var currentNode *core.ExecutionNode
	for i := range plan.Nodes {
		if plan.Nodes[i].StepID == currentStepID {
			currentNode = &plan.Nodes[i]
			break
		}
	}
	if currentNode == nil {
		return nil, fmt.Errorf("step_id '%s' not found in plan", currentStepID)
	}

	// 1. Resolve Inputs
	resolvedInput, err := e.resolver.ResolveInputs(plan.PlanID, currentNode.Inputs, runtimeOutputs)
	if err != nil {
		return nil, fmt.Errorf("resolve inputs for step '%s' failed: %w", currentStepID, err)
	}

	// 2. Chay Task Handler cuc bo
	handler, exists := e.handlers[currentNode.TaskName]
	if !exists {
		return nil, fmt.Errorf("handler for task '%s' not found on node '%s'", currentNode.TaskName, e.nodeID)
	}

	flightKey := fmt.Sprintf("%s:%s", plan.PlanID, currentStepID)
	outputRaw, err := e.singleFlight.Do(flightKey, func() (any, error) {
		return handler(ctx, resolvedInput)
	})
	if err != nil {
		if currentNode.OnFailure == core.FailFast {
			return nil, fmt.Errorf("task '%s' failed: %w", currentNode.TaskName, err)
		}
		outputRaw = map[string]any{"error": err.Error(), "partial": true}
	}

	// 3. Luu output cua buoc hien tai
	stepOutput, ok := outputRaw.(map[string]any)
	if !ok {
		stepOutput = map[string]any{"result": outputRaw}
	}
	runtimeOutputs[currentStepID] = stepOutput
	e.store.Set(e.store.MakeKey(plan.PlanID, currentStepID), stepOutput, 10*time.Minute)

	// 4. Tim tat ca cac Node con tiep theo can ket qua cua buoc hien tai
	var nextNodes []core.ExecutionNode
	for _, n := range plan.Nodes {
		for _, binding := range n.Inputs {
			if binding.FromStepID == currentStepID {
				nextNodes = append(nextNodes, n)
				break
			}
		}
	}

	// Neu khong con node nao phu thuoc -> Leaf node (Chot ket qua)
	if len(nextNodes) == 0 {
		return runtimeOutputs, nil
	}

	// 5. Chuyen tiep (Dispatch) sang cac node tiep theo
	for _, nextNode := range nextNodes {
		payload := map[string]any{
			"execution_plan":  plan,
			"current_step_id": nextNode.StepID,
			"runtime_outputs": runtimeOutputs,
		}
		_, err := e.p2pClient.ForwardTask(ctx, nextNode.NodeIP, reqID, payload)
		if err != nil {
			return nil, fmt.Errorf("dispatch to step '%s' (%s) failed: %w", nextNode.StepID, nextNode.NodeIP, err)
		}
	}

	return runtimeOutputs, nil
}
