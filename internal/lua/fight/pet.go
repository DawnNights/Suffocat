package fight

import (
	"suffocat/internal/protocol/schema"

	lua "github.com/yuin/gopher-lua"
)

// 出战背包精灵
type FightBagPet struct {
	CatchTime uint32
	ID        uint32
	HP        uint32
	MaxHP     uint32
	Name      string
}

// 当前在场精灵
type FightActivePet struct {
	FightBagPet
	FightLV      FightLevel
	FightStatus_ []uint8
	SkillList_   []schema.PetSkillInfo

	FightStatus *lua.LTable
	SkillList   *lua.LTable
}

func (p FightActivePet) IsDeath() bool {
	return p.MaxHP == 0
}

// 强化等级-6~6
type FightLevel struct {
	Atk      int // 攻击
	Def      int // 防御
	SpAtk    int // 特攻
	SpDef    int // 特防
	Speed    int // 速度
	Accuracy int // 命中
}

func (fl FightLevel) Total() int {
	return fl.Atk + fl.Def + fl.SpAtk + fl.SpDef + fl.Speed + fl.Accuracy
}

// 技能结算
type SkillResult struct {
	Skill        schema.PetSkillMoreInfo
	AskTimes     uint32 // 连击次数
	IsCrit       bool   // 是否暴击
	Damage       uint32 // 红伤
	GainDamage   uint32 // 粉伤
	RealDamage   uint32 // 真伤
	ZhuijiDamage uint32 // 二段伤
}
