package engine

import (
	"fmt"

	"github.com/duongess/khoai-link-sdk/types"

	"github.com/duongess/khoai-link-protocol/core"
)

type Resolver struct {
	store *BufferStore
}

func NewResolver(store *BufferStore) *Resolver {
	return &Resolver{store: store}
}

// ResolveInputs giai ma toan bo map[string]InputBinding thanh TaskInput cu the
func (r *Resolver) ResolveInputs(planID string, bindings map[string]core.InputBinding, localContext map[string]map[string]any) (types.TaskInput, error) {
	resolved := make(types.TaskInput)

	for paramName, binding := range bindings {
		// 1. Gia tri Tinh (Static) da biet tu luc Planner phan tich
		if binding.Static != nil {
			resolved[paramName] = binding.Static
			continue
		}

		// 2. Gia tri Dong (Runtime Binding) phu thuoc output cua step cha
		if binding.FromStepID != "" {
			var stepOutputs map[string]any
			var found bool

			// Kiem tra trong local context truoc
			if localContext != nil {
				stepOutputs, found = localContext[binding.FromStepID]
			}

			// Neu khong co trong local context, tim trong BufferStore (lay tu cac buoc chay truoc do)
			if !found && r.store != nil {
				storeKey := r.store.MakeKey(planID, binding.FromStepID)
				if val, ok := r.store.Get(storeKey); ok {
					if m, ok := val.(map[string]any); ok {
						stepOutputs = m
						found = true
					}
				}
			}

			if !found {
				return nil, fmt.Errorf("dependency missing: output for step_id '%s' not found for input '%s'", binding.FromStepID, paramName)
			}

			// Boc tach field cu the hoac lay toan bo
			if binding.FromField != "" {
				fieldVal, ok := stepOutputs[binding.FromField]
				if !ok {
					return nil, fmt.Errorf("field '%s' not found in output of step_id '%s'", binding.FromField, binding.FromStepID)
				}
				resolved[paramName] = fieldVal
			} else {
				resolved[paramName] = stepOutputs
			}
			continue
		}

		return nil, fmt.Errorf("invalid input binding for parameter '%s': neither static nor runtime reference provided", paramName)
	}

	return resolved, nil
}
