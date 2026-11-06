package common

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// ToLuaList 把 Go slice 转为 Lua table（1-based）
// T 可以是 struct / *struct / 基础类型
func ToLuaList[T any](L *lua.LState, list []T) *lua.LTable {
	t := L.NewTable()
	for i, v := range list {
		t.RawSetInt(i+1, luar.New(L, v))
	}
	return t
}

// ToGoList 把 Lua table（1-based array）转换为 Go slice
//
// 支持：int / string / bool / float64 等基础类型
//
// 示例：
// nums := ToGoList[int](tb)
// names := ToGoList[string](tb)
func ToGoList[T any](lv lua.LValue) ([]T, error) {
	if lv.Type() != lua.LTTable {
		return nil, fmt.Errorf("expected table, got %s", lv.Type().String())
	}

	tb := lv.(*lua.LTable)
	result := make([]T, 0, tb.Len())

	for i := 1; i <= tb.Len(); i++ {
		v := tb.RawGetInt(i)

		val, err := convertLuaValue[T](v)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}

		result = append(result, val)
	}

	return result, nil
}

// ToGoListAny 将 Lua table 转为 []any（非泛型版本，用于 T=any 场景）
func ToGoListAny(lv lua.LValue) ([]any, error) {
	if lv.Type() != lua.LTTable {
		return nil, fmt.Errorf("expected table, got %s", lv.Type().String())
	}

	tb := lv.(*lua.LTable)
	result := make([]any, 0, tb.Len())

	for i := 1; i <= tb.Len(); i++ {
		v := tb.RawGetInt(i)
		switch val := v.(type) {
		case lua.LNumber:
			result = append(result, float64(val))
		case lua.LString:
			result = append(result, string(val))
		case lua.LBool:
			result = append(result, bool(val))
		default:
			return nil, fmt.Errorf("index %d: unsupported type %T", i, v)
		}
	}

	return result, nil
}

func convertLuaValue[T any](v lua.LValue) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int:
		if n, ok := v.(lua.LNumber); ok {
			return any(int(n)).(T), nil
		}

	case int64:
		if n, ok := v.(lua.LNumber); ok {
			return any(int64(n)).(T), nil
		}

	case uint32:
		if n, ok := v.(lua.LNumber); ok {
			// 可选：防止负数转 uint32
			if n < 0 {
				return zero, fmt.Errorf("negative number cannot convert to uint32: %v", n)
			}
			return any(uint32(n)).(T), nil
		}

	case float64:
		if n, ok := v.(lua.LNumber); ok {
			return any(float64(n)).(T), nil
		}

	case string:
		if s, ok := v.(lua.LString); ok {
			return any(string(s)).(T), nil
		}

	case bool:
		if b, ok := v.(lua.LBool); ok {
			return any(bool(b)).(T), nil
		}
	}

	return zero, fmt.Errorf("unsupported or mismatched type: %T", v)
}
