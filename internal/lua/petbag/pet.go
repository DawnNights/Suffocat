package petbag

import (
	"fmt"
	"suffocat/internal/common"
	"suffocat/internal/lua/mintmark"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"
	"suffocat/internal/seerapi"

	lua "github.com/yuin/gopher-lua"
)

// 背包精灵
type BagPet struct {
	c         protocol.SeerConn
	l         *lua.LState
	ID        uint32      // 精灵ID
	Name      string      // 精灵名称
	CatchTime uint32      // 精灵捕获时间
	Position  PetPosition // 精灵位置
}

// PetInfoRaw 获取精灵详细信息，不构建 Lua 表。
func (p *BagPet) PetInfoRaw() schema.PetInfo {
	body := p.c.SendAndRecv(command.GET_PET_INFO, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime)
	}).Wait().Body
	return schema.NewPetInfo(body)
}

// 获取精灵的详细信息
func (p *BagPet) PetInfo() PetInfo {
	body := p.c.SendAndRecv(command.GET_PET_INFO, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime)
	}).Wait().Body
	info := schema.NewPetInfo(body)

	lua_info := PetInfo{
		PetInfo:       info,
		SkillInfos:    common.ToLuaList(p.l, info.SkillInfos[:]),
		EffectList:    common.ToLuaList(p.l, luaEffectList(p.l, info.EffectList)),
		PetValueInfos: common.ToLuaList(p.l, info.PetValueInfos[:]),
	}
	return lua_info
}

// 移至战斗背包
func (p *BagPet) GotoBattleBag() bool {
	return p.GotoBag(TYPE_BATTLE_BAG)
}

// 移至待命背包
func (p *BagPet) GotoSupportBag() bool {
	return p.GotoBag(TYPE_SUPPORT_BAG)
}

// 移至仓库
func (p *BagPet) GotoStorage() bool {
	return p.GotoBag(TYPE_STORAGE)
}

// 与其它精灵交换位置
func (p *BagPet) Swap(pet *BagPet) bool {
	if !p.checkSwap(pet.Position) {
		p.c.LogFn("WARN", fmt.Sprintf("%s[%s] 无法与 %s[%s] 交换位置", p.Name, p.Position.Name, pet.Name, pet.Position.Name))
		return false
	}

	switch p.Position.Type {
	case TYPE_BATTLE_BAG, TYPE_SUPPORT_BAG:
		if pet.Position.Type == TYPE_STORAGE {
			return bagSwapStorage(p, pet)
		} else {
			return bagSwapBag(p, pet)
		}
	case TYPE_STORAGE:
		return pet.Swap(p)
	}

	return false
}

// 使用指定ID道具并返回使用结果
func (p *BagPet) UseItem(item_id uint32) bool {
	body := p.c.SendAndRecv(command.USE_PET_ITEM_OUT_OF_FIGHT, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime).WriteU32(item_id)
	}).Wait().Body

	return body.Length() != 0
}

// 装备指定孔位刻印
func (p *BagPet) EquipMark(idx uint32, mark mintmark.MintMark) bool {
	if mark.BindMonID != p.ID {
		if mark.BindMonID != 0 {
			p.c.LogFn("WARN", fmt.Sprintf("刻印[%s]已绑定精灵[%s]", mark.Name, getPetName(mark.BindMonID)))
			return false
		}

		if mark.Class != 17 && mark.Class != 21 && mark.Class != 24 {
			p.c.LogFn("WARN", fmt.Sprintf("刻印[%s]禁止使用，未绑定刻印仅允许使用泰坦、衡、偃月系列", mark.Name))
			return false
		}
	}

	p.c.SendAndRecv(command.COUNTERMARK_BANDING, func(bs *common.ByteStream) {
		bs.WriteU32(mark.ObtainTime).WriteU32(p.CatchTime)
		bs.WriteU32(p.ID).WriteU32(mark.BindMoveID)
	})

	return !p.c.SendAndRecv(command.USE_COUNTERMARK, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime).WriteU32(idx - 1)
		bs.WriteU32(mark.ID).WriteU32(mark.ObtainTime)
	}).Wait().Body.IsEmpty()
}

// 卸下指定孔位刻印
func (p *BagPet) UnEquipMark(idx uint32) {
	p.c.SendAndRecv(command.USE_COUNTERMARK, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime).WriteU32(idx - 1).WriteU32(0).WriteU32(0)
	})
}

// 设置指定号位技能
func (p *BagPet) SetSkill(idx uint32, skill_id uint32) bool {
	info := p.PetInfo()
	old_skill := info.GetSkillInfo(idx).ID

	if old_skill == skill_id {
		return true
	}
	if old_skill == 0 {
		return false
	}

	// 判断是否为技能石
	if skill_id > 100000 {
		for i := range uint32(5) {
			if info.GetSkillInfo(i+1).ID > 100000 {
				return p.setSkillStone(0, skill_id)
			}
		}

		return p.setSkillStone(old_skill, skill_id)
	}

	// 判断该精灵是否拥有该技能
	if client := seerapi.Default(); client != nil && !client.PetHasMove(p.ID, skill_id) {
		p.c.LogFn("ERROR", fmt.Sprintf("精灵 %s 不存在ID为 %d 的技能", p.Name, skill_id))
		return false
	}

	if idx >= 5 {
		return p.setFifthSkill(old_skill, skill_id)
	}

	return p.setNormalSkill(old_skill, skill_id)
}

// ----------------------- 以下是私有方法 -----------------------

func (p *BagPet) GotoBag(bag_type int) bool {
	if p.Position.Type == bag_type {
		return true
	}

	id := parsePosID(p.Position.Type, bag_type)
	if id == -1 {
		p.c.LogFn("WARN", fmt.Sprintf("%s处于[%s]中, 无法前往[%s]", p.Name, p.Position.Name, typeName(bag_type)))
		return false
	}

	body := p.c.SendAndRecv(command.PET_RELEASE, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime).WriteU32(uint32(id))
	}).Wait().Body

	if body.IsEmpty() {
		p.c.LogFn("WARN", fmt.Sprintf("%s前往[%s]失败", p.Name, typeName(bag_type)))
		return false
	}

	p.Position.Type = bag_type
	p.Position.Name = typeName(bag_type)
	return true
}

func (p *BagPet) checkSwap(pos PetPosition) bool {
	if p.Position.Type == TYPE_OTHER || pos.Type == TYPE_OTHER {
		return false
	}

	if p.Position.Type == TYPE_STORAGE && pos.Type == TYPE_STORAGE {
		return false
	}

	return true
}

func (p *BagPet) setNormalSkill(old_skill, skill_id uint32) bool {
	return !p.c.SendAndRecv(command.PET_SKILL_SWICTH, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime)
		bs.WriteU32(1).WriteU32(1)
		bs.WriteU32(old_skill).WriteU32(skill_id)
	}).Wait().Body.IsEmpty()
}

func (p *BagPet) setFifthSkill(old_skill, skill_id uint32) bool {
	return !p.c.SendAndRecv(41831, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime)
		bs.WriteU32(old_skill).WriteU32(skill_id)
	}).Wait().Body.IsEmpty()
}

func (p *BagPet) setSkillStone(old_skill, stone_id uint32) bool {
	var skill_type uint32 = 1

	if stone_id > 200000 {
		skill_type = 2
		stone_id = stone_id - 100000
	}
	stone_id = stone_id + 1000000

	return !p.c.SendAndRecv(command.SKILL_STONE_USE_ITEM, func(bs *common.ByteStream) {
		bs.WriteU32(p.CatchTime).WriteU32(stone_id)
		bs.WriteU32(old_skill).WriteU32(skill_type)
	}).Wait().Body.IsEmpty()
}

func bagSwapStorage(bagPet, storagePet *BagPet) bool {
	ok := bagPet.c.SendAndRecv(41467, func(bs *common.ByteStream) {
		bs.WriteU32(bagPet.Position.Index)
		bs.WriteU32(bagPet.CatchTime)
		bs.WriteU32(storagePet.CatchTime)

	}).Wait().Body.Length() != 0

	if !ok {
		bagPet.c.LogFn("WARN", fmt.Sprintf("%s[%s(%d)] 与 %s[%s] 交换位置失败", bagPet.Name, bagPet.Position.Name, bagPet.Position.Index, storagePet.Name, storagePet.Position.Name))
		return false
	}

	bagPet.Position, storagePet.Position = storagePet.Position, bagPet.Position
	return true
}

func bagSwapBag(petA, petB *BagPet) bool {
	if petA.Position.Index == petB.Position.Index {
		return true
	}

	if petA.Position.Name == "出战首发" {
		return fristSwapBag(petA, petB)
	}

	if petB.Position.Name == "出战首发" {
		return fristSwapBag(petB, petA)
	}

	body := petA.c.SendAndRecv(41462, func(bs *common.ByteStream) {
		bs.WriteU32(petA.Position.Index).WriteU32(petA.CatchTime)
		bs.WriteU32(petB.Position.Index).WriteU32(petB.CatchTime)
	}).Wait().Body

	if body.IsEmpty() {
		return false
	}

	petA.Position, petB.Position = petB.Position, petA.Position
	return true
}

func fristSwapBag(fristPet, bagPet *BagPet) bool {
	bagPet.c.SendAndRecv(command.PET_DEFAULT, func(bs *common.ByteStream) {
		bs.WriteU32(bagPet.CatchTime)
	}).Wait()

	fristPet.Position, bagPet.Position = bagPet.Position, fristPet.Position
	return true
}
