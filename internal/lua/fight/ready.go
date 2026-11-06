package fight

import (
	"suffocat/internal/common"
	"suffocat/internal/protocol"
	"suffocat/internal/protocol/command"
	"suffocat/internal/protocol/schema"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	FIGHTING = iota
	FIGHT_WIN
	FIGHT_LOST
)

type ReadyFight struct {
	l      *lua.LState
	conn   protocol.SeerConn
	logger FightLogger

	onPetDeath  func(*FightPlayer)
	onPetAction func(uint32, *FightPlayer, *FightEnemy)

	roundNum uint32
	player   *FightPlayer
	enemy    *FightEnemy

	fightState int
}

func (rf *ReadyFight) OnPetDeath(fn *lua.LFunction) {
	rf.updateLuaField()
	rf.onPetDeath = func(player *FightPlayer) {
		rf.l.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, luar.New(rf.l, player))
	}
}

func (rf *ReadyFight) OnPetAction(fn *lua.LFunction) {
	rf.updateLuaField()
	rf.onPetAction = func(roundNum uint32, player *FightPlayer, enemy *FightEnemy) {
		rf.l.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LNumber(roundNum), luar.New(rf.l, player), luar.New(rf.l, enemy))
	}
}

// Driver 战斗驱动器接口，可由外部脚本系统实现
type Driver interface {
	OnPetAction(roundNum uint32, player *FightPlayer, enemy *FightEnemy)
	OnPetDeath(player *FightPlayer)
}

func (rf *ReadyFight) BindDriver(driver Driver) {
	rf.updateLuaField()
	rf.onPetAction = driver.OnPetAction
	rf.onPetDeath = driver.OnPetDeath
}

func (rf *ReadyFight) Start() bool {
	rf.beforeFight()
	defer rf.afterFight()

	rf.conn.RegisterHandler(command.NOTE_USE_SKILL, func(packet protocol.SeerPacket) {
		// 更新对战信息
		frist, second := schema.NewUseSkillInfo(packet.Body)
		rf.roundNum = frist.Round
		playerLastPet := *rf.player.ActivePet
		enemyLastPet := *rf.enemy.ActivePet

		rf.logger.LogRound(rf.roundNum, *rf.player, *rf.enemy)
		if frist.UserID == rf.player.UserID {
			playerGain := int(enemyLastPet.HP) - int(second.RemainHP) - int(frist.TotalDamage())
			if playerGain > 0 {
				frist.GainHP = uint32(playerGain)
			}

			enemyGain := int(playerLastPet.HP) - int(frist.RemainHP) - int(second.TotalDamage())
			if enemyGain > 0 {
				second.GainHP = uint32(enemyGain)
			}

			rf.player.Update(frist)
			rf.enemy.Update(second)

			rf.logger.LogSkillResult(true, playerLastPet, rf.player.LastResult)
			rf.logger.LogSkillResult(false, *rf.enemy.ActivePet, rf.enemy.LastResult)
		} else {
			enemyGain := int(playerLastPet.HP) - int(frist.RemainHP) - int(second.TotalDamage())
			if enemyGain > 0 {
				frist.GainHP = uint32(enemyGain)
			}

			playerGain := int(enemyLastPet.HP) - int(second.RemainHP) - int(frist.TotalDamage())
			if playerGain > 0 {
				second.GainHP = uint32(playerGain)
			}

			rf.enemy.Update(frist)
			rf.player.Update(second)

			rf.logger.LogSkillResult(false, enemyLastPet, rf.enemy.LastResult)
			rf.logger.LogSkillResult(true, *rf.player.ActivePet, rf.player.LastResult)
		}

		rf.logger.LogRoundEnd(rf.roundNum, *rf.player, *rf.enemy)
		rf.player.IsAction = false

		// 若任意一方精灵全部阵亡，则战斗即将结束，FIGHT_OVER 随后会负责唤醒
		if !rf.player.IsAllDead() && !rf.enemy.IsAllDead() {
			select {
			case rf.player.actionDone <- struct{}{}:
			default:
			}
		}
	})

	for rf.fightState == FIGHTING {
		// 输出我方精灵死亡信息并执行切换处理
		if rf.player.ActivePet.IsDeath() {
			rf.logger.LogPetDeath(true, *rf.player.ActivePet)
			rf.onPetDeath(rf.player)
		}

		// 输出敌方精灵死亡信息
		if rf.enemy.ActivePet.IsDeath() {
			rf.logger.LogPetDeath(false, *rf.enemy.ActivePet)
		}

		// 执行行动函数
		rf.onPetAction(rf.roundNum, rf.player, rf.enemy)
		if !rf.player.IsAction {
			rf.player.UseSkillByIdx(1)
		}
		<-rf.player.actionDone
	}

	rf.conn.RegisterHandler(command.NOTE_USE_SKILL, nil)
	return rf.fightState == FIGHT_WIN
}

func (rf *ReadyFight) beforeFight() {
	// 注册精灵切换包处理
	rf.conn.RegisterHandler(command.CHANGE_PET, func(packet protocol.SeerPacket) {
		if packet.Body.IsEmpty() {
			return
		}

		change := schema.NewChangePet(packet.Body)
		if change.UserID == rf.player.UserID {
			rf.player.PetChange(change)
			rf.logger.LogPetChange(true, *rf.player.ActivePet)
		} else {
			rf.enemy.PetChange(change)
			rf.logger.LogPetChange(false, *rf.enemy.ActivePet)

		}
	})

	// 注册对战结束包处理
	rf.conn.RegisterHandler(command.FIGHT_OVER, func(packet protocol.SeerPacket) {
		packet.Body.ReadU8()  // _type
		packet.Body.ReadU32() // _reason

		if packet.Body.ReadU32() == rf.player.UserID {
			rf.fightState = FIGHT_WIN
		} else {
			rf.fightState = FIGHT_LOST
		}
		select {
		case rf.player.actionDone <- struct{}{}:
		default:
		}
	})

	// 注册撤退处理
	rf.conn.RegisterHandler(command.ESCAPE_FIGHT, func(packet protocol.SeerPacket) {
		rf.player.IsEscape = true
	})

	// 注册使用道具处理
	rf.conn.RegisterHandler(command.USE_PET_ITEM, func(packet protocol.SeerPacket) {
		body := packet.Body
		if body.Length() > 8 {
			rf.logger.LogItemResult(body.ReadU32() == rf.player.UserID, body.ReadU32())
		}
	})

	// 输出开战信息
	rf.logger.LogFightStart(*rf.player, *rf.enemy)
}

func (rf *ReadyFight) afterFight() {
	// 销毁对战包处理
	rf.conn.RegisterHandler(command.CHANGE_PET, nil)
	rf.conn.RegisterHandler(command.ESCAPE_FIGHT, nil)
	rf.conn.RegisterHandler(command.FIGHT_OVER, nil)
	rf.conn.RegisterHandler(command.USE_PET_ITEM, nil)

	// 输出对战结果
	rf.logger.LogFightOver(rf.fightState == FIGHT_WIN, rf.player.IsEscape)
}

func (rf *ReadyFight) updateLuaField() {
	rf.player.PetBag = common.ToLuaList(rf.l, rf.player.PetBag_)
	rf.enemy.PetBag = common.ToLuaList(rf.l, rf.enemy.PetBag_)

	rf.player.ActivePet.FightStatus = common.ToLuaList(rf.l, rf.player.ActivePet.FightStatus_)
	rf.enemy.ActivePet.FightStatus = common.ToLuaList(rf.l, rf.enemy.ActivePet.FightStatus_)

	rf.player.ActivePet.SkillList = common.ToLuaList(rf.l, rf.player.ActivePet.SkillList_)
	rf.enemy.ActivePet.SkillList = common.ToLuaList(rf.l, rf.enemy.ActivePet.SkillList_)
}

func NewReadyFight(l *lua.LState, conn protocol.SeerConn) *ReadyFight {
	// 发送准备包
	ready_info := schema.NewReadyToFightInfo(conn.Recv(command.NOTE_READY_TO_FIGHT).Wait().Body)
	conn.SendAndRecv(command.READY_TO_FIGHT, nil)

	// 获取对战信息
	bag_pets := []schema.PetInfo{}
	body := conn.SendAndRecv(command.GET_PET_INFO_BY_ONCE, nil).Wait().Body
	for range body.ReadU32() {
		bag_pets = append(bag_pets, schema.NewPetInfo(body))
	}

	// 构建战斗对象
	player := newPlayer(conn, ready_info.Player, bag_pets)
	player.Clothes = common.ToLuaList(l, player.Clothes_)

	rf := &ReadyFight{
		l:           l,
		conn:        conn,
		logger:      defaultLogger,
		onPetDeath:  defaultPetDeath,
		onPetAction: defaultPetAction,
		roundNum:    0,
		player:      player,
		enemy:       newEnemy(ready_info.Enemy),
		fightState:  FIGHTING,
	}

	return rf
}

func defaultPetDeath(player *FightPlayer) {
	player.SwitchNextPet()
}

func defaultPetAction(_ uint32, player *FightPlayer, _ *FightEnemy) {
	player.UseSkillByIdx(1)
}
