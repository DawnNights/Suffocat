package mintmark

import (
	"fmt"
	"suffocat/internal/common"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
)

type MarkAttr struct {
	// 攻击
	Atk uint32

	// 防御
	Def uint32

	// 特攻
	SpAtk uint32

	// 特防
	SpDef uint32

	// 速度
	Spd uint32

	// 体力
	Hp uint32
}

type MintMark struct {
	c protocol.SeerConn

	// 刻印ID
	ID uint32

	// 刻印获取时间
	ObtainTime uint32

	// 绑定精灵获取时间
	CatchTime uint32

	// 刻印名称
	Name string

	// 绑定精灵ID
	BindMonID uint32

	// 绑定技能ID
	BindMoveID uint32

	// 绑定宝石ID
	BindGemID uint32

	// 是否绑定
	HoleType uint32

	// 刻印属性
	Attr MarkAttr

	// 刻印额外属性
	ExtraAttr MarkAttr

	// 刻印系列
	Class uint32
}

// 刻印升级
func (m MintMark) LevelUp() bool {
	body := m.c.SendAndRecv(command.STRENGTHEN_COUNTERMARK, func(bs *common.ByteStream) {
		bs.WriteU32(m.ObtainTime)
	}).Wait().Body

	if body.IsEmpty() {
		return false
	}

	m.c.SendAndRecv(command.SAVE_COUNTERMARK_PROPERTY, func(bs *common.ByteStream) {
		bs.WriteU32(m.ObtainTime)
	}).Wait()
	return true
}

// 刻印一键满级
func (m MintMark) LevelMax() {
	for range 5 {
		if !m.LevelUp() {
			break
		}
	}
}

// 刻印分解(仅泰坦、衡、偃月系列)
func (m MintMark) Resolve() bool {
	if m.Class == 17 || m.Class == 21 || m.Class == 24 {
		return m.c.SendAndRecv(command.COUNTERMARK_RESOLVE, func(bs *common.ByteStream) {
			bs.WriteU32(m.ObtainTime)
		}).Wait().Body.Length() != 0
	}

	m.c.LogFn("WARN", fmt.Sprintf("刻印[%s]禁止分解，仅允许分解泰坦、衡、偃月系列刻印", m.Name))
	return false
}

// 重新绑定技能ID
func (m MintMark) ReBindMoveID(id uint32) MintMark {
	m.BindMoveID = id
	return m
}
