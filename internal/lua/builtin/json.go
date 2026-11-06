package builtin

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// jsonDumps 将任意 Lua 值转为 JSON 字符串并压入 Lua 栈。
// 用法：json.dumps(value [, indent])
//   - indent 为整数，指定缩进空格数（例如 2 表示用两个空格缩进）；
//   - 若 indent ≤ 0 或不提供，则输出紧凑 JSON。
func jsonDumps(L *lua.LState) int {
	val := L.CheckAny(1)
	indent := 0
	if L.GetTop() >= 2 {
		indent = L.CheckInt(2)
	}

	visited := make(map[*lua.LTable]bool)
	result := dumpLuaValue(val, visited)
	result = cleanMetatableNil(result)

	var (
		jsonBytes []byte
		err       error
	)
	if indent > 0 {
		jsonBytes, err = json.MarshalIndent(result, "", strings.Repeat(" ", indent))
	} else {
		jsonBytes, err = json.Marshal(result)
	}

	if err != nil {
		L.Push(lua.LString(fmt.Sprintf("error: %v", err)))
		return 1
	}
	L.Push(lua.LString(string(jsonBytes)))
	return 1
}

// dumpLuaValue 将 lua.LValue 转换为 JSON 兼容的 Go 值。
func dumpLuaValue(val lua.LValue, visited map[*lua.LTable]bool) any {
	switch v := val.(type) {
	case *lua.LTable:
		if visited[v] {
			return nil // 循环引用 -> null
		}
		visited[v] = true

		obj := make(map[string]any)
		v.ForEach(func(key, value lua.LValue) {
			// Lua 的 key 转换为字符串，数字 key 会变成 "123" 这样的字符串 key
			keyStr := key.String()
			obj[keyStr] = dumpLuaValue(value, visited)
		})
		return obj

	case *lua.LUserData:
		obj := make(map[string]any)
		if v.Value == nil {
			obj["go_type"] = nil
			return obj
		}
		obj["go_type"] = fmt.Sprintf("%T", v.Value)
		// 将 Go 结构体/值展开到同一层级（也可放在子字段里，这里与原逻辑保持相似）
		goObj := goValueToJSON(reflect.ValueOf(v.Value))
		maps.Copy(obj, goObj)
		return obj

	default:
		return basicToGoValue(val)
	}
}

// basicToGoValue 处理 Lua 基本类型（nil, bool, number, string）。
func basicToGoValue(v lua.LValue) any {
	switch t := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(t)
	case lua.LNumber:
		return float64(t)
	case lua.LString:
		return string(t)
	default:
		return v.String()
	}
}

// goValueToJSON 将 reflect.Value 转换为 JSON 兼容的 map[string]any。
func goValueToJSON(rv reflect.Value) map[string]any {
	obj := make(map[string]any)

	if !rv.IsValid() {
		return obj
	}
	// 解引用指针
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return obj
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if field.PkgPath != "" { // 跳过未导出字段
				continue
			}
			obj[field.Name] = convertReflectValue(rv.Field(i))
		}
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			obj[keyStr] = convertReflectValue(rv.MapIndex(key))
		}
	case reflect.Slice, reflect.Array:
		// 切片/数组不能直接转为 map，统一放在名为 "data" 的字段中
		arr := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr[i] = convertReflectValue(rv.Index(i))
		}
		obj["data"] = arr
	default:
		obj["value"] = convertReflectValue(rv)
	}

	return obj
}

// convertReflectValue 递归转换任意 reflect.Value 为 JSON 兼容类型。
func convertReflectValue(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}
	// 处理指针
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		return goValueToJSON(rv)
	case reflect.Map:
		return goValueToJSON(rv)
	case reflect.Slice, reflect.Array:
		arr := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr[i] = convertReflectValue(rv.Index(i))
		}
		return arr
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint()
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	default:
		if rv.CanInterface() {
			return fmt.Sprintf("%v", rv.Interface())
		}
		return "<unexported>"
	}
}

// cleanMetatableNil 递归清理对象中值为 {"Metatable":"nil"} 的键
func cleanMetatableNil(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, sub := range val {
			cleaned := cleanMetatableNil(sub)
			if isMetatableNil(cleaned) {
				delete(val, k)
			} else {
				val[k] = cleaned
			}
		}
		return val
	case []any:
		for i, item := range val {
			val[i] = cleanMetatableNil(item)
		}
		return val
	default:
		return v
	}
}

// isMetatableNil 判断一个值是否为 map[string]any{"Metatable":"nil"}
func isMetatableNil(v any) bool {
	m, ok := v.(map[string]any)
	if !ok || len(m) != 1 {
		return false
	}
	s, ok := m["Metatable"].(string)
	return ok && s == "nil"
}

// jsonLoads 将 JSON 字符串解析为 Lua table。
// 用法：json.loads(str)
func jsonLoads(L *lua.LState) int {
	str := L.CheckString(1)

	var raw any
	if err := json.Unmarshal([]byte(str), &raw); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("error: %v", err)))
		return 2
	}

	result := goValueToLua(L, raw)
	L.Push(result)
	return 1
}

// goValueToLua 将 Go 值（json.Unmarshal 的结果）递归转为 lua.LValue。
func goValueToLua(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch val := v.(type) {
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []any:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, goValueToLua(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range val {
			tbl.RawSetString(k, goValueToLua(L, item))
		}
		return tbl
	default:
		return lua.LNil
	}
}
