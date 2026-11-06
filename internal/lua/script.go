package lua

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type LuaScript struct {
	Type  string
	Main  *lua.LFunction
	Setup *lua.LFunction

	Name    string
	Author  string
	Desc    string
	Version string
}

// FromSource 从源码加载脚本
func (s *LuaScript) FromSource(source string, L *lua.LState) error {
	fn, err := L.LoadString(source)
	if err != nil {
		return fmt.Errorf("compile source failed: %w", err)
	}

	L.Push(fn)
	if err := L.PCall(0, 1, nil); err != nil {
		return fmt.Errorf("execute source failed: %w", err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	tbl, ok := ret.(*lua.LTable)
	if !ok {
		return fmt.Errorf("lua文件返回值必须是 table")
	}

	newLuaScript, err := parseLuaScript(tbl)
	if err == nil {
		*s = newLuaScript
	}
	return err
}

// ================= 公有解析逻辑（原来抽出来的） =================
func parseLuaScript(tbl *lua.LTable) (LuaScript, error) {
	var script LuaScript
	var err error

	if script.Type, err = getStringField(tbl, "Type"); err != nil {
		return script, err
	}
	if script.Type != "UNITY" && script.Type != "FLASH" {
		return script, fmt.Errorf("Type 字段必须是 UNITY 或 FLASH")
	}

	if script.Main, err = getFuncField(tbl, "Main"); err != nil {
		return script, err
	}
	// Setup 可选：纯执行脚本可以不提供配置项。
	if fn, ok := tbl.RawGetString("Setup").(*lua.LFunction); ok {
		script.Setup = fn
	}

	if script.Name, err = getStringField(tbl, "Name"); err != nil {
		return script, err
	}
	if script.Author, err = getStringField(tbl, "Author"); err != nil {
		script.Author = "匿名"
	}
	if script.Desc, err = getStringField(tbl, "Desc"); err != nil {
		script.Desc = ""
	}
	if script.Version, err = getStringField(tbl, "Version"); err != nil {
		script.Version = "0.0.1"
	}

	return script, nil
}

func getStringField(tbl *lua.LTable, key string) (string, error) {
	if v, ok := tbl.RawGetString(key).(lua.LString); ok {
		return string(v), nil
	}
	return "", fmt.Errorf("%s 字段必须是 string", key)
}

func getFuncField(tbl *lua.LTable, key string) (*lua.LFunction, error) {
	if fn, ok := tbl.RawGetString(key).(*lua.LFunction); ok {
		return fn, nil
	}
	return nil, fmt.Errorf("%s 字段必须是 function", key)
}
