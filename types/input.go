package types

import (
	"context"
	"fmt"

	"github.com/duongess/khoai-link-sdk/utils"
)

// TaskInput dai dien cho tham so dau vao cua mot task
type TaskInput map[string]any

// TaskOutput dai dien cho ket qua tra ve cua mot task
type TaskOutput map[string]any

// TaskHandler la signature ham xu ly nghiep vu ma lap trinh vien viet
type TaskHandler func(ctx context.Context, in TaskInput) (TaskOutput, error)

// --- Cac ham Helper giup get data an toan (Safe Casting) ---

func (in TaskInput) Get(key string) (any, bool) {
	val, ok := in[key]
	return val, ok
}

func (in TaskInput) GetString(key string) (string, error) {
	val, ok := in[key]
	if !ok {
		return "", fmt.Errorf("missing required input field: '%s'", key)
	}
	return utils.ToString(val), nil
}

func (in TaskInput) GetInt(key string) (int, error) {
	val, ok := in[key]
	if !ok {
		return 0, fmt.Errorf("missing required input field: '%s'", key)
	}
	return utils.ToInt(val)
}

func (in TaskInput) GetFloat(key string) (float64, error) {
	val, ok := in[key]
	if !ok {
		return 0, fmt.Errorf("missing required input field: '%s'", key)
	}
	return utils.ToFloat(val)
}

func (in TaskInput) GetBool(key string) (bool, error) {
	val, ok := in[key]
	if !ok {
		return false, fmt.Errorf("missing required input field: '%s'", key)
	}
	return utils.ToBool(val)
}

// GetMap lay du lieu nested map
func (in TaskInput) GetMap(key string) (map[string]any, error) {
	val, ok := in[key]
	if !ok {
		return nil, fmt.Errorf("missing required map field: '%s'", key)
	}
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field '%s' is not a valid JSON object", key)
	}
	return m, nil
}
