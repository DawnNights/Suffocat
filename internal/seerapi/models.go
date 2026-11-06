package seerapi

// Stats 六维属性。
type Stats struct {
	Atk   int `json:"atk"`
	Def   int `json:"def"`
	SpAtk int `json:"sp_atk"`
	SpDef int `json:"sp_def"`
	Spd   int `json:"spd"`
	HP    int `json:"hp"`
}

// PetSkill 精灵可学习技能。
type PetSkill struct {
	ID       uint32 `json:"id"`
	Level    uint32 `json:"level"`
	Special  bool   `json:"special"`
	Advanced bool   `json:"advanced"`
	Fifth    bool   `json:"fifth"`
}

// Pet 精灵基础数据。
type Pet struct {
	ID          uint32     `json:"id"`
	Name        string     `json:"name"`
	ResourceID  uint32     `json:"resource_id"`
	CatchRate   int        `json:"catch_rate"`
	Type        uint32     `json:"type"`
	Gender      uint32     `json:"gender"`
	BaseStats   Stats      `json:"base_stats"`
	Skills      []PetSkill `json:"skills"`
	SoulmarkIDs []uint32   `json:"soulmark_ids"`
}

// Skill 技能数据。
type Skill struct {
	ID       uint32  `json:"id"`
	Name     string  `json:"name"`
	Category uint32  `json:"category"`
	Type     uint32  `json:"type"`
	Power    uint32  `json:"power"`
	MaxPP    uint32  `json:"max_pp"`
	Accuracy uint32  `json:"accuracy"`
	Priority int     `json:"priority"`
	CritRate float64 `json:"crit_rate"`
	MustHit  bool    `json:"must_hit"`
}

// Item 道具数据。
type Item struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// MintMark 刻印数据。
type MintMark struct {
	ID        uint32 `json:"id"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	ClassID   uint32 `json:"class_id"`
	ExtraAttr Stats  `json:"extra_attr"`
}

// Suit 套装数据。
type Suit struct {
	ID       uint32   `json:"id"`
	Name     string   `json:"name"`
	EquipIDs []uint32 `json:"equip_ids"`
}

// Equip 套装部件数据。
type Equip struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Part uint32 `json:"part"`
}

// Cloth 是游戏切换套装封包所需的部件集合。
type Cloth struct {
	Name  string
	ID    uint32
	Head  uint32
	Face  uint32
	Body  uint32
	Waist uint32
	Foot  uint32
}

func (c Cloth) ToBody() []uint32 {
	var result []uint32
	for _, part := range []uint32{c.Head, c.Face, c.Body, c.Waist, c.Foot} {
		if part != 0 {
			result = append(result, part)
		}
	}
	return append([]uint32{uint32(len(result))}, result...)
}

// Factor 因子道具资源，包含兑换精灵、第五技能和魂印所需的数量。
type Factor struct {
	ID              uint32 `json:"id"`
	Name            string `json:"name"`
	PetLimit        bool   `json:"pet_limit"`
	PetConsume      uint32 `json:"pet_consume"`
	SkillConsume    uint32 `json:"skill_consume"`
	SoulmarkConsume uint32 `json:"soulmark_consume"`
	ItemID          uint32 `json:"item_id"`
	DesignID        uint32 `json:"design_id"`
	NeedPetID       uint32 `json:"need_pet_id"`
	PetID           uint32 `json:"pet_id"`
	SkillID         uint32 `json:"skill_id"`
	SoulmarkID      uint32 `json:"soulmark_id"`
}

// FactorDesign 因子关卡资源。
type FactorDesign struct {
	ID               uint32             `json:"id"`
	Times            uint32             `json:"times"`
	TimeValue        uint32             `json:"time_value"`
	ProgressValue    uint32             `json:"progress_value"`
	FailTimes        uint32             `json:"fail_times"`
	AchievementID    uint32             `json:"achievement_id"`
	RewardID         uint32             `json:"reward_id"`
	MintmarkRewardID uint32             `json:"mintmark_reward_id"`
	Difficulties     []FactorDifficulty `json:"difficulties"`
}

// FactorDifficulty 单个难度下的有序 BOSS、禁用精灵和规则。
type FactorDifficulty struct {
	Level       string       `json:"level"`
	RewardCount uint32       `json:"reward_count"`
	Tasks       []FactorTask `json:"tasks"`
	BannedPets  []uint32     `json:"banned_pets"`
	Rules       []FactorRule `json:"rules"`
}

type FactorTask struct {
	Desc        string `json:"desc"`
	IsSinglePet bool   `json:"is_single_pet"`
	BossPetID   uint32 `json:"boss_pet_id"`
}

type FactorRule struct {
	ID       uint32 `json:"id"`
	Tips     string `json:"tips"`
	FailTips string `json:"fail_tips"`
}

// Difficulty 按模式返回难度：1 简单、2 普通、3 困难。
func (d FactorDesign) Difficulty(mode int) *FactorDifficulty {
	level := "easy"
	switch mode {
	case 2:
		level = "normal"
	case 3:
		level = "hard"
	}

	for i := range d.Difficulties {
		if d.Difficulties[i].Level == level {
			return &d.Difficulties[i]
		}
	}

	index := mode - 1
	if index >= 0 && index < len(d.Difficulties) {
		return &d.Difficulties[index]
	}
	return nil
}
