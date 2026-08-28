package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ToString ep moi kieu du lieu sang chuoi an toan
func ToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// ToInt ep cac kieu float64 tu JSON hoac chuoi so ve int
func ToInt(val any) (int, error) {
	if val == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		return int(i), err
	case string:
		v = strings.TrimSpace(v)
		i, err := strconv.Atoi(v)
		if err != nil {
			// Thu parse sang float truoc neu la chuoi "10.0"
			f, fErr := strconv.ParseFloat(v, 64)
			if fErr == nil {
				return int(f), nil
			}
			return 0, fmt.Errorf("cannot parse string '%s' to int: %w", v, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for int conversion", val)
	}
}

// ToFloat ep cac kieu ve float64
func ToFloat(val any) (float64, error) {
	if val == nil {
		return 0, fmt.Errorf("cannot convert nil to float64")
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		v = strings.TrimSpace(v)
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string '%s' to float64: %w", v, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for float64 conversion", val)
	}
}

// ToBool nhan dien cac gia tri boolean, chuoi "true"/"false" va so 1/0
func ToBool(val any) (bool, error) {
	if val == nil {
		return false, fmt.Errorf("cannot convert nil to bool")
	}
	switch v := val.(type) {
	case bool:
		return v, nil
	case int, int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		switch v {
		case "true", "1", "yes", "y", "on":
			return true, nil
		case "false", "0", "no", "n", "off", "":
			return false, nil
		default:
			return false, fmt.Errorf("cannot parse string '%s' to bool", v)
		}
	default:
		return false, fmt.Errorf("unsupported type %T for bool conversion", val)
	}
}

// ToStringSlice ep mang JSON []any thanh []string
func ToStringSlice(val any) ([]string, error) {
	if val == nil {
		return nil, nil
	}
	switch v := val.(type) {
	case []string:
		return v, nil
	case []any:
		res := make([]string, len(v))
		for i, item := range v {
			res[i] = ToString(item)
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported type %T for []string conversion", val)
	}
}
