package fight

import (
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/schema"
	"suffocat/internal/seerapi"

	lua "github.com/yuin/gopher-lua"
)

type FightEnemy struct {
	PetBag_    []FightBagPet
	ActivePet  *FightActivePet
	LastResult SkillResult

	PetBag *lua.LTable
}

type FightPlayer struct {
	FightEnemy
	c        protocol.SeerConn
	UserID   uint32
	Name     string
	IsAction bool
	IsEscape bool
	Clothes_ []uint32

	Clothes    *lua.LTable
	actionDone chan struct{}
}

func (e *FightEnemy) Update(atk schema.AttackValue) {
	// 更新背包精灵状态
	for _, change := range atk.ChangeHps {
		for idx := range e.PetBag_ {
			if e.PetBag_[idx].CatchTime == e.ActivePet.CatchTime {
				e.PetBag_[idx].HP = atk.RemainHP
				e.PetBag_[idx].MaxHP = atk.MaxHP
				continue
			}

			if e.PetBag_[idx].CatchTime == change.CatchTime {
				e.PetBag_[idx].HP = change.HP
				e.PetBag_[idx].MaxHP = change.MaxHP
			}
		}
	}

	// 更新出战精灵状态
	e.ActivePet.HP = atk.RemainHP
	e.ActivePet.MaxHP = atk.MaxHP
	e.ActivePet.FightLV = FightLevel{
		Atk:      atk.SideEffects[0],
		Def:      atk.SideEffects[1],
		SpAtk:    atk.SideEffects[2],
		SpDef:    atk.SideEffects[3],
		Speed:    atk.SideEffects[4],
		Accuracy: atk.SideEffects[5],
	}
	e.ActivePet.FightStatus_ = atk.Status

	for _, s1 := range atk.SkillList {
		for idx, s2 := range e.ActivePet.SkillList_ {
			if s1.ID != s2.ID {
				continue
			}

			e.ActivePet.SkillList_[idx].PP = s1.PP
		}
	}

	// 更新技能结算
	e.LastResult = SkillResult{
		Skill:        schema.PetSkillInfo{ID: atk.SkillID}.Info(),
		AskTimes:     atk.AskTimes,
		IsCrit:       atk.IsCrit == 1,
		Damage:       atk.LostHP,
		GainDamage:   atk.GainHP,
		RealDamage:   atk.RealHurtHP,
		ZhuijiDamage: atk.ZhuijiHurt,
	}
}

func (e *FightEnemy) PetChange(change schema.ChangePet) {
	for _, pet := range e.PetBag_ {
		if pet.CatchTime != change.CatchTime {
			continue
		}

		e.ActivePet = &FightActivePet{pet, FightLevel{}, []uint8{}, []schema.PetSkillInfo{}, nil, nil}
		for _, s := range change.SkillList {
			skill := schema.PetSkillInfo{
				ID: s.ID,
				PP: s.PP,
			}

			e.ActivePet.SkillList_ = append(e.ActivePet.SkillList_, skill)
		}
	}
}

func (e *FightEnemy) IsAllDead() bool {
	for _, pet := range e.PetBag_ {
		if pet.HP > 0 {
			return false
		}
	}
	return true
}

func (e *FightEnemy) CurrentPetIndex() int {
	for idx, pet := range e.PetBag_ {
		if pet.CatchTime == e.ActivePet.CatchTime {
			return idx
		}
	}

	return -1
}

func newEnemy(user schema.FightUser) *FightEnemy {
	enemy := FightEnemy{}
	for _, info := range user.FightPets {
		pet := FightBagPet{
			CatchTime: info.CatchTime,
			ID:        info.ID,
			HP:        info.HP,
			MaxHP:     info.HP,
		}

		if client := seerapi.Default(); client != nil {
			pet.Name = client.PetName(pet.ID)
		}

		enemy.PetBag_ = append(enemy.PetBag_, pet)
	}

	enemy.ActivePet = &FightActivePet{(enemy.PetBag_[0]), FightLevel{}, []uint8{}, []schema.PetSkillInfo{}, nil, nil}
	for _, skill_id := range user.FightPets[0].Skills {
		skill := schema.PetSkillInfo{
			ID: skill_id,
		}

		skill.PP = skill.Info().MaxPP
		enemy.ActivePet.SkillList_ = append(enemy.ActivePet.SkillList_, skill)
	}

	return &enemy
}

func newPlayer(c protocol.SeerConn, user schema.FightUser, bag_pets []schema.PetInfo) *FightPlayer {
	clothes := []uint32{}
	for _, v := range user.Clothes {
		clothes = append(clothes, v.ID)
	}

	player := FightPlayer{c: c, UserID: user.UserID, Name: user.NickName, Clothes_: clothes, actionDone: make(chan struct{}, 1)}

	for _, info := range bag_pets {
		pet := FightBagPet{
			CatchTime: info.CatchTime,
			ID:        info.ID,
			HP:        info.PveCurHP,
			MaxHP:     info.PetValueInfos[0].PveCur,
		}

		if client := seerapi.Default(); client != nil {
			pet.Name = client.PetName(pet.ID)
		}

		player.PetBag_ = append(player.PetBag_, pet)
	}

	player.ActivePet = &FightActivePet{(player.PetBag_[0]), FightLevel{}, []uint8{}, []schema.PetSkillInfo{}, nil, nil}
	for _, info := range bag_pets[0].SkillInfos {
		if info.ID != 0 {
			skill := schema.PetSkillInfo{
				ID: info.ID,
				PP: info.PP,
			}

			player.ActivePet.SkillList_ = append(player.ActivePet.SkillList_, skill)
		}
	}

	return &player
}
