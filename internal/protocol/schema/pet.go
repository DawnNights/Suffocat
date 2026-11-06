package schema

import (
	"fmt"
	"suffocat/internal/common"
	"suffocat/internal/seerapi"
)

// --------------------------------
// 精灵信息（PetInfo）
// --------------------------------
type PetInfo struct {
	ID                uint32            // 精灵 ID
	UserID            uint32            // 玩家 ID
	Name              string            // 精灵名称
	Generation        uint32            // 未知
	DV                uint32            // 个体值
	Nature            uint32            // 性格 ID
	AbilityType       uint32            // 特性 ID
	Level             uint32            // 等级
	Exp               uint32            // 当前经验值
	LvExp             uint32            // 当前等级经验
	NextLvExp         uint32            // 下一级所需经验
	HP                uint32            // 当前生命值
	MaxHP             uint32            // 最大生命值
	Attack            uint32            // 攻击力
	Defence           uint32            // 防御力
	SA                uint32            // 特殊攻击
	SD                uint32            // 特殊防御
	Speed             uint32            // 速度
	EvHP              uint32            // 学习力：HP
	EvAttack          uint32            // 学习力：攻击
	EvDefence         uint32            // 学习力：防御
	EvSA              uint32            // 学习力：特攻
	EvSD              uint32            // 学习力：特防
	EvSP              uint32            // 学习力：速度
	Index             int               // 队伍索引（出战顺序）
	SkillInfos        [5]PetSkillInfo   // 技能列表（最多5个）
	CatchTime         uint32            // 捕获时间戳
	CatchMap          uint32            // 捕获地图 ID
	CatchRect         uint32            // 捕获位置区域
	CatchLevel        uint32            // 捕获时等级
	AbilityMark       uint32            // 第一刻印
	SkillMark         uint32            // 第二刻印
	CommonMark        uint32            // 第三刻印
	CommonMarkActived uint32            // 是否开启第三刻印孔
	EffectCount       int               // 效果数量
	EffectList        []PetEffectInfo   // 效果列表
	ResistanceInfo    PetResistanceInfo // 抗性信息
	SkinID            uint32            // 皮肤 ID
	AssistMoveID      uint32            // 辅助技能 ID
	BaseCurHP         uint32            // 当前基础 HP
	PveCurHP          uint32            // PvE 模式下当前 HP
	PvpCurHP          uint32            // PvP 模式下当前 HP
	PetValueInfos     [6]PetValueInfo   // 体力 | 攻击 | 防御 | 特攻 | 特防 | 速度
	Position          int               // 精灵所处位置
}

func (p PetInfo) GetSkillInfo(idx uint32) PetSkillInfo {
	if idx < 1 {
		idx = 1
	}

	if idx > 5 {
		idx = 5
	}

	return p.SkillInfos[idx-1]
}

// ----------------------------
// 精灵技能信息（PetSkillInfo）
// ----------------------------
type PetSkillInfo struct {
	ID uint32 // 技能 ID
	PP uint32 // 技能剩余使用次数（Power Point）
}

func (s PetSkillInfo) Info() PetSkillMoreInfo {
	if s.ID > 200000 {
		s.ID = s.ID - 100000
	}

	if s.ID > 100000 {
		s.ID = 100000 + (s.ID % 1000)
	}

	if info, exist := StoneInfoMap[s.ID]; exist {
		return info
	}

	info := PetSkillMoreInfo{ID: s.ID}
	if client := seerapi.Default(); client != nil {
		if move, err := client.Skill(s.ID); err == nil {
			info.Name = move.Name
			info.Type = move.Type
			info.Power = move.Power
			info.MaxPP = move.MaxPP
			info.Accuracy = move.Accuracy
			info.Priority = move.Priority
		}
	}

	return info
}

// 精灵技能详细信息
type PetSkillMoreInfo struct {
	ID       uint32
	Name     string
	Type     uint32
	Power    uint32
	MaxPP    uint32
	Accuracy uint32
	Priority int
}

// --------------------------------
// 精灵效果信息（PetEffectInfo）
// --------------------------------
type PetEffectInfo struct {
	ItemID    uint32 // 效果来源道具 ID
	Status    uint32 // 效果状态（启用、禁用等）
	LeftCount uint32 // 剩余回合数
	EffectID  uint32 // 效果 ID

	Args       string            // 效果附加参数
	ArgsAll    map[uint32]uint32 // 附加参数总数
	Args6      string            // 附加参数（第6个）
	AddArray   []int             // 加成数值数组
	LevelArray []int             // 等级修正数组
}

// --------------------------------
// 精灵抗性信息（PetResistanceInfo）
// --------------------------------
type PetResistanceInfo struct {
	ResistState   uint32 // 抗性状态标志位
	RedGem        uint32 // 红色宝石抗性数值
	GreenGem      uint32 // 绿色宝石抗性数值
	Reserve       uint32 // 预留字段
	ResistAll     uint32 // 总抗性
	HurtResistarr int    // 伤害减免抗性
	EffResistarr  int    // 异常状态抗性
	Cirt          uint32 // 暴击概率
	CirtAdj       uint32 // 暴击修正值
	Regular       uint32 // 命中率
	RegularAdj    uint32 // 命中修正值
	Precent       uint32 // 效果触发率
	PrecentAdj    uint32 // 效果触发修正值
	Ctl1Idx       uint32 // 控制类异常1索引
	Ctl1          uint32 // 控制类异常1数值
	Ctl1Adj       uint32 // 控制类异常1修正
	Ctl2Idx       uint32 // 控制类异常2索引
	Ctl2          uint32 // 控制类异常2数值
	Ctl2Adj       uint32 // 控制类异常2修正
	Ctl3Idx       uint32 // 控制类异常3索引
	Ctl3          uint32 // 控制类异常3数值
	Ctl3Adj       uint32 // 控制类异常3修正
	Weak1Idx      uint32 // 弱化类异常1索引
	Weak1         uint32 // 弱化类异常1数值
	Weak1Adj      uint32 // 弱化类异常1修正
	Weak2Idx      uint32 // 弱化类异常2索引
	Weak2         uint32 // 弱化类异常2数值
	Weak2Adj      uint32 // 弱化类异常2修正
	Weak3Idx      uint32 // 弱化类异常3索引
	Weak3         uint32 // 弱化类异常3数值
	Weak3Adj      uint32 // 弱化类异常3修正
}

// --------------------------------
// 精灵数值信息（PetValueInfo）
// --------------------------------
type PetValueInfo struct {
	BaseCur uint32 // 当前基础值
	PveCur  uint32 // PvE 模式当前值
	PvpCur  uint32 // PvP 模式当前值
}

// --------------------------------
// 仓库精灵信息（StoragePet）
// --------------------------------
type StoragePet struct {
	ID        uint32
	CatchTime uint32
	Posi      uint32
	Level     uint32
}

func NewPetInfo(buf *common.ByteStream, params ...bool) PetInfo {
	pet := PetInfo{}
	pet.ID = buf.ReadU32()
	buf.ReadBytes(16)

	if len(params) < 1 || params[0] {
		pet.Generation = buf.ReadU32()
	}

	pet.DV = buf.ReadU32()
	pet.Nature = buf.ReadU32()
	pet.AbilityType = buf.ReadU32()
	pet.Level = buf.ReadU32()
	pet.Exp = buf.ReadU32()
	pet.LvExp = buf.ReadU32()
	pet.NextLvExp = buf.ReadU32()
	pet.HP = buf.ReadU32()
	pet.MaxHP = buf.ReadU32()
	pet.Attack = buf.ReadU32()
	pet.Defence = buf.ReadU32()
	pet.SA = buf.ReadU32()
	pet.SD = buf.ReadU32()
	pet.Speed = buf.ReadU32()
	pet.EvHP = buf.ReadU32()
	pet.EvAttack = buf.ReadU32()
	pet.EvDefence = buf.ReadU32()
	pet.EvSA = buf.ReadU32()
	pet.EvSD = buf.ReadU32()
	pet.EvSP = buf.ReadU32()

	buf.ReadU32()
	for i := range 5 {
		pet.SkillInfos[i] = PetSkillInfo{
			ID: buf.ReadU32(),
			PP: buf.ReadU32(),
		}
	}

	pet.CatchTime = buf.ReadU32()
	pet.CatchMap = buf.ReadU32()
	pet.CatchRect = buf.ReadU32()
	pet.CatchLevel = buf.ReadU32()
	pet.AbilityMark = buf.ReadU32()
	pet.SkillMark = buf.ReadU32()
	pet.CommonMark = buf.ReadU32()

	if len(params) < 2 || params[1] {
		pet.CommonMarkActived = buf.ReadU32()
	}

	pet.EffectCount = int(buf.ReadU16())
	for i := 0; i < pet.EffectCount; i++ {
		info := PetEffectInfo{
			ItemID:    buf.ReadU32(),
			Status:    uint32(buf.ReadU8()),
			LeftCount: uint32(buf.ReadU8()),
			EffectID:  uint32(buf.ReadU16()),
		}

		// 8 个效果参数（对应 AS3 的 a1 ~ a8）
		A1 := readEffectArg(buf)
		A2 := readEffectArg(buf)
		A3 := readEffectArg(buf)
		A4 := readEffectArg(buf)
		A5 := readEffectArg(buf)
		A6 := readEffectArg(buf)
		A7 := readEffectArg(buf)
		A8 := readEffectArg(buf)

		// ArgsAll（成对存储）
		info.ArgsAll = map[uint32]uint32{
			A1: A2,
			A3: A4,
			A5: A6,
			A7: A8,
		}

		// Args（动态长度）
		a := []uint32{
			A2, A3, A4,
			A5, A6, A7, A8,
		}

		if A2 == 0 && A3 == 0 && A4 == 0 &&
			A5 == 0 && A6 == 0 {
			info.Args = fmt.Sprintf("%d 0", A1)
		} else {
			info.Args = fmt.Sprintf("%d", A1)
			for i := 6; i >= 0; i-- {
				if a[i] > 0 {
					for j := 0; j <= i; j++ {
						info.Args += fmt.Sprintf(" %d", a[j])
					}
					break
				}
			}
		}

		// Args6
		info.Args6 = fmt.Sprintf(
			"%d %d %d %d %d %d",
			A2, A3, A4,
			A5, A6, A1,
		)

		// effectID == 171：队伍科技
		if info.EffectID == 171 {
			info.AddArray = []int{
				int(A1), int(A2), int(A3),
				int(A4), int(A5), int(A6),
			}

			parseLv := func(v uint32) []int {
				return []int{
					int(v % 16),
					int((v / 16) % 16),
					int((v / 256) % 16),
				}
			}

			info.LevelArray = append(info.LevelArray, parseLv(A7)...)
			info.LevelArray = append(info.LevelArray, parseLv(A8)...)
		}

		pet.EffectList = append(pet.EffectList, info)
	}

	pet.ResistanceInfo = NewPetResistance(buf)

	pet.SkinID = buf.ReadU32()
	if pet.SkinID != 0 {
		pet.SkinID = 1400000 + pet.SkinID
	}

	pet.AssistMoveID = buf.ReadU32()

	buf.Skip(3 * 4)
	for i := range 6 {
		pet.PetValueInfos[i].BaseCur = buf.ReadU32()
		pet.PetValueInfos[i].PvpCur = buf.ReadU32()
		pet.PetValueInfos[i].PveCur = buf.ReadU32()
	}

	pet.BaseCurHP = buf.ReadU32()
	pet.PvpCurHP = buf.ReadU32()
	pet.PveCurHP = buf.ReadU32()

	return pet
}

func NewStoragePet(buf *common.ByteStream) StoragePet {
	return StoragePet{
		ID:        buf.ReadU32(),
		CatchTime: buf.ReadU32(),
		Posi:      buf.ReadU32(),
		Level:     buf.ReadU32(),
	}
}

func NewPetResistance(buf *common.ByteStream) PetResistanceInfo {
	var info PetResistanceInfo

	var v uint32

	// --- 暴击 ---
	v = buf.ReadU32()
	info.Cirt = getBitValue(v, 17, 16)
	info.CirtAdj = getBitValue(v, 1, 16)

	// --- 命中 ---
	v = buf.ReadU32()
	info.Regular = getBitValue(v, 17, 16)
	info.RegularAdj = getBitValue(v, 1, 16)

	// --- 效果触发 ---
	v = buf.ReadU32()
	info.Precent = getBitValue(v, 17, 16)
	info.PrecentAdj = getBitValue(v, 1, 16)

	// --- 控制类抗性（3 组） ---
	v = buf.ReadU32()
	info.Ctl1Idx = getBitValue(v, 17, 8)
	info.Ctl1 = getBitValue(v, 9, 8)
	info.Ctl1Adj = getBitValue(v, 1, 8)

	v = buf.ReadU32()
	info.Ctl2Idx = getBitValue(v, 17, 8)
	info.Ctl2 = getBitValue(v, 9, 8)
	info.Ctl2Adj = getBitValue(v, 1, 8)

	v = buf.ReadU32()
	info.Ctl3Idx = getBitValue(v, 17, 8)
	info.Ctl3 = getBitValue(v, 9, 8)
	info.Ctl3Adj = getBitValue(v, 1, 8)

	// --- 弱化类抗性（3 组） ---
	v = buf.ReadU32()
	info.Weak1Idx = getBitValue(v, 17, 8)
	info.Weak1 = getBitValue(v, 9, 8)
	info.Weak1Adj = getBitValue(v, 1, 8)

	v = buf.ReadU32()
	info.Weak2Idx = getBitValue(v, 17, 8)
	info.Weak2 = getBitValue(v, 9, 8)
	info.Weak2Adj = getBitValue(v, 1, 8)

	v = buf.ReadU32()
	info.Weak3Idx = getBitValue(v, 17, 8)
	info.Weak3 = getBitValue(v, 9, 8)
	info.Weak3Adj = getBitValue(v, 1, 8)

	// --- 基础字段 ---
	info.ResistAll = buf.ReadU32()
	info.ResistState = buf.ReadU32()
	info.RedGem = buf.ReadU32()
	info.GreenGem = buf.ReadU32()
	info.Reserve = buf.ReadU32()

	return info
}

func readEffectArg(buf *common.ByteStream) uint32 {
	low := uint32(buf.ReadU8())
	high := uint32(buf.ReadU8())
	if high != 0 {
		return low + (high << 8)
	}
	return low
}

func getBit(param1 uint32, param2 uint32) uint32 {
	return (param1 & (1 << (param2 - 1))) >> (param2 - 1)
}

func getBitValue(param1 uint32, param2 uint32, param3 uint32) uint32 {
	var result uint32 = 0

	for i := range param3 {
		bit := getBit(param1, param2+i)
		result += bit << i
	}

	return result
}
