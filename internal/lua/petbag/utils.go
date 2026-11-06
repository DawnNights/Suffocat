package petbag

import (
	"suffocat/internal/common"
	"suffocat/internal/seerapi"
	"sync"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var (
	petNameMap   = map[uint32]string{}
	petNameMapMu sync.RWMutex
)

func getPetName(pet_id uint32) string {
	petNameMapMu.RLock()
	name, ok := petNameMap[pet_id]
	petNameMapMu.RUnlock()
	if ok {
		return name
	}

	name = ""
	if client := seerapi.Default(); client != nil {
		name = client.PetName(pet_id)
	}

	// 只缓存成功结果，避免 SeerAPI 暂时不可用时把空名字永久缓存
	if name != "" {
		petNameMapMu.Lock()
		petNameMap[pet_id] = name
		petNameMapMu.Unlock()
	}
	return name
}

func bagPetsForEach(list []*BagPet, l *lua.LState, fn *lua.LFunction) lua.LValue {
	pets := []*BagPet{}
	for _, pet := range list {
		if err := l.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, luar.New(l, pet)); err != nil {
			panic(err)
		}

		ret, isBool := l.Get(-1).(lua.LBool)
		if !isBool {
			panic("ForEach传参函数的返回值不为boolen")
		}

		if bool(ret) {
			pets = append(pets, pet)
		}
		l.Pop(1)
	}

	return common.ToLuaList(l, pets)
}

type FindPet struct {
	ID    uint32
	Level uint32
}

func (ps *PetStorage) SetPetBag(bag_type int, petList []FindPet) []FindPet {
	found := false
	notFound := []FindPet{}

	for _, findpet := range petList[:min(6, len(petList))] {
		for _, pet := range ps.pets {
			if pet.ID != findpet.ID || pet.Position.Type != TYPE_STORAGE {
				continue
			}

			if findpet.Level > 0 && pet.PetInfo().Level != findpet.Level {
				continue
			}

			found = pet.GotoBag(bag_type)
			break
		}

		if !found {
			notFound = append(notFound, findpet)
		}
		found = false
	}

	return notFound
}
