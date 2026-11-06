package fightscript

import "suffocat/internal/lua/fight"

type FightDriver struct {
	script        *FightScript
	petExecutors  map[int]*ActionExecutor // 出战精灵索引 -> 执行器
	petRoundCount map[int]uint32          // 站场回合计数
}

func NewFightDriver(script *FightScript) *FightDriver {
	return &FightDriver{
		script:        script,
		petExecutors:  make(map[int]*ActionExecutor),
		petRoundCount: make(map[int]uint32),
	}
}

// 实现 fight.Driver 接口
func (sd *FightDriver) OnPetAction(roundNum uint32, player *fight.FightPlayer, enemy *fight.FightEnemy) {
	if sd.script.ScriptConfig.FightRoundLimit > 0 && sd.script.ScriptConfig.FightRoundLimit == roundNum {
		player.Escape()
		return
	}

	idx := player.CurrentPetIndex()
	if idx < 0 || idx >= 6 {
		player.Escape()
		return
	}

	exec, ok := sd.petExecutors[idx]
	if !ok {
		exec = NewActionExecutor(sd.script)
		exec.petIndex = idx
		sd.petExecutors[idx] = exec
		sd.petRoundCount[idx] = 0
	}

	sd.petRoundCount[idx]++

	actionType, arg, found := exec.GetNextAction(roundNum, sd.petRoundCount[idx], player, enemy)
	if !found {
		player.UseSkip() // 待机
		return
	}

	switch actionType {
	case "UseSkill":
		player.UseSkill(uint32(toInt(arg)))
	case "UseItem":
		player.UseItem(uint32(toInt(arg)))
	case "UseChangePet":
		targetIdx := toInt(arg)
		if targetIdx >= 0 && targetIdx < len(player.PetBag_) {
			// 清除新精灵的执行器，使其重新开始
			delete(sd.petExecutors, targetIdx)
			player.SwitchPet(player.PetBag_[targetIdx])
		}
	case "UseRetreat":
		player.Escape()
	case "UseSkip":
		player.UseSkip()
	}
}

func (sd *FightDriver) OnPetDeath(player *fight.FightPlayer) {
	idx := player.CurrentPetIndex()
	if idx < 0 || idx >= 6 {
		return
	}
	pet := sd.script.RosterSet[idx]

	// 死切逻辑
	targetIdx := pet.Config.DeathChange

	if targetIdx < 0 || targetIdx >= 6 || targetIdx == idx || player.PetBag_[targetIdx].HP == 0 {
		for i, p := range player.PetBag_ {
			if p.HP == 0 || p.CatchTime == player.ActivePet.CatchTime {
				continue
			}

			targetIdx = i
			break
		}
	}

	player.SwitchPet(player.PetBag_[targetIdx])
	// 清理执行器
	delete(sd.petExecutors, int(targetIdx))
	delete(sd.petExecutors, idx)
	delete(sd.petRoundCount, idx)
}
