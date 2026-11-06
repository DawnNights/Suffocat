package fight

import (
	"strings"
	"suffocat/internal/common"
	"suffocat/internal/protocol/command"
)

func (p *FightPlayer) UseSkill(skillID uint32) {
	if p.IsAction {
		return
	}

	for _, skill := range p.ActivePet.SkillList_ {
		if skill.PP == 0 && skill.ID == skillID {
			p.UseItem(300017)
			return
		}
	}

	p.c.SendFn(command.USE_SKILL, func(bs *common.ByteStream) {
		bs.WriteU32(skillID)
	})
	p.IsAction = true
}

func (p *FightPlayer) UseSkillByIdx(skillIdx int) {
	if p.IsAction {
		return
	}

	maxIdx := len(p.ActivePet.SkillList_)
	if skillIdx > maxIdx {
		skillIdx = maxIdx
	} else if skillIdx <= 0 {
		skillIdx = 1
	}

	p.UseSkill(p.ActivePet.SkillList_[skillIdx-1].ID)
}

func (p *FightPlayer) UseSkillByName(skillName string) {
	if p.IsAction {
		return
	}

	for _, skill := range p.ActivePet.SkillList_ {
		if strings.Contains(skill.Info().Name, skillName) {
			p.UseSkill(skill.ID)
		}
	}
}

func (p *FightPlayer) UseSkip() {
	p.UseSkill(0)
}

func (p *FightPlayer) UseItem(itemID uint32) {
	if p.IsAction {
		return
	}

	p.c.SendFn(command.USE_PET_ITEM, func(bs *common.ByteStream) {
		bs.WriteU32List(p.ActivePet.CatchTime, itemID, 0)
	})
	p.IsAction = true
}

func (p *FightPlayer) SwitchPet(pet FightBagPet) {
	if p.IsAction {
		return
	}

	// 若当前精灵阵亡则算死切
	if p.ActivePet.MaxHP != 0 {
		p.IsAction = true
	}

	p.c.SendAndRecv(command.CHANGE_PET, func(bs *common.ByteStream) {
		bs.WriteU32(pet.CatchTime)
	}).Wait()
}

func (p *FightPlayer) SwitchPetByID(petID uint32) {
	for _, pet := range p.PetBag_ {
		if pet.HP > 0 && pet.ID == petID {
			p.SwitchPet(pet)
		}
	}
}

func (p *FightPlayer) SwitchPetByName(petName string) {
	for _, pet := range p.PetBag_ {
		if pet.HP > 0 && strings.Contains(pet.Name, petName) {
			p.SwitchPet(pet)
		}
	}
}

func (p *FightPlayer) SwitchNextPet() {
	for _, pet := range p.PetBag_ {
		if pet.HP > 0 && pet.CatchTime != p.ActivePet.CatchTime {
			p.SwitchPet(pet)
			return
		}
	}
}

func (p *FightPlayer) Escape() {
	if p.IsAction {
		return
	}

	p.c.SendFn(command.ESCAPE_FIGHT, nil)
	p.IsAction = true
}
