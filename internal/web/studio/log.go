package studio

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"suffocat/internal/lua/fight"
	"suffocat/internal/seerapi"
	"suffocat/internal/web"
)

// ---------- 日志缓冲 ----------

var (
	logBuf        []LogEntry
	logBufMu      sync.Mutex
	fightLogBuf   []LogEntry
	fightBufMu    sync.Mutex
	progressBuf   []progressMsg
	progressBufMu sync.Mutex
)

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type progressMsg struct {
	Index  int    `json:"index"`
	File   string `json:"file"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Total  int    `json:"total"`
}

func appendLogBuf(level, msg string) {
	logBufMu.Lock()
	defer logBufMu.Unlock()
	const maxLogs = 300
	if len(logBuf) >= maxLogs {
		logBuf = logBuf[maxLogs/2:]
	}
	logBuf = append(logBuf, LogEntry{Level: level, Message: msg})
}

func clearLogBuf() {
	logBufMu.Lock()
	defer logBufMu.Unlock()
	logBuf = nil
}

func appendFightLogBuf(level, msg string) {
	fightBufMu.Lock()
	defer fightBufMu.Unlock()
	const maxLogs = 300
	if len(fightLogBuf) >= maxLogs {
		fightLogBuf = fightLogBuf[maxLogs/2:]
	}
	fightLogBuf = append(fightLogBuf, LogEntry{Level: level, Message: msg})
}

func clearFightLogBuf() {
	fightBufMu.Lock()
	defer fightBufMu.Unlock()
	fightLogBuf = nil
}

func appendProgress(p progressMsg) {
	progressBufMu.Lock()
	defer progressBufMu.Unlock()
	for i, existing := range progressBuf {
		if existing.File == p.File {
			progressBuf[i] = p
			return
		}
	}
	progressBuf = append(progressBuf, p)
}

func clearProgressBuf() {
	progressBufMu.Lock()
	defer progressBufMu.Unlock()
	progressBuf = nil
}

// ---------- 日志发送 ----------

func studioLog(level, msg string) {
	cstZone := time.FixedZone("CST", 8*3600)
	ts := time.Now().In(cstZone).Format("15:04:05")
	fullMsg := fmt.Sprintf("[%s] %s", ts, msg)

	entry := LogEntry{Level: level, Message: fullMsg}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return
	}

	if level == "FIGHT" {
		appendFightLogBuf(level, fullMsg)
		web.WsSend("studio:fight", string(encoded))
	} else {
		appendLogBuf(level, fullMsg)
		web.WsSend("studio:log", string(encoded))
	}
}

func mustJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"marshal failed: %v"}`, err)
	}
	return string(encoded)
}

// ---------- 状态查询 ----------

type statusResp struct {
	Running   bool          `json:"running"`
	Logs      []LogEntry    `json:"logs"`
	FightLogs []LogEntry    `json:"fightLogs"`
	Progress  []progressMsg `json:"progress"`
	PetCache  int           `json:"petCache"`
	PetIndex  int           `json:"petIndex"`
	Syncing   bool          `json:"syncing"`
	SyncDone  int           `json:"syncDone"`
	SyncTotal int           `json:"syncTotal"`
}

func getStatus() statusResp {
	engineMu.Lock()
	running := engineRunning
	engineMu.Unlock()

	logBufMu.Lock()
	logs := make([]LogEntry, len(logBuf))
	copy(logs, logBuf)
	logBufMu.Unlock()

	fightBufMu.Lock()
	fightLogs := make([]LogEntry, len(fightLogBuf))
	copy(fightLogs, fightLogBuf)
	fightBufMu.Unlock()

	progressBufMu.Lock()
	progress := make([]progressMsg, len(progressBuf))
	copy(progress, progressBuf)
	progressBufMu.Unlock()

	syncMu.Lock()
	syncing := syncRunning
	syncDone := syncDoneCount
	syncTotal := syncTotalCount
	syncMu.Unlock()

	petCache := 0
	petIndex := 0
	if client := seerapi.Default(); client != nil {
		petCache = client.PetCacheCount()
		petIndex = client.PetIndexCount()
	}

	return statusResp{
		Running:   running,
		Logs:      logs,
		FightLogs: fightLogs,
		Progress:  progress,
		PetCache:  petCache,
		PetIndex:  petIndex,
		Syncing:   syncing,
		SyncDone:  syncDone,
		SyncTotal: syncTotal,
	}
}

// ---------- 战斗日志 ----------

type fightEvent struct {
	Type       string         `json:"type"`
	Round      uint32         `json:"round,omitempty"`
	IsPlayer   *bool          `json:"isPlayer,omitempty"`
	Pet        *fightPetData  `json:"pet,omitempty"`
	PlayerPets []fightPetData `json:"playerPets,omitempty"`
	EnemyPets  []fightPetData `json:"enemyPets,omitempty"`
	Skill      *skillData     `json:"skill,omitempty"`
	Item       *itemData      `json:"item,omitempty"`
	Win        *bool          `json:"win,omitempty"`
	Escape     *bool          `json:"escape,omitempty"`
}

type fightPetData struct {
	Name  string `json:"name"`
	HP    uint32 `json:"hp"`
	MaxHP uint32 `json:"maxHp"`
}

type skillData struct {
	Name       string `json:"name"`
	Damage     uint32 `json:"damage"`
	GainDamage uint32 `json:"gainDmg"`
	RealDamage uint32 `json:"realDmg"`
	ZhuiDamage uint32 `json:"zhuiDmg"`
	IsCrit     bool   `json:"isCrit"`
	PetHP      uint32 `json:"petHp"`
	PetMaxHP   uint32 `json:"petMaxHp"`
}

type itemData struct {
	Name string `json:"name"`
}

func sendFightEvent(e fightEvent) {
	encoded, err := json.Marshal(e)
	if err != nil {
		return
	}
	studioLog("FIGHT", string(encoded))
}

func boolPtr(b bool) *bool { return &b }

type WebFightLogger struct{}

func (l *WebFightLogger) LogFightStart(player fight.FightPlayer, enemy fight.FightEnemy) {
	playerPets := make([]fightPetData, 0, len(player.PetBag_))
	for _, pet := range player.PetBag_ {
		playerPets = append(playerPets, fightPetData{Name: pet.Name, HP: pet.HP, MaxHP: pet.MaxHP})
	}
	enemyPets := make([]fightPetData, 0, len(enemy.PetBag_))
	for _, pet := range enemy.PetBag_ {
		enemyPets = append(enemyPets, fightPetData{Name: pet.Name, HP: pet.HP, MaxHP: pet.MaxHP})
	}
	sendFightEvent(fightEvent{
		Type:       "start",
		PlayerPets: playerPets,
		EnemyPets:  enemyPets,
	})
}

func (l *WebFightLogger) LogRound(round uint32, _ fight.FightPlayer, _ fight.FightEnemy) {
	sendFightEvent(fightEvent{Type: "round", Round: round})
}

func (l *WebFightLogger) LogRoundEnd(_ uint32, _ fight.FightPlayer, _ fight.FightEnemy) {}

func (l *WebFightLogger) LogPetChange(isPlayer bool, pet fight.FightActivePet) {
	sendFightEvent(fightEvent{
		Type:     "pet_in",
		IsPlayer: boolPtr(isPlayer),
		Pet:      &fightPetData{Name: pet.Name, HP: pet.HP, MaxHP: pet.MaxHP},
	})
}

func (l *WebFightLogger) LogPetDeath(isPlayer bool, pet fight.FightActivePet) {
	sendFightEvent(fightEvent{
		Type:     "pet_out",
		IsPlayer: boolPtr(isPlayer),
		Pet:      &fightPetData{Name: pet.Name, HP: pet.HP, MaxHP: pet.MaxHP},
	})
}

func (l *WebFightLogger) LogSkillResult(isPlayer bool, pet fight.FightActivePet, result fight.SkillResult) {
	sendFightEvent(fightEvent{
		Type:     "skill",
		IsPlayer: boolPtr(isPlayer),
		Pet:      &fightPetData{Name: pet.Name},
		Skill: &skillData{
			Name:       result.Skill.Name,
			Damage:     result.Damage,
			GainDamage: result.GainDamage,
			RealDamage: result.RealDamage,
			ZhuiDamage: result.ZhuijiDamage,
			IsCrit:     result.IsCrit,
			PetHP:      pet.HP,
			PetMaxHP:   pet.MaxHP,
		},
	})
}

func (l *WebFightLogger) LogItemResult(isPlayer bool, itemID uint32) {
	itemName := "???"
	if client := seerapi.Default(); client != nil {
		if name := client.ItemName(itemID); name != "" {
			itemName = name
		}
	}
	sendFightEvent(fightEvent{
		Type:     "item",
		IsPlayer: boolPtr(isPlayer),
		Item:     &itemData{Name: itemName},
	})
}

func (l *WebFightLogger) LogFightOver(win bool, escape bool) {
	sendFightEvent(fightEvent{
		Type:   "over",
		Win:    boolPtr(win),
		Escape: boolPtr(escape),
	})
}
