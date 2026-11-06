package petbag

import (
	"suffocat/internal/common"
	"suffocat/internal/protocol/schema"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type PetInfo struct {
	schema.PetInfo
	SkillInfos    *lua.LTable // 技能列表（最多5个）
	EffectList    *lua.LTable // 效果列表
	PetValueInfos *lua.LTable // 属性值信息（六维）
}

type PetEffectInfo struct {
	schema.PetEffectInfo
	ArgsAll    *lua.LTable // 附加参数总数
	AddArray   *lua.LTable // 加成数值数组
	LevelArray *lua.LTable // 等级修正数组
}

func luaEffectList(l *lua.LState, list []schema.PetEffectInfo) []PetEffectInfo {
	var result []PetEffectInfo

	for _, eff := range list {
		ArgsAll := l.NewTable()
		for k, v := range eff.ArgsAll {
			ArgsAll.RawSetInt(int(k), luar.New(l, v))
		}

		result = append(result, PetEffectInfo{
			PetEffectInfo: eff,
			ArgsAll:       ArgsAll,
			AddArray:      common.ToLuaList(l, eff.AddArray),
			LevelArray:    common.ToLuaList(l, eff.LevelArray),
		})
	}

	return result
}
