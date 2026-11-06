package schema

import (
	"strings"
	"suffocat/internal/common"
)

// UserInfo 包含玩家的核心信息、状态、社交关系及装备详情
type UserInfo struct {
	UserID          uint32 // 玩家米米号
	NickName        string // 玩家的昵称
	Color           uint32
	Texture         uint32
	Vip             uint32   // VIP类型
	Status          uint32   // 玩家在线状态（如在线、离开、忙碌）
	MapType         uint32   // 当前所在区域的地图类型
	MapID           uint32   // 当前所在地图的实例ID
	IsCanBeTeacher  bool     // 标记该玩家是否有资格成为导师
	TeacherID       uint32   // 此玩家的导师ID
	StudentID       uint32   // 此玩家所属学员的ID
	GraduationCount uint32   // 已完成毕业的学员数量
	VipLevel        uint32   // VIP等级
	TeamID          uint32   // 当前所在战队的ID
	Clothes         []uint32 // 玩家当前穿戴的套装部件列表
	FightArenaPoint uint32
	FireBuff        uint8  // 火焰Buff
	LoginTime       uint32 // 最后登录时间戳
}

type UserMoreInfo struct {
	// ===== 基础信息 =====
	UserID   uint32 // 玩家米米号
	NickName string // 昵称
	RegTime  uint32 // 注册时间

	Color   uint32
	Texture uint32

	Vip      uint32
	VipLevel uint32

	IsExtremeNono bool

	// ===== 在线状态 =====
	Status  uint32
	MapType uint32
	MapID   uint32

	// ===== 师徒 =====
	IsCanBeTeacher bool
	TeacherID      uint32
	StudentID      uint32

	GraduationCount uint32

	// ===== 战队 =====
	TeamID uint32

	// ===== 装扮 =====
	Clothes []uint32

	// ===== 战斗 =====
	FightArenaPoint uint32
	FireBuff        uint8

	LoginTime uint32

	// ===== 宠物 / 成就 =====
	PetAllNum    uint32
	PetMaxLev    uint32
	TotalAchieve uint32
	AchieShine   uint32
	AchieRank    uint32
	CurTitle     uint32

	BossAchievement []bool

	// ===== 对战统计 =====
	TotalClassWins uint32
	MonKingWin     uint32
	TeamBoss       uint32
	MessWin        uint32
	LuckyFightWin  uint32
	FightArenaWin  uint32
	RoyaleWin      uint32
	DarkFightWin   uint32
	HighFightWin   uint16

	// ===== 关卡 =====
	MaxFreshStage    uint32
	MaxStage         uint32
	MaxKingStage     uint32
	MaxKingHeroStage uint32
	MaxLadderState   uint32
	MaxFortuneState  uint32
	MaxArenaWins     uint32

	// ===== 巅峰数据 =====
	DeltaTopHonour uint32
	CurTopHonour   uint32
	DeltaTopLv     uint32

	TopWinCount   uint32
	TopLossCount  uint32
	MaxTopWinSucc uint32
	CurTopWinSucc uint32

	TopTwoLvl    uint32
	TopTwoWin    uint32
	TopTwoLose   uint32
	TopTwoMaxWin uint32
	TopTwoSurWin uint32

	TopThreeLvl    uint32
	TopThreeWin    uint32
	TopThreeLose   uint32
	TopThreeMaxWin uint32
	TopThreeSurWin uint32

	TopFourLvl    uint32
	TopFourWin    uint32
	TopFourLose   uint32
	TopFourMaxWin uint32
	TopFourSurWin uint32

	TopFiveLvl    uint32
	TopFiveWin    uint32
	TopFiveLose   uint32
	TopFiveMaxWin uint32
	TopFiveSurWin uint32

	TopSixLvl    uint32
	TopSixWin    uint32
	TopSixLose   uint32
	TopSixMaxWin uint32
	TopSixSurWin uint32

	TopSevenLvl    uint32
	TopSevenWin    uint32
	TopSevenLose   uint32
	TopSevenMaxWin uint32
	TopSevenSurWin uint32

	TopEightLvl    uint32
	TopEightWin    uint32
	TopEightLose   uint32
	TopEightMaxWin uint32
	TopEightSurWin uint32

	TopNineLvl    uint32
	TopNineWin    uint32
	TopNineLose   uint32
	TopNineMaxWin uint32
	TopNineSurWin uint32

	TopTenLvl    uint32
	TopTenWin    uint32
	TopTenLose   uint32
	TopTenMaxWin uint32
	TopTenSurWin uint32

	TopElevenLvl    uint32
	TopElevenWin    uint32
	TopElevenLose   uint32
	TopElevenMaxWin uint32
	TopElevenSurWin uint32

	TopTwelveLvl    uint32
	TopTwelveWin    uint32
	TopTwelveLose   uint32
	TopTwelveMaxWin uint32
	TopTwelveSurWin uint32

	// ===== 活动 =====
	TrainerDoorNum     uint32
	CarnivalTotalScore uint32

	HurdlesWinNum  uint32
	HurdlesLoseNum uint32

	TugWinNum  uint32
	TugLoseNum uint32
	TugDrawNum uint32

	JumpNum uint32

	MaxSpaceArenaWins uint32

	ExtremeLawLevel uint32
	BattleLabInfo   uint32

	ZheguangWinTimes uint32
	DreamMessWins    uint32

	AresUnionTeam uint32
}

func NewUserMoreInfo(buf *common.ByteStream) UserMoreInfo {
	info := UserMoreInfo{}

	// hasMoreInfo = true

	info.UserID = buf.ReadU32()
	info.NickName = strings.ReplaceAll(string(buf.ReadBytes(16)), "\u0000", "")
	info.RegTime = buf.ReadU32()

	// isExtremeNono
	flag := buf.ReadU8()
	info.IsExtremeNono = ((flag >> 1) & 1) == 1

	info.PetAllNum = buf.ReadU32()
	info.PetMaxLev = buf.ReadU32()
	info.TotalClassWins = buf.ReadU32()
	info.TotalAchieve = buf.ReadU32()
	info.AchieShine = buf.ReadU32()
	info.AchieRank = buf.ReadU32()
	info.CurTitle = buf.ReadU32()

	// bossAchievement[199]
	info.BossAchievement = make([]bool, 199)
	for i := 0; i < 199; i++ {
		info.BossAchievement[i] = buf.ReadU8() != 0
	}

	info.GraduationCount = buf.ReadU32()
	info.MonKingWin = buf.ReadU32()
	info.TeamBoss = buf.ReadU32()
	info.MessWin = buf.ReadU32()
	info.LuckyFightWin = buf.ReadU32()
	info.FightArenaWin = buf.ReadU32()
	info.RoyaleWin = buf.ReadU32()
	info.DarkFightWin = buf.ReadU32()

	info.MaxFreshStage = buf.ReadU32()
	info.MaxStage = buf.ReadU32()
	info.MaxKingStage = buf.ReadU32()
	info.MaxKingHeroStage = buf.ReadU32()
	info.MaxLadderState = buf.ReadU32()
	info.MaxFortuneState = buf.ReadU32()
	info.MaxArenaWins = buf.ReadU32()

	info.DeltaTopHonour = buf.ReadU32()
	info.CurTopHonour = buf.ReadU32()
	info.DeltaTopLv = buf.ReadU32()

	info.TopWinCount = buf.ReadU32()
	info.TopLossCount = buf.ReadU32()
	info.MaxTopWinSucc = buf.ReadU32()
	info.CurTopWinSucc = buf.ReadU32()

	// top two
	info.TopTwoLvl = buf.ReadU32()
	info.TopTwoWin = buf.ReadU32()
	info.TopTwoLose = buf.ReadU32()
	info.TopTwoMaxWin = buf.ReadU32()
	info.TopTwoSurWin = buf.ReadU32()

	// top three
	info.TopThreeLvl = buf.ReadU32()
	info.TopThreeWin = buf.ReadU32()
	info.TopThreeLose = buf.ReadU32()
	info.TopThreeMaxWin = buf.ReadU32()
	info.TopThreeSurWin = buf.ReadU32()

	// top four
	info.TopFourLvl = buf.ReadU32()
	info.TopFourWin = buf.ReadU32()
	info.TopFourLose = buf.ReadU32()
	info.TopFourMaxWin = buf.ReadU32()
	info.TopFourSurWin = buf.ReadU32()

	// top five
	info.TopFiveLvl = buf.ReadU32()
	info.TopFiveWin = buf.ReadU32()
	info.TopFiveLose = buf.ReadU32()
	info.TopFiveMaxWin = buf.ReadU32()
	info.TopFiveSurWin = buf.ReadU32()

	// top six
	info.TopSixLvl = buf.ReadU32()
	info.TopSixWin = buf.ReadU32()
	info.TopSixLose = buf.ReadU32()
	info.TopSixMaxWin = buf.ReadU32()
	info.TopSixSurWin = buf.ReadU32()

	// top seven
	info.TopSevenLvl = buf.ReadU32()
	info.TopSevenWin = buf.ReadU32()
	info.TopSevenLose = buf.ReadU32()
	info.TopSevenMaxWin = buf.ReadU32()
	info.TopSevenSurWin = buf.ReadU32()

	// top eight
	info.TopEightLvl = buf.ReadU32()
	info.TopEightWin = buf.ReadU32()
	info.TopEightLose = buf.ReadU32()
	info.TopEightMaxWin = buf.ReadU32()
	info.TopEightSurWin = buf.ReadU32()

	// top nine
	info.TopNineLvl = buf.ReadU32()
	info.TopNineWin = buf.ReadU32()
	info.TopNineLose = buf.ReadU32()
	info.TopNineMaxWin = buf.ReadU32()
	info.TopNineSurWin = buf.ReadU32()

	// top ten
	info.TopTenLvl = buf.ReadU32()
	info.TopTenWin = buf.ReadU32()
	info.TopTenLose = buf.ReadU32()
	info.TopTenMaxWin = buf.ReadU32()
	info.TopTenSurWin = buf.ReadU32()

	// top eleven
	info.TopElevenLvl = buf.ReadU32()
	info.TopElevenWin = buf.ReadU32()
	info.TopElevenLose = buf.ReadU32()
	info.TopElevenMaxWin = buf.ReadU32()
	info.TopElevenSurWin = buf.ReadU32()

	// top twelve
	info.TopTwelveLvl = buf.ReadU32()
	info.TopTwelveWin = buf.ReadU32()
	info.TopTwelveLose = buf.ReadU32()
	info.TopTwelveMaxWin = buf.ReadU32()
	info.TopTwelveSurWin = buf.ReadU32()

	info.HighFightWin = buf.ReadU16()

	// skip 3 uint32
	buf.Skip(12)

	info.TrainerDoorNum = buf.ReadU32()
	info.CarnivalTotalScore = buf.ReadU32()

	info.HurdlesWinNum = buf.ReadU32()
	info.HurdlesLoseNum = buf.ReadU32()

	info.TugWinNum = buf.ReadU32()
	info.TugLoseNum = buf.ReadU32()
	info.TugDrawNum = buf.ReadU32()

	info.JumpNum = buf.ReadU32()

	// skip 3 uint32
	buf.Skip(12)

	info.MaxSpaceArenaWins = buf.ReadU32()
	info.ExtremeLawLevel = buf.ReadU32()
	info.BattleLabInfo = buf.ReadU32()
	info.ZheguangWinTimes = buf.ReadU32()
	info.DreamMessWins = buf.ReadU32()
	info.AresUnionTeam = buf.ReadU32()

	return info
}

func NewUserInfo(buf *common.ByteStream) UserInfo {
	info := UserInfo{}

	info.UserID = buf.ReadU32()
	info.NickName = strings.ReplaceAll(string(buf.ReadBytes(16)), "\u0000", "")

	info.Color = buf.ReadU32()
	info.Texture = buf.ReadU32()
	info.Vip = buf.ReadU32()

	buf.Skip(1)
	info.Status = buf.ReadU32()
	info.MapType = buf.ReadU32()
	info.MapID = buf.ReadU32()
	info.IsCanBeTeacher = buf.ReadU32() == 1
	info.TeacherID = buf.ReadU32()
	info.StudentID = buf.ReadU32()
	info.GraduationCount = buf.ReadU32()
	info.VipLevel = buf.ReadU32()
	info.TeamID = buf.ReadU32()

	buf.Skip(4)
	tmp := int(buf.ReadU32())

	for range tmp {
		info.Clothes = append(info.Clothes, buf.ReadU32())
		buf.ReadU32()
	}

	info.FightArenaPoint = buf.ReadU32()
	info.FireBuff = buf.ReadU8()
	info.LoginTime = buf.ReadU32()

	return info
}

func NewMapUserInfo(buf *common.ByteStream) UserInfo {
	info := UserInfo{}

	// 1. sysTime (int32) -> 跳过
	_ = buf.ReadU32()

	// 2. userID
	info.UserID = buf.ReadU32()

	// 3. nick (16字节)
	nickBytes := buf.ReadBytes(16)
	info.NickName = strings.ReplaceAll(string(nickBytes), "\x00", "")

	// 4. curTitle -> 跳过
	_ = buf.ReadU32()

	// 5. color
	info.Color = buf.ReadU32()

	// 6. texture
	info.Texture = buf.ReadU32()

	// 7. jobTitle -> 跳过
	_ = buf.ReadU32()

	// 8. isFamous (uint32) -> 跳过
	_ = buf.ReadU32()

	// 9. vipTitle -> 跳过
	_ = buf.ReadU32()

	// 10. _loc3_ (uint32) 提取 bit0 作为 Vip
	loc3 := buf.ReadU32()
	info.Vip = loc3 & 1

	// 11. _loc4_ (1字节) -> 跳过
	_ = buf.ReadU8()

	// 12. vipStage -> VipLevel
	info.VipLevel = buf.ReadU32()

	// 13. actionType -> 跳过
	_ = buf.ReadU32()

	// 14. pos (两个 uint32) -> 跳过
	_ = buf.ReadU32()
	_ = buf.ReadU32()

	// 15. action -> 跳过
	_ = buf.ReadU32()

	// 16. direction -> 跳过
	_ = buf.ReadU32()

	// 17. changeShape -> 跳过
	_ = buf.ReadU32()

	// 18. darkLight -> 跳过
	_ = buf.ReadU32()

	// 19. luoboteStatus -> 跳过
	_ = buf.ReadU32()

	// 20. aresUnionTeam -> 跳过
	_ = buf.ReadU32()

	// 21. aiErFuAndMiYouLaStatus -> 跳过
	_ = buf.ReadU32()

	// 22. usersCamp -> 跳过
	_ = buf.ReadU32()

	// 23. spiritTime -> 跳过
	_ = buf.ReadU32()

	// 24. spiritID -> 跳过
	_ = buf.ReadU32()

	// 25. isBright (uint32) -> 跳过
	_ = buf.ReadU32()

	// 26. specialBright -> 跳过
	_ = buf.ReadU32()

	// 27. otherPetID -> 跳过
	_ = buf.ReadU32()

	// 28. otherBright -> 跳过
	_ = buf.ReadU32()

	// 29. otherEatBright -> 跳过
	_ = buf.ReadU32()

	// 30. fightFlag -> 跳过
	_ = buf.ReadU32()

	// 31. teacherID
	info.TeacherID = buf.ReadU32()

	// 32. studentID
	info.StudentID = buf.ReadU32()

	// 33. _loc5_ (uint32) 及后续 nonoState 循环 -> 跳过
	_ = buf.ReadU32()

	// 34. nonoColor -> 跳过
	_ = buf.ReadU32()

	// 35. superNono (uint32) -> 跳过
	_ = buf.ReadU32()

	// 36. nonoChangeToPet -> 跳过
	_ = buf.ReadU32()

	// 37. transId -> 跳过
	_ = buf.ReadU32()

	// 38. transDuration -> 跳过
	_ = buf.ReadU32()

	// 39. _loc7_ (数组长度) 及其后的 uint32 数组 -> 跳过
	loc7 := buf.ReadU32()
	for i := 0; i < int(loc7); i++ {
		_ = buf.ReadU32()
	}

	// 40. mountId -> 跳过
	_ = buf.ReadU32()

	// 41. GroupInfo (包含 GroupIDInfo) -> 跳过
	_ = buf.ReadU16()     // svrID
	_ = buf.ReadU32()     // seqID
	_ = buf.ReadU32()     // time
	_ = buf.ReadU32()     // leaderID
	_ = buf.ReadBytes(16) // groupName
	_ = buf.ReadU8()      // sctID
	_ = buf.ReadU8()      // pointID

	// 42. TeamInfo -> 跳过
	_ = buf.ReadU32()    // id
	_ = buf.ReadU32()    // coreCount
	_ = buf.ReadU32()    // isShow
	_ = buf.ReadU16()    // logoBg
	_ = buf.ReadU16()    // logoIcon
	_ = buf.ReadU16()    // logoColor
	_ = buf.ReadU16()    // txtColor
	_ = buf.ReadBytes(4) // logoWord

	// 43. clothes 数组
	clothesLen := buf.ReadU32()
	info.Clothes = make([]uint32, 0, clothesLen)
	for i := 0; i < int(clothesLen); i++ {
		id := buf.ReadU32()
		_ = buf.ReadU32()
		info.Clothes = append(info.Clothes, id)
	}

	// 44. topFightEffect -> 跳过
	_ = buf.ReadU32()

	// 45. fireBuff
	info.FireBuff = buf.ReadU8()

	// 46. tangyuan -> 跳过
	_ = buf.ReadU32()

	// 47. foolsdayMask -> 跳过
	_ = buf.ReadU32()

	// 48. tigerFightTeam -> 跳过
	_ = buf.ReadU32()

	// 49. tigerFightScore -> 跳过
	_ = buf.ReadU32()

	// 50. crackCupTeamId -> 跳过
	_ = buf.ReadU32()

	// 51. lordOfWarTeamId -> 跳过
	_ = buf.ReadU32()

	// 52. decorateList (长度固定5，但按读取值跳过)
	_ = buf.ReadU32()
	for i := 0; i < 5; i++ {
		_ = buf.ReadU32() // 每个元素只读一个 uint32 (id)
	}

	// 53. 最后的额外 uint32
	_ = buf.ReadU32()

	return info
}
