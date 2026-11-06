package petbag

import (
	"fmt"
	"sort"
	"suffocat/internal/common"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// 精灵背包
type PetBag struct {
	l    *lua.LState
	pets []*BagPet
}

// 清空出战背包的精灵
func (pb *PetBag) ClearBattleBag() int {
	return pb.clearBag(TYPE_BATTLE_BAG)
}

// 清空待命背包的精灵
func (pb *PetBag) ClearSupportBag() int {
	return pb.clearBag(TYPE_SUPPORT_BAG)
}

// 清空背包的精灵
func (pb *PetBag) ClearBag() int {
	return pb.ClearBattleBag() + pb.ClearSupportBag()
}

// FindByID 在背包中查找指定精灵。
func (pb *PetBag) FindByID(id uint32) *BagPet {
	for _, pet := range pb.pets {
		if pet.ID == id {
			return pet
		}
	}
	return nil
}

// 返回出战背包的精灵
func (pb *PetBag) InBattleBag() lua.LValue {
	return common.ToLuaList(pb.l, pb.GetBag(TYPE_BATTLE_BAG))
}

// 返回待命背包的精灵
func (pb *PetBag) InSupportBag() lua.LValue {
	return common.ToLuaList(pb.l, pb.GetBag(TYPE_SUPPORT_BAG))
}

// 遍历精灵背包
func (pb *PetBag) ForEach(fn *lua.LFunction) lua.LValue {
	return bagPetsForEach(pb.pets, pb.l, fn)
}

// 排序精灵背包
func (pb *PetBag) Sort(fn *lua.LFunction) {
	for _, pet := range pb.pets {
		if err := pb.l.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, luar.New(pb.l, pet)); err != nil {
			panic(err)
		}

		ret, isNumber := pb.l.Get(-1).(lua.LNumber)
		if !isNumber {
			panic("Sort传参函数的返回值不为number")
		}

		if int(ret) == -1 {
			continue
		}

		pet2 := pb.getPetByIdx(uint32(ret))
		pet.Swap(pet2)
	}

	sort.Slice(pb.pets, func(i, j int) bool {
		return pb.pets[i].Position.Index < pb.pets[j].Position.Index
	})
}

// 排序精灵背包
func (pb *PetBag) String() string {
	battlePets := ""
	supportPets := ""

	for _, v := range pb.GetBag(TYPE_BATTLE_BAG) {
		battlePets = battlePets + " " + fmt.Sprintf("%s(%d)", v.Name, v.ID)
	}

	for _, v := range pb.GetBag(TYPE_SUPPORT_BAG) {
		supportPets = supportPets + " " + fmt.Sprintf("%s(%d)", v.Name, v.ID)
	}

	if supportPets == "" {
		return fmt.Sprintf("出战背包: %s", battlePets)
	} else {
		return fmt.Sprintf("出战背包: %s\n待命背包: %s", battlePets, supportPets)
	}
}

// ----------------------- 以下是私有方法 -----------------------

func (pb *PetBag) clearBag(bag_type int) int {
	num := 0
	for _, pet := range pb.pets {
		if pet.Position.Type == bag_type && pet.GotoStorage() {
			num++
		}
	}

	return num
}

func (pb *PetBag) GetBag(bag_type int) []*BagPet {
	pets := []*BagPet{}
	for _, pet := range pb.pets {
		if pet.Position.Type == bag_type {
			pets = append(pets, pet)
		}
	}

	return pets
}

func (pb *PetBag) getPetByIdx(idx uint32) *BagPet {
	if idx < 1 || idx > 12 {
		panic(fmt.Sprintf("错误的背包位置: %d", idx))
	}

	for _, pet := range pb.pets {
		if pet.Position.Index == idx {
			return pet
		}
	}

	if idx < 7 {
		panic(fmt.Sprintf("当前出战背包仅有%d只精灵，无法在%d号位上放置精灵", len(pb.GetBag(TYPE_BATTLE_BAG)), idx))
	} else {
		panic(fmt.Sprintf("当前待命背包仅有%d只精灵，无法在%d号位上放置精灵", len(pb.GetBag(TYPE_SUPPORT_BAG)), idx))
	}
}

func NewPetBag(c protocol.SeerConn, l *lua.LState) *PetBag {
	bag := &PetBag{pets: []*BagPet{}, l: l}
	body := c.SendAndRecv(command.GET_PET_INFO_BY_ONCE, nil).Wait().Body

	// 获取出战背包
	for idx := range body.ReadU32() {
		info := schema.NewPetInfo(body)
		pet := &BagPet{
			l:         l,
			c:         c,
			ID:        info.ID,
			Name:      getPetName(info.ID),
			CatchTime: info.CatchTime,
			Position: PetPosition{
				Type:  TYPE_BATTLE_BAG,
				Name:  posName(1),
				Index: idx + 1,
			},
		}

		bag.pets = append(bag.pets, pet)
	}

	// 标注首发精灵
	if len(bag.pets) > 0 {
		bag.pets[0].Position.Name = posName(2)
	}

	// 获取待命背包
	for idx := range body.ReadU32() {
		info := schema.NewPetInfo(body)
		pet := &BagPet{
			c:         c,
			ID:        info.ID,
			Name:      getPetName(info.ID),
			CatchTime: info.CatchTime,
			Position: PetPosition{
				Type:  TYPE_SUPPORT_BAG,
				Name:  posName(7),
				Index: idx + 7,
			},
		}

		bag.pets = append(bag.pets, pet)
	}

	return bag
}

// 切换精灵背包
func SwitchPetBag(c protocol.SeerConn, idx uint32) {
	c.SendAndRecv(41922, func(bs *common.ByteStream) {
		bs.WriteU32(2).WriteU32(idx)
	}).Wait()
	c.LogFn("DEBUG", fmt.Sprintf("已成功切换至 %d 号阵容背包", idx))
}
