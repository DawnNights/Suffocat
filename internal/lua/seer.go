package lua

import (
	"fmt"
	"suffocat/internal/common"
	"suffocat/internal/lua/fight"
	"suffocat/internal/lua/fight/fightscript"
	"suffocat/internal/lua/mintmark"
	"suffocat/internal/lua/petbag"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"
	"suffocat/internal/seerapi"
	"time"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type LuaSeer struct {
	c protocol.SeerConn
	l *lua.LState

	UserID uint32
	Type   string
}

// ------------ 封包相关 ------------

// 发送封包
func (ls *LuaSeer) Send(cmd int, fn *lua.LFunction) *common.ByteStream {
	// 创建写入用 ByteStream
	body := &common.ByteStream{}

	// 判断是否为风险命令
	v, ok := command.UnsafeCommands[cmd]
	if ok {
		ls.c.LogFn("WARN", v)
		return body
	}

	// 调用 Lua 回调：fn(body)
	if fn != nil {
		ls.l.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, luar.New(ls.l, body))
	}

	// 发送数据
	return ls.c.SendAndRecv(uint32(cmd), func(bs *common.ByteStream) {
		bs.WriteBytes(body.ReadAll())
	}).Wait().Body
}

// 接收封包
func (ls *LuaSeer) Recv(cmd int) *common.ByteStream {
	return ls.c.Recv(uint32(cmd)).Wait().Body
}

// ------------ 精灵相关 ------------

// 返回精灵背包
func (ls *LuaSeer) PetBag() *petbag.PetBag {
	return petbag.NewPetBag(ls.c, ls.l)
}

// 设置精灵背包
func (ls *LuaSeer) SetPetBag(v1, v2 *lua.LTable) *lua.LTable {
	battleList, err := common.ToGoList[uint32](v1)
	if err != nil {
		panic(err)
	}

	supportList, err := common.ToGoList[uint32](v2)
	if err != nil {
		panic(err)
	}

	var battlePets []petbag.FindPet
	for _, petId := range battleList {
		battlePets = append(battlePets, petbag.FindPet{ID: petId, Level: 0})
	}

	var supportPets []petbag.FindPet
	for _, petId := range supportList {
		supportPets = append(supportPets, petbag.FindPet{ID: petId, Level: 0})
	}

	var list []uint32
	battleNotFound, supportNotFound := ls.setPetBag(battlePets, supportPets)
	for _, pet := range append(battleNotFound, supportNotFound...) {
		list = append(list, pet.ID)
	}
	return common.ToLuaList(ls.l, list)
}

// 返回精灵仓库
func (ls *LuaSeer) PetStorage() *petbag.PetStorage {
	return petbag.NewPetStorage(ls.c, ls.l)
}

// 返回刻印仓库
func (ls *LuaSeer) MarkStorage() *mintmark.MarkStorage {
	return mintmark.NewMarkStorage(ls.c, ls.l)
}

// 切换背包阵容
func (ls *LuaSeer) SwitchBag(idx uint32) {
	petbag.SwitchPetBag(ls.c, idx)
}

// 是否有指定ID精灵
func (ls *LuaSeer) HasPet(pet_id uint32) bool {
	body := ls.c.SendAndRecv(46089, func(bs *common.ByteStream) {
		bs.WriteU32List(1, pet_id)
	}).Wait().Body

	return body.Skip(4).ReadU32() == 1
}

// ------------ 用户相关 ------------

// 统计指定道具数量
func (ls *LuaSeer) ItemCount(item_id uint32) uint32 {
	body := ls.c.SendAndRecv(command.MULTI_ITEM_LIST, func(bs *common.ByteStream) {
		bs.WriteU32(1).WriteU32(item_id)
	}).Wait().Body

	if body.Length() < 12 {
		return 0
	}

	return body.Skip(8).ReadU32()
}

// 购买指定赛尔豆道具
func (ls *LuaSeer) ItemBuy(item_id uint32, num uint32) bool {
	return !ls.c.SendAndRecv(command.ITEM_BUY, func(bs *common.ByteStream) {
		bs.WriteU32(item_id).WriteU32(num)
	}).Wait().Body.IsEmpty()
}

// 设置称号
func (ls *LuaSeer) SetTitle(title_id uint32) bool {
	state := ls.ensureUserState()
	if current, known := state.Title(); known && current == title_id {
		return true
	}

	body := ls.c.SendAndRecv(command.SETTITLE, func(bs *common.ByteStream) {
		bs.WriteU32(title_id)
	}).Wait().Body

	if body.IsEmpty() {
		current := ls.GetUserInfo(ls.UserID).Title
		state.SetTitle(current)
		if current == title_id {
			ls.c.LogFn("DEBUG", "切换称号成功")
			return true
		}
		return false
	}

	state.SetTitle(title_id)
	ls.c.LogFn("DEBUG", "切换称号成功")
	return true
}

// 设置套装
func (ls *LuaSeer) SetCloth(suit_id, equip_id uint32) bool {
	client := seerapi.Default()
	if client == nil {
		return false
	}
	cloth := client.ClothBySuitID(suit_id)
	if cloth.ID == 0 && suit_id != 0 {
		return false
	}

	if cloth.Face == 0 && equip_id != 0 {
		cloth.Face = equip_id
	}

	body := cloth.ToBody()
	parts := append([]uint32(nil), body[1:]...)

	state := ls.ensureUserState()
	if match, known := state.ClothMatches(parts); known && match {
		return true
	}

	ok := !ls.c.SendAndRecv(command.CHANGE_CLOTH, func(bs *common.ByteStream) {
		bs.WriteU32List(body...)
	}).Wait().Body.IsEmpty()
	if ok {
		state.SetClothes(parts)
		ls.c.LogFn("DEBUG", "切换套装成功")
	}
	return ok
}

// ensureUserState 返回当前连接的角色状态缓存。
// 首次需要时获取称号与装扮，之后复用，直到重新登录。
func (ls *LuaSeer) ensureUserState() *protocol.UserState {
	state := ls.c.UserState()
	if state == nil || state.Known() {
		return state
	}

	title, clothes := ls.fetchUserState()
	state.Init(title, clothes)
	return state
}

func (ls *LuaSeer) fetchUserState() (uint32, []uint32) {
	sim := schema.NewUserInfo(ls.c.SendAndRecv(command.GET_SIM_USERINFO, func(bs *common.ByteStream) {
		bs.WriteU32(ls.UserID)
	}).Wait().Body)
	more := schema.NewUserMoreInfo(ls.c.SendAndRecv(command.GET_MORE_USERINFO, func(bs *common.ByteStream) {
		bs.WriteU32(ls.UserID)
	}).Wait().Body)
	return more.CurTitle, sim.Clothes
}

// 获取火焰
func (ls *LuaSeer) GetFire(fire_type string) bool {
	var fire_id uint32

	switch fire_type {
	case "green":
		fire_id = 6
	case "blue":
		fire_id = 7
	case "purple":
		fire_id = 8
	case "gold":
		fire_id = 9
	default:
		return false
	}

	if ok, _ := ls.hasFire(fire_id); ok {
		return true
	}
	return ls.getFire(fire_id)
}

// 背包精灵治疗
func (ls *LuaSeer) PetCure() {
	ls.c.SendAndRecv(command.PET_CURE_FREE, nil).Wait()
}

// 开关自动精灵治疗
func (ls *LuaSeer) PetCureAuto(open bool) {
	ls.c.SendAndRecv(command.COMMAND_42019, func(bs *common.ByteStream) {
		if open {
			bs.WriteU32(22439).WriteU32(1)
		} else {
			bs.WriteU32(22439).WriteU32(0)
		}
	}).Wait()
}

// 背包精灵压血
func (ls *LuaSeer) PetHPLimit() {
	ls.PetCureAuto(false)
	ls.PetCure()

	count := len(ls.PetBag().GetBag(petbag.TYPE_BATTLE_BAG))
	if count == 0 {
		return
	}

	ls.ItemBuy(300011, uint32(count))
	ls.ItemBuy(300017, uint32(count))

	ls.c.SendAndRecv(41129, func(bs *common.ByteStream) {
		hour := time.Now().In(time.FixedZone("CST", 8*3600)).Hour()
		if hour > 17 && hour < 21 {
			bs.WriteU32(8692)
		} else {
			bs.WriteU32(8694)
		}
	})
	fight.NewReadyFight(ls.l, ls.c).Start()

	for _, pet := range ls.PetBag().GetBag(petbag.TYPE_BATTLE_BAG) {
		pet.UseItem(300011)
		pet.UseItem(300017)
	}
}

// 获取指定用户信息
func (ls *LuaSeer) GetUserInfo(userID uint32) userInfo {
	sim_user := schema.NewUserInfo(ls.c.SendAndRecv(command.GET_SIM_USERINFO, func(bs *common.ByteStream) {
		bs.WriteU32(userID)
	}).Wait().Body)

	user := schema.NewUserMoreInfo(ls.c.SendAndRecv(command.GET_MORE_USERINFO, func(bs *common.ByteStream) {
		bs.WriteU32(userID)
	}).Wait().Body)

	return userInfo{
		UserID:   user.UserID,
		NickName: user.NickName,
		Vip:      user.Vip,
		VipLevel: user.VipLevel,
		MapType:  user.MapType,
		MapID:    user.MapID,
		TeamID:   user.TeamID,
		FireBuff: user.FireBuff,
		Title:    user.CurTitle,
		Clothes:  common.ToLuaList(ls.l, sim_user.Clothes),
	}
}

// ------------ 战斗相关 ------------

// 发送战斗封包
func (ls *LuaSeer) ReadyFight(cmd int, fn *lua.LFunction) *fight.ReadyFight {
	// 创建写入用 ByteStream
	body := &common.ByteStream{}

	// 判断是否为风险命令
	v, ok := command.UnsafeCommands[cmd]
	if ok {
		ls.c.LogFn("ERROR", v)
		return nil
	}

	// 调用 Lua 回调：fn(body)
	ls.l.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, luar.New(ls.l, body))

	// 发送数据
	ls.c.SendAndRecv(uint32(cmd), func(bs *common.ByteStream) {
		bs.WriteBytes(body.ReadAll())
	})
	return fight.NewReadyFight(ls.l, ls.c)
}

func (ls *LuaSeer) ExecFightScript(fs *fightscript.FightScript) *fightscript.FightDriver {
	// 设置背包精灵
	battlePets, supportPets := parsePetList(fs)
	battleNotFound, supportNotFound := ls.setPetBag(battlePets, supportPets)
	shouldReturn := false

	if len(battleNotFound) > 0 {
		client := seerapi.Default()
		for _, findpet := range battleNotFound {
			name := ""
			if client != nil {
				name = client.PetName(findpet.ID)
			}
			ls.c.LogFn("WARN", fmt.Sprintf("未找到出战精灵: %s(%d) ", name, findpet.ID))
		}
		shouldReturn = true
	}

	if len(supportNotFound) > 0 && fs.ScriptConfig.SupportPetsMust {
		client := seerapi.Default()
		for _, findpet := range supportNotFound {
			name := ""
			if client != nil {
				name = client.PetName(findpet.ID)
			}
			ls.c.LogFn("WARN", fmt.Sprintf("未找到待命精灵: %s(%d) ", name, findpet.ID))
		}
		shouldReturn = true
	}

	if shouldReturn {
		return nil
	}

	// 设置称号、套装
	if fs.SeerSet.EnableEquipments {
		if !ls.SetCloth(fs.SeerSet.Suit, fs.SeerSet.Equip) && fs.ScriptConfig.ClothMust {
			ls.c.LogFn("WARN", "切换套装失败，请修改战斗方案")
			return nil
		}
	}

	if fs.SeerSet.EnableTitle {
		if !ls.SetTitle(fs.SeerSet.Title) && fs.ScriptConfig.TitleMust {
			ls.c.LogFn("WARN", "切换称号失败，请修改战斗方案")
			return nil
		}
	}

	// 获取火焰buff
	if fs.SeerSet.BattleFire >= 6 && fs.SeerSet.BattleFire <= 9 {
		file_type := parseFireName(fs.SeerSet.BattleFire)
		ok, remain := ls.hasFire(fs.SeerSet.BattleFire)

		if ok {
			minutes := int(remain.Minutes())
			seconds := int(remain.Seconds()) % 60

			ls.c.LogFn("DEBUG", fmt.Sprintf("当前已拥有%s端%s，剩余%d分钟%d秒", ls.Type, file_type, minutes, seconds))

		} else {
			if ls.getFire(fs.SeerSet.BattleFire) {
				ls.c.LogFn("DEBUG", fmt.Sprintf("领取%s端%s成功", ls.Type, file_type))
			} else {
				ls.c.LogFn("WARN", fmt.Sprintf("领取%s端%s失败", ls.Type, file_type))
			}
		}
	}

	return fightscript.NewFightDriver(fs)
}

func NewLuaSeer(c protocol.SeerConn, l *lua.LState) *LuaSeer {
	lua.OpenBase(l)
	lua.OpenMath(l)
	lua.OpenString(l)
	lua.OpenTable(l)

	return &LuaSeer{
		c:      c,
		l:      l,
		UserID: c.UserID(),
		Type:   c.Type(),
	}
}

// ------------ 辅助函数 ------------

func (ls *LuaSeer) setPetBag(battlePets, supportPets []petbag.FindPet) ([]petbag.FindPet, []petbag.FindPet) {
	ls.PetBag().ClearBag()
	storage := ls.PetStorage()
	return storage.SetPetBag(1, battlePets), storage.SetPetBag(2, supportPets)
}

// Flash端复制绿火
func (ls *LuaSeer) copyGreenFire() bool {
	if ls.Type != "FLASH" {
		return false
	}

	body := ls.c.SendAndRecv(2003, nil).Wait().Body
	count := body.ReadU32()

	for range count {
		info := schema.NewMapUserInfo(body)
		if info.FireBuff != 5 {
			continue
		}

		ls.c.SendAndRecv(command.FIRE_ACT_COPY, func(bs *common.ByteStream) {
			bs.WriteU32(info.UserID)
		})
		return true
	}

	return false
}

func (ls *LuaSeer) hasFire(fire_id uint32) (bool, time.Duration) {
	body := ls.c.SendAndRecv(46046, func(bs *common.ByteStream) {
		bs.WriteU32List(2, 2913, 2914)
	}).Wait().Body.Skip(4)

	fireId := body.ReadU32()
	timestamp := body.ReadU32()

	targetTime := time.Unix(int64(timestamp), 0)
	remain := targetTime.Sub(time.Now().In(time.FixedZone("CST", 8*3600)))

	if fireId == fire_id && remain > 0 {
		return true, remain
	}

	return false, 0
}

func (ls *LuaSeer) getFire(fire_id uint32) bool {
	if ls.Type == "FLASH" {
		body := ls.c.SendAndRecv(command.GET_FIRE_BUFF, func(bs *common.ByteStream) {
			bs.WriteU32(fire_id)
		}).Wait().Body

		if body.IsEmpty() {
			if fire_id == 6 {
				return ls.copyGreenFire()
			}
			return false
		}

		return true
	}

	body := ls.c.SendAndRecv(47348, func(bs *common.ByteStream) {
		bs.WriteU32(1).WriteU32(fire_id).WriteU32(0)
	}).Wait().Body

	if body.IsEmpty() {
		return !ls.c.SendAndRecv(47348, func(bs *common.ByteStream) {
			bs.WriteU32(2).WriteU32(fire_id).WriteU32(0)
		}).Wait().Body.IsEmpty()
	}

	return true
}

// 获取玩家简单信息
type userInfo struct {
	UserID   uint32      // 玩家米米号
	NickName string      // 玩家的昵称
	Vip      uint32      // VIP类型
	VipLevel uint32      // VIP等级
	MapType  uint32      // 当前所在区域的地图类型
	MapID    uint32      // 当前所在地图的实例ID
	TeamID   uint32      // 当前所在战队的ID
	FireBuff uint8       // 火焰Buff
	Title    uint32      // 称号ID
	Clothes  *lua.LTable // 玩家当前穿戴的套装部件列表
}

func parsePetList(fs *fightscript.FightScript) ([]petbag.FindPet, []petbag.FindPet) {
	var battlePets []petbag.FindPet
	var supportPets []petbag.FindPet

	for i, pet := range fs.RosterSet {
		if pet.ID == 0 {
			continue
		}

		findpet := petbag.FindPet{ID: pet.ID, Level: pet.Config.Level}
		if i < 6 {
			battlePets = append(battlePets, findpet)
		} else {
			supportPets = append(supportPets, findpet)
		}
	}

	return battlePets, supportPets
}

func parseFireName(fire_id uint32) string {
	switch fire_id {
	case 6:
		return "绿火"
	case 7:
		return "蓝火"
	case 8:
		return "紫火"
	case 9:
		return "金火"
	}

	return ""
}
