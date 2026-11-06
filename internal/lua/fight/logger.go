package fight

import (
	"fmt"
	"strings"
	"suffocat/internal/seerapi"
)

type FightLogger interface {
	LogFightStart(player FightPlayer, enemy FightEnemy)
	LogRound(round uint32, player FightPlayer, enemy FightEnemy)
	LogRoundEnd(round uint32, player FightPlayer, enemy FightEnemy)
	LogPetChange(isPlayer bool, pet FightActivePet)
	LogPetDeath(isPlayer bool, pet FightActivePet)
	LogSkillResult(isPlayer bool, pet FightActivePet, skill SkillResult)
	LogItemResult(isPlayer bool, itemID uint32)
	LogFightOver(win bool, escape bool)
}

func SetFightLogger(logger FightLogger) {
	defaultLogger = logger
}

var defaultLogger FightLogger = &CustomLogger{
	LogFn: func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	},
}

type CustomLogger struct {
	LogFn func(format string, args ...any)
}

func (l *CustomLogger) LogFightStart(player FightPlayer, enemy FightEnemy) {
	l.LogFn("\n------- 战斗开始 -------\n")
	l.LogPetChange(true, *player.ActivePet)
	l.LogPetChange(false, *enemy.ActivePet)
}

func (l *CustomLogger) LogRound(round uint32, _ FightPlayer, _ FightEnemy) {
	l.LogFn("\n------- 回合:%d -------\n", round)
}

func (l *CustomLogger) LogRoundEnd(_ uint32, _ FightPlayer, _ FightEnemy) {
	// l.LogFn("------- 回合结束 -------\n\n")
}

func (l *CustomLogger) LogPetChange(isPlayer bool, pet FightActivePet) {
	if isPlayer {
		l.LogFn("【我方】%s[%d] 出场了！", pet.Name, pet.ID)
	} else {
		l.LogFn("（敌方）%s[%d] 出场了！", pet.Name, pet.ID)
	}
}

func (l *CustomLogger) LogPetDeath(isPlayer bool, pet FightActivePet) {
	if isPlayer {
		l.LogFn("【我方】%s[%d] 战败了！", pet.Name, pet.ID)
	} else {
		l.LogFn("（敌方）%s[%d] 战败了！", pet.Name, pet.ID)
	}
}

func (l *CustomLogger) LogSkillResult(isPlayer bool, pet FightActivePet, result SkillResult) {
	var sb strings.Builder

	if isPlayer {
		sb.WriteString("【我方】")
	} else {
		sb.WriteString("（敌方）")
	}

	fmt.Fprintf(&sb, "%s<%d/%d> ", pet.Name, pet.HP, pet.MaxHP)

	if result.Skill.ID == 0 {
		sb.WriteString("未使用技能")
	} else {
		var crit string
		if result.IsCrit {
			crit = "打出了致命一击"
		}

		fmt.Fprintf(
			&sb,
			"使用了 %s[%d] %s造成了 %d 点伤害",
			result.Skill.Name,
			result.Skill.ID,
			crit,
			result.Damage,
		)
	}

	l.LogFn("%s", sb.String())
}

func (l *CustomLogger) LogItemResult(isPlayer bool, itemID uint32) {
	itemName := "???"
	var sb strings.Builder

	if isPlayer {
		sb.WriteString("【我方】")
	} else {
		sb.WriteString("（敌方）")
	}

	if client := seerapi.Default(); client != nil {
		if name := client.ItemName(itemID); name != "" {
			itemName = name
		}
	}

	fmt.Fprintf(&sb, "使用了 %s[%d]", itemName, itemID)

	l.LogFn("%s", sb.String())
}

func (l *CustomLogger) LogFightOver(win bool, escape bool) {
	if escape {
		l.LogFn("战斗结束，我方撤退\n\n")
		return
	}

	if win {
		l.LogFn("战斗结束，我方胜利\n\n")
	} else {
		l.LogFn("战斗结束，我方失败\n\n")
	}
}
