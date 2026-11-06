package fightscript

import (
	"fmt"
	"strconv"
	"suffocat/internal/lua/fight"
)

type ActionExecutor struct {
	script      *FightScript
	currentLoop int
	currentStep int
	petIndex    int
}

func NewActionExecutor(script *FightScript) *ActionExecutor {
	return &ActionExecutor{
		script:      script,
		currentLoop: 0,
		currentStep: 0,
	}
}

func (e *ActionExecutor) GetNextAction(roundNum uint32, thisRound uint32, player *fight.FightPlayer, enemy *fight.FightEnemy) (actionType string, arg any, found bool) {
	pet := e.script.RosterSet[e.petIndex]
	loops := pet.ActionSequence

	startLoop := e.currentLoop
	for {
		if e.currentLoop >= len(loops) {
			e.currentLoop = 0
		}
		loop := loops[e.currentLoop]
		if e.currentStep >= len(loop) {
			e.currentLoop++
			e.currentStep = 0
			if e.currentLoop == startLoop {
				return "Skip", nil, false
			}
			continue
		}

		step := loop[e.currentStep]

		if e.checkConditions(step.If, roundNum, thisRound, player, enemy) {
			actionType = step.Use.Type
			arg = step.Use.Arg

			if actionType == "Goto" {
				targetLoop := toInt(arg)
				e.currentLoop = targetLoop
				e.currentStep = 0
				continue
			}

			e.currentStep++
			return actionType, arg, true
		} else {
			e.currentStep++
			continue
		}
	}
}

// checkConditions 现在接收 roundNum 和 thisRound 用于条件解析
func (e *ActionExecutor) checkConditions(conds []Condition, roundNum, thisRound uint32, player *fight.FightPlayer, enemy *fight.FightEnemy) bool {
	if len(conds) == 0 {
		return true
	}
	for _, c := range conds {
		leftVal, err := e.resolveLeftValue(c.Left, c.LeftArg, roundNum, thisRound, player, enemy)
		if err != nil {
			return false
		}

		if !compare(leftVal, c.Operator, toInt(c.Right)) {
			return false
		}
	}
	return true
}

// resolveLeftValue 加入 roundNum 和 thisRound 用于解析动态变量
func (e *ActionExecutor) resolveLeftValue(left string, arg any, roundNum, thisRound uint32, player *fight.FightPlayer, enemy *fight.FightEnemy) (int, error) {
	switch left {
	case "PetHp":
		idx := toInt(arg)
		if idx >= 0 && idx < len(player.PetBag_) {
			return int(player.PetBag_[idx].HP), nil
		}
		return 0, fmt.Errorf("invalid pet index %d", idx)
	case "PetHpPercent":
		idx := toInt(arg)
		if idx >= 0 && idx < len(player.PetBag_) {
			pet := player.PetBag_[idx]
			if pet.MaxHP == 0 {
				return 0, nil
			}
			return int(pet.HP * 100 / pet.MaxHP), nil
		}
		return 0, fmt.Errorf("invalid pet index %d", idx)
	case "EnemyPetHp":
		return int(enemy.ActivePet.HP), nil
	case "EnemyPetHpPercent":
		if enemy.ActivePet.MaxHP == 0 {
			return 0, nil
		}
		return int(enemy.ActivePet.HP * 100 / enemy.ActivePet.MaxHP), nil
	case "SkillPPCount":
		skillID := toInt(arg)
		for _, sk := range player.ActivePet.SkillList_ {
			if int(sk.ID) == skillID {
				return int(sk.PP), nil
			}
		}
		return 0, fmt.Errorf("skill %d not found", skillID)
	case "Round":
		return int(roundNum), nil
	case "ThisRound":
		return int(thisRound), nil
	case "Status":
		statusIdx := toInt(arg)
		if statusIdx == -1 {
			return sum(player.ActivePet.FightStatus_...), nil
		}

		if len(player.ActivePet.FightStatus_) > statusIdx {
			return int(player.ActivePet.FightStatus_[statusIdx]), nil
		}

		return 0, fmt.Errorf("invalid Status index %d", statusIdx)
	case "EnemyStatus":
		statusIdx := toInt(arg)
		if statusIdx == -1 {
			return sum(enemy.ActivePet.FightStatus_...), nil
		}

		if len(enemy.ActivePet.FightStatus_) > statusIdx {
			return int(enemy.ActivePet.FightStatus_[statusIdx]), nil
		}

		return 0, fmt.Errorf("invalid Status index %d", statusIdx)
	case "StatLevel":
		stat := toInt(arg)
		lv := e.getPlayerStatLevel(player, stat)
		return lv, nil
	case "EnemyStatLevel":
		stat := toInt(arg)
		lv := e.getEnemyStatLevel(enemy, stat)
		return lv, nil
	case "EnemyPetId":
		return int(enemy.ActivePet.ID), nil
	default:
		return 0, fmt.Errorf("unknown left: %s", left)
	}
}

func (e *ActionExecutor) getPlayerStatLevel(player *fight.FightPlayer, stat int) int {
	lv := player.ActivePet.FightLV
	switch stat {
	case 0:
		return lv.Atk
	case 1:
		return lv.Def
	case 2:
		return lv.SpAtk
	case 3:
		return lv.SpDef
	case 4:
		return lv.Speed
	case 5:
		return lv.Accuracy
	case -1:
		return lv.Total()
	default:
		return 0
	}
}

func (e *ActionExecutor) getEnemyStatLevel(enemy *fight.FightEnemy, stat int) int {
	lv := enemy.ActivePet.FightLV
	switch stat {
	case 0:
		return lv.Atk
	case 1:
		return lv.Def
	case 2:
		return lv.SpAtk
	case 3:
		return lv.SpDef
	case 4:
		return lv.Speed
	case 5:
		return lv.Accuracy
	case -1:
		return lv.Total()
	default:
		return 0
	}
}

// ---------- 工具函数 ----------
func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f, true
	default:
		return 0, false
	}
}

func compare(a int, op string, b int) bool {
	switch op {
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "==":
		return a == b
	case "!=":
		return a != b
	}
	return false
}

func sum(numbers ...uint8) int {
	var total int
	for _, n := range numbers {
		total = total + int(n)
	}
	return total
}
