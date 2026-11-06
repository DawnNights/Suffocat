package schema

import "suffocat/internal/common"

type CountmarkInfo struct {
	// 刻印ID
	ID uint32
	// 绑定精灵获取时间
	CatchTime uint32
	// 是否绑定
	HoleType uint32
	// 刻印获取时间
	ObtainTime uint32
	// 绑定精灵ID
	BindMonID uint32
	// 绑定技能ID
	BindMoveID uint32
	// 攻击
	AtkAdj uint32
	// 防御
	DefAdj uint32
	// 特攻
	SpAtkAdj uint32
	// 特防
	SpDefAdj uint32
	// 速度
	SpdAdj uint32
	// 体力
	HpAdj uint32
	// 绑定宝石ID
	GemID uint32
	// 刻印索引位置
	PkgPos uint32
}

func NewCountmarkInfo(buf *common.ByteStream) CountmarkInfo {
	return CountmarkInfo{
		ID:         buf.ReadU32(),
		CatchTime:  buf.ReadU32(),
		HoleType:   buf.ReadU32(),
		ObtainTime: buf.ReadU32(),
		BindMonID:  buf.ReadU32(),
		BindMoveID: buf.ReadU32(),
		AtkAdj:     buf.ReadU32(),
		DefAdj:     buf.ReadU32(),
		SpAtkAdj:   buf.ReadU32(),
		SpDefAdj:   buf.ReadU32(),
		SpdAdj:     buf.ReadU32(),
		HpAdj:      buf.ReadU32(),
		GemID:      buf.ReadU32(),
		PkgPos:     buf.ReadU32(),
	}
}
