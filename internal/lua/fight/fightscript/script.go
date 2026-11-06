package fightscript

// SeerSet 赛尔配置 (火焰、套装、称号等)
type SeerSet struct {
	BattleFire       uint32 `yaml:"battleFire"`
	Suit             uint32 `yaml:"suit"`
	Equip            uint32 `yaml:"equip"`
	Title            uint32 `yaml:"title"`
	EnableEquipments bool   `yaml:"enableEquipments"`
	EnableTitle      bool   `yaml:"enableTitle"`
}

// ScriptConfig 脚本全局设置
type ScriptConfig struct {
	FightRoundLimit uint32 `yaml:"fightRoundLimit"`
	HPLimit         bool   `yaml:"hpLimit"`
	ClothMust       bool   `yaml:"clothMust"`
	TitleMust       bool   `yaml:"titleMust"`
	SupportPetsMust bool   `yaml:"supportPetsMust"`
}

// Use 动作类型及其参数
type Use struct {
	Type string `yaml:"type"`
	Arg  any    `yaml:"arg"` // 通常为 uint32 或 nil
}

// Condition 单个条件
type Condition struct {
	Left     string `yaml:"left"`
	LeftArg  any    `yaml:"leftArg"` // uint32 / string / nil
	Operator string `yaml:"operator"`
	Right    any    `yaml:"right"` // 通常为数字
}

// Step 一个动作步骤
type Step struct {
	Use Use         `yaml:"use"`
	If  []Condition `yaml:"if"`
}

// PetConfig 精灵附加配置
type PetConfig struct {
	FightItems  []uint32 `yaml:"fightItems"`
	Level       uint32   `yaml:"level"`
	DeathChange int      `yaml:"deathChange"`
}

// RosterPet 阵容中的一只精灵
type RosterPet struct {
	ID             uint32    `yaml:"id"`
	Config         PetConfig `yaml:"config"`
	ActionSequence [][]Step  `yaml:"actionSequence"`
}

// MetaData 脚本元数据
type MetaData struct {
	Name string
}

// FightScript 完整的战斗脚本
type FightScript struct {
	MetaData     MetaData     `yaml:"metadata"`
	SeerSet      SeerSet      `yaml:"seerSet"`
	ScriptConfig ScriptConfig `yaml:"scriptConfig"`
	RosterSet    []RosterPet  `yaml:"rosterSet"`
}
