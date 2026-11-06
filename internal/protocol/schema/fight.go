package schema

import (
	"strings"
	"suffocat/internal/common"
)

type FightEffect struct {
	ItemID    uint32
	Status    uint8
	LeftCount uint8
	EffectID  uint16
	Args      [8]uint16
}

// 待命精灵
type SupportPet struct {
	ID     uint32
	SkinID uint32
}

type FightCloth struct {
	ID   uint32
	Type uint32
}

// 出战精灵
type FightPet struct {
	CatchTime uint32
	ID        uint32
	HP        uint32
	SkillNum  uint32
	Skills    []uint32

	EffectCount uint16
	EffectList  []FightEffect
	SkinID      uint32
}

type FightUser struct {
	UserID   uint32
	NickName string
	TopLevel uint32

	SupportCount uint32
	SupportPets  []SupportPet
	FightCount   uint32
	FightPets    []FightPet

	ClothNum uint32
	Clothes  []FightCloth
	Reserve  uint32
}

type ReadyToFightInfo struct {
	Mode      uint32
	FightType uint32
	Player    FightUser
	Enemy     FightUser
}

type ChangePet struct {
	UserID    uint32
	ID        uint32
	CatchTime uint32
	Level     uint32
	HP        uint32
	MaxHP     uint32
	SkillList []PetSkillInfo
}

func NewChangePet(buf *common.ByteStream) ChangePet {
	pet := ChangePet{}
	pet.UserID = buf.ReadU32()
	pet.ID = buf.ReadU32()
	pet.CatchTime = buf.ReadU32()
	buf.ReadBytes(16)
	pet.Level = buf.ReadU32()
	pet.HP = buf.ReadU32()
	pet.MaxHP = buf.ReadU32()

	times := buf.ReadU32()
	for range times {
		pet.SkillList = append(pet.SkillList, PetSkillInfo{buf.ReadU32(), buf.ReadU32()})
	}

	return pet
}

func newFightEffect(buf *common.ByteStream) FightEffect {
	eff := FightEffect{}
	eff.ItemID = buf.ReadU32()
	eff.Status = buf.ReadU8()
	eff.LeftCount = buf.ReadU8()
	eff.EffectID = buf.ReadU16()

	for i := range 8 {
		low := uint16(buf.ReadU8())
		high := uint16(buf.ReadU8())

		eff.Args[i] = low + high*256
	}

	return eff
}

func newFightPet(buf *common.ByteStream) FightPet {
	pet := FightPet{}
	pet.CatchTime = buf.ReadU32()
	pet.ID = buf.ReadU32()
	pet.HP = buf.ReadU32()

	pet.SkillNum = buf.ReadU32()
	for range pet.SkillNum {
		pet.Skills = append(pet.Skills, buf.ReadU32())
	}

	pet.EffectCount = buf.ReadU16()
	for range pet.EffectCount {
		pet.EffectList = append(pet.EffectList, newFightEffect(buf))
	}

	pet.SkinID = buf.ReadU32()
	return pet
}

func newFightUser(buf *common.ByteStream) FightUser {
	info := FightUser{}
	info.UserID = buf.ReadU32()
	info.NickName = strings.ReplaceAll(string(buf.ReadBytes(16)), "\u0000", "")
	info.TopLevel = buf.ReadU32()

	info.SupportCount = buf.ReadU32()
	for range info.SupportCount {
		info.SupportPets = append(info.SupportPets, SupportPet{buf.ReadU32(), buf.ReadU32()})
	}

	info.FightCount = buf.ReadU32()
	for range info.FightCount {
		if buf.ShowU32() > 100000 {
			info.FightPets = append(info.FightPets, newFightPet(buf))
			continue
		}

		pet := NewPetInfo(buf, false, false)
		fight_pet := FightPet{
			CatchTime:   pet.CatchTime,
			ID:          pet.ID,
			HP:          pet.HP,
			EffectCount: uint16(pet.EffectCount),
			SkinID:      pet.SkinID,
		}

		for _, v := range pet.SkillInfos {
			if v.ID != 0 {
				fight_pet.SkillNum++
				fight_pet.Skills = append(fight_pet.Skills, v.ID)
			}
		}

		info.FightPets = append(info.FightPets, fight_pet)

	}

	info.ClothNum = buf.ReadU32()
	for range info.ClothNum {
		info.Clothes = append(info.Clothes, FightCloth{buf.ReadU32(), buf.ReadU32()})
	}

	info.Reserve = buf.ReadU32()
	return info
}

func NewReadyToFightInfo(buf *common.ByteStream) ReadyToFightInfo {
	return ReadyToFightInfo{
		Mode:      buf.ReadU32(),
		FightType: buf.ReadU32(),
		Player:    newFightUser(buf),
		Enemy:     newFightUser(buf),
	}
}

type AttackValue struct {
	UserID uint32 // 用户账号

	SkillID uint32 // 技能ID
	Round   uint32 // 回合

	EffectName string

	AskTimes uint32 // 技能次数
	LostHP   uint32 // 我方红伤

	GainHP uint32 // 技能吸血

	RemainHP uint32 // 剩余血量
	MaxHP    uint32 // 我方最大生命

	IsCrit uint32 // 是否暴击

	SkillList []PetSkillInfo // 技能组

	Status []uint8 // 异常状态回合数

	State     uint32 // 未知
	PetStatus uint32 // 未知

	SideEffects [6]int // 能力强化

	BattleLV uint32 // 能力等级变化总数

	ChangeBitset uint32
	Priority     uint32

	ImmunizationStates []int

	SpecailArr []uint32

	IsSecondFight bool

	ChangeValue uint32

	ChangeHps []ChangeHps

	RequireSwitchCthTime uint32

	ChangeValue2 uint32

	MaxHPSelf  uint32
	MaxHPOther uint32

	SecretLaw uint32

	RealHurtHP uint32

	MoveNum uint32

	SkillRunawayMarks []uint32

	SiteBuffInfo     SiteBuffInfo
	BothSiteBuffInfo SiteBuffInfo

	MarkBuffInfo MarkBuffInfo

	SignInfoList []FightSignInfo

	LockedSkillArr [5]uint32
	SkillResultArr []uint32
	ZhuijiID       uint32
	ZhuijiHurt     uint32
}

func (atk AttackValue) TotalDamage() uint32 {
	return atk.LostHP + atk.RealHurtHP + atk.ZhuijiHurt
}

type UseSkillInfo struct {
	SelfAttack  AttackValue
	EnemyAttack AttackValue
}

type MarkBuffArr struct {
	ID      uint32
	MarkNum uint32
}

type MarkBuffInfo struct {
	MarkBuffCnt uint8
	MarkBuffArr []MarkBuffArr
}

type SiteBuffInfo struct {
	SiteBuffID   uint32
	SiteBuffTurn uint32
}

type ChangeHps struct {
	CatchTime     uint32
	HP            uint32
	MaxHP         uint32
	Lock          uint32
	ChujueNumber  uint32
	ChujueRound   uint32
	PetBagMarkArr MarkBuffInfo
}

type FightSignInfo struct {
	ID       int
	LvNum    int
	RoundNum int
	SpValue  int
}

type PetStatusEffectInfo struct {
	Type   uint32
	ID     uint32
	Parm   uint32
	Name   string
	Des    string
	IsShow bool
}

func subByte(a, b, c uint32) uint32 {
	if b+c > 32 {
		return 0
	}

	if c > 16 {
		i := subByte(a, b, 16)
		return i + (subByte(a-i, b+16, c-16) << 16)
	}

	return (65535 >> (16 - c)) & (a >> b)
}

func newAttackValue(buf *common.ByteStream) AttackValue {
	atk := AttackValue{}

	atk.UserID = buf.ReadU32()
	atk.SkillID = buf.ReadU32()
	atk.Round = buf.ReadU32()

	buf.ReadU32()
	buf.ReadU32()

	atk.AskTimes = buf.ReadU32()
	atk.LostHP = buf.ReadU32()
	atk.RealHurtHP = buf.ReadU32()
	atk.GainHP = buf.ReadU32()
	atk.RemainHP = buf.ReadU32()
	atk.MaxHP = buf.ReadU32()

	atk.State = buf.ReadU32()
	atk.PetStatus = buf.ReadU32()

	times := buf.ReadU32()
	for range times {
		atk.SkillList = append(atk.SkillList, PetSkillInfo{
			ID: buf.ReadU32(),
			PP: buf.ReadU32(),
		})
	}

	atk.IsCrit = buf.ReadU32()

	times2 := buf.ReadU8()
	for range times2 {
		atk.Status = append(atk.Status, buf.ReadU8())
	}

	times = buf.ReadU32()
	for range times {
		atk.SpecailArr = append(atk.SpecailArr, buf.ReadU32())
	}

	times = buf.ReadU32()
	for range times {
		status := PetStatusEffectInfo{
			Type: buf.ReadU32(),
			ID:   buf.ReadU32(),
			Parm: buf.ReadU32(),
		}

		if status.Type == 1 {
			atk.SideEffects[status.ID-1] = int(status.Parm) - 6
		}
	}

	atk.BattleLV = buf.ReadU32()
	atk.ChangeBitset = buf.ReadU32()
	atk.Priority = buf.ReadU32()

	times = buf.ReadU32()
	for i := uint32(0); i < times; i++ {
		atk.ImmunizationStates = append(atk.ImmunizationStates, int(buf.ReadU32()))
	}

	times = buf.ReadU32()
	for i := uint32(0); i < times; i++ {
		ch := ChangeHps{
			CatchTime:    buf.ReadU32(),
			HP:           buf.ReadU32(),
			MaxHP:        buf.ReadU32(),
			Lock:         buf.ReadU32(),
			ChujueNumber: buf.ReadU32(),
			ChujueRound:  buf.ReadU32(),
		}

		ch.PetBagMarkArr.MarkBuffCnt = buf.ReadU8()
		for j := uint8(0); j < ch.PetBagMarkArr.MarkBuffCnt; j++ {
			ch.PetBagMarkArr.MarkBuffArr = append(ch.PetBagMarkArr.MarkBuffArr, MarkBuffArr{
				ID:      uint32(buf.ReadU16()),
				MarkNum: uint32(buf.ReadU8()),
			})
		}

		atk.ChangeHps = append(atk.ChangeHps, ch)
	}

	atk.RequireSwitchCthTime = buf.ReadU32()
	atk.MaxHPSelf = buf.ReadU32()
	atk.MaxHPOther = buf.ReadU32()
	atk.SecretLaw = buf.ReadU32()
	atk.MoveNum = buf.ReadU32()

	for i := uint32(0); i < atk.MoveNum; i++ {
		atk.SkillRunawayMarks = append(atk.SkillRunawayMarks, buf.ReadU32())
	}

	atk.SiteBuffInfo = SiteBuffInfo{
		SiteBuffID:   uint32(buf.ReadU16()),
		SiteBuffTurn: uint32(buf.ReadU8()),
	}

	atk.BothSiteBuffInfo = SiteBuffInfo{
		SiteBuffID:   uint32(buf.ReadU16()),
		SiteBuffTurn: uint32(buf.ReadU8()),
	}

	atk.MarkBuffInfo.MarkBuffCnt = buf.ReadU8()
	for i := uint8(0); i < atk.MarkBuffInfo.MarkBuffCnt; i++ {
		atk.MarkBuffInfo.MarkBuffArr = append(atk.MarkBuffInfo.MarkBuffArr, MarkBuffArr{
			ID:      uint32(buf.ReadU16()),
			MarkNum: uint32(buf.ReadU8()),
		})
	}

	times = buf.ReadU32()
	for i := uint32(0); i < times; i++ {
		a := buf.ReadU32()

		atk.SignInfoList = append(atk.SignInfoList, FightSignInfo{
			SpValue:  int(buf.ReadU32()),
			ID:       int(subByte(a, 0, 16)),
			LvNum:    int(subByte(a, 16, 8)),
			RoundNum: int(subByte(a, 24, 8)),
		})
	}

	for i := range 5 {
		atk.LockedSkillArr[i] = buf.ReadU32()
	}

	resultCount := buf.ReadU32()

	for range resultCount {
		atk.SkillResultArr = append(atk.SkillResultArr, buf.ReadU32())
	}

	atk.ZhuijiID = buf.ReadU32()
	atk.ZhuijiHurt = buf.ReadU32()

	return atk
}

func NewUseSkillInfo(buf *common.ByteStream) (AttackValue, AttackValue) {
	frist := newAttackValue(buf)
	second := newAttackValue(buf)
	return frist, second
}
