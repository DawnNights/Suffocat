package petbag

import (
	"suffocat/internal/common"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"

	lua "github.com/yuin/gopher-lua"
)

// 精灵仓库
type PetStorage struct {
	pets []*BagPet
	l    *lua.LState
}

// FindByID 在仓库中查找指定精灵。
func (ps *PetStorage) FindByID(id uint32) *BagPet {
	for _, pet := range ps.pets {
		if pet.ID == id {
			return pet
		}
	}
	return nil
}

// 返回位于仓库的精灵
func (ps *PetStorage) InStorage() *lua.LTable {
	return common.ToLuaList(ps.l, ps.getStorage(TYPE_STORAGE))
}

// 返回其它位置的精灵
func (ps *PetStorage) InOther() *lua.LTable {
	return common.ToLuaList(ps.l, ps.getStorage(TYPE_OTHER))
}

// 遍历精灵仓库
func (ps *PetStorage) ForEach(fn *lua.LFunction) lua.LValue {
	return bagPetsForEach(ps.pets, ps.l, fn)
}

// ----------------------- 以下是私有方法 -----------------------

func (ps *PetStorage) getStorage(bag_type int) []*BagPet {
	pets := []*BagPet{}
	for _, pet := range ps.pets {
		if pet.Position.Type == bag_type {
			pets = append(pets, pet)
		}
	}

	return pets
}

func NewPetStorage(c protocol.SeerConn, l *lua.LState) *PetStorage {
	storage := &PetStorage{pets: []*BagPet{}, l: l}

	var a, b, d uint32 = 0, 999, 1
	for {
		body := c.SendAndRecv(command.GET_PET_STORAGE, func(bs *common.ByteStream) {
			bs.WriteU32(a).WriteU32(b)
		}).Wait().Body

		count := body.ReadU32()
		for range count {
			info := schema.NewStoragePet(body)
			pet := &BagPet{
				l:         l,
				c:         c,
				ID:        info.ID,
				Name:      getPetName(info.ID),
				CatchTime: info.CatchTime,
				Position: PetPosition{
					Type: TYPE_STORAGE,
					Name: posName(info.Posi),
				},
			}

			storage.pets = append(storage.pets, pet)
		}

		if count >= 1000 {
			d = d + 1
			a = d*1000 - 1000
			b = a + 999
			continue
		}
		break
	}

	return storage
}
