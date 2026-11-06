package mintmark

import (
	"suffocat/internal/common"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"
	"suffocat/internal/seerapi"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type MarkStorage struct {
	marks []MintMark
	l     *lua.LState
}

func (ms *MarkStorage) ForEach(fn *lua.LFunction) lua.LValue {
	marks := []MintMark{}
	for _, mark := range ms.marks {
		if err := ms.l.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, luar.New(ms.l, mark)); err != nil {
			panic(err)
		}

		ret, isBool := ms.l.Get(-1).(lua.LBool)
		if !isBool {
			panic("ForEach传参函数的返回值不为boolen")
		}

		if bool(ret) {
			marks = append(marks, mark)
		}
		ms.l.Pop(1)
	}

	return common.ToLuaList(ms.l, marks)
}

func NewMarkStorage(c protocol.SeerConn, l *lua.LState) *MarkStorage {
	var a uint32
	var marks []MintMark

	for {
		body := c.SendAndRecv(command.QUERY_COUNTMARK_BY_TYPE_PAGE, func(bs *common.ByteStream) {
			bs.WriteU32(1).WriteU32(1).WriteU32(a * 1000).WriteU32(1000)
		}).Wait().Body

		count := body.ReadU32() / 14
		for range count {
			m := schema.NewCountmarkInfo(body)
			mark := MintMark{
				c:          c,
				ID:         m.ID,
				ObtainTime: m.ObtainTime,
				CatchTime:  m.CatchTime,
				BindMonID:  m.BindMonID,
				BindMoveID: m.BindMoveID,
				BindGemID:  m.GemID,
				HoleType:   m.HoleType,
				Attr: MarkAttr{
					Atk:   m.AtkAdj,
					Def:   m.DefAdj,
					SpAtk: m.SpAtkAdj,
					SpDef: m.SpDefAdj,
					Spd:   m.SpdAdj,
					Hp:    m.HpAdj,
				},
			}

			info, err := getMintMarkByID(mark.ID)
			if err == nil {
				mark.Name = info.Name
				mark.Class = info.ClassID
				mark.ExtraAttr = MarkAttr{
					Atk:   uint32(info.ExtraAttr.Atk),
					Def:   uint32(info.ExtraAttr.Def),
					SpAtk: uint32(info.ExtraAttr.SpAtk),
					SpDef: uint32(info.ExtraAttr.SpDef),
					Spd:   uint32(info.ExtraAttr.Spd),
					Hp:    uint32(info.ExtraAttr.HP),
				}
			}

			marks = append(marks, mark)
		}

		if count >= 1000 {
			a++
		} else {
			return &MarkStorage{marks: marks, l: l}
		}
	}
}

func getMintMarkByID(markID uint32) (seerapi.MintMark, error) {
	client := seerapi.Default()
	if client == nil {
		return seerapi.MintMark{}, seerapi.ErrNotFound
	}
	return client.MintMark(markID)
}
