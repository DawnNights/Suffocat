package builtin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"suffocat/internal/config"
	"suffocat/internal/lua/fight/fightscript"

	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v3"
)

// fightscript.New(source) -> table|nil
func fightscriptNew(L *lua.LState) int {
	source := L.CheckString(1)
	var script fightscript.FightScript
	if yaml.Unmarshal([]byte(source), &script) != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(fightScriptToLuaTable(L, &script))
	return 1
}

// fightscript.LoadLocal(path) -> table|nil
// 只允许读取战斗方案编辑器目录内的文件。
func fightscriptLoadLocal(L *lua.LState) int {
	path, ok := resolvePlanPath(L.CheckString(1))
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	data, err := os.ReadFile(path)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	var script fightscript.FightScript
	if yaml.Unmarshal(data, &script) != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(fightScriptToLuaTable(L, &script))
	return 1
}

// LoadPlan 从战斗方案编辑器目录加载方案，供 Go 侧逻辑复用。
func LoadPlan(rel string) (*fightscript.FightScript, bool) {
	path, ok := resolvePlanPath(rel)
	if !ok {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var script fightscript.FightScript
	if yaml.Unmarshal(data, &script) != nil {
		return nil, false
	}
	return &script, true
}

func resolvePlanPath(rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	rel = filepath.FromSlash(strings.ReplaceAll(rel, "\\", "/"))
	if filepath.IsAbs(rel) {
		return "", false
	}

	root, err := filepath.Abs(config.PlansDir())
	if err != nil {
		return "", false
	}

	candidates := []string{rel}
	if filepath.Ext(rel) == "" {
		candidates = append(candidates, rel+".yaml", rel+".yml")
	}

	for _, candidate := range candidates {
		full, err := filepath.Abs(filepath.Join(root, candidate))
		if err != nil {
			continue
		}
		if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
			continue
		}
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return full, true
		}
	}

	return "", false
}

// fightScriptToLuaTable 通过 JSON 中转将 *FightScript 转为 Lua table
func fightScriptToLuaTable(L *lua.LState, script *fightscript.FightScript) lua.LValue {
	jsonBytes, err := json.Marshal(script)
	if err != nil {
		return lua.LNil
	}
	var raw any
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return lua.LNil
	}
	return goValueToLua(L, raw)
}
