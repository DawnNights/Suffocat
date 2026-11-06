package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"suffocat/internal/config"
	"suffocat/internal/seerapi"
)

// ---------- 持久化结构 ----------

type FactorStageConfig struct {
	Port string `json:"port"` // UNITY / FLASH
	Plan string `json:"plan"` // 相对于 data/scripts 的方案路径
}

type FactorSettings struct {
	Mode      int                 `json:"mode"` // 1 简单 / 2 普通 / 3 困难
	Speedrun  bool                `json:"speedrun"`
	FailTimes int                 `json:"failTimes"`
	Stages    []FactorStageConfig `json:"stages"`
}

type factorState struct {
	Enabled  map[string]bool            `json:"enabled"`
	Settings map[string]*FactorSettings `json:"settings"`
}

var (
	factorStateMu sync.Mutex
)

func factorStatePath() string {
	return filepath.Join(config.ConfigDir(), "factors.json")
}

func factorKey(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

func loadFactorState() factorState {
	state := factorState{
		Enabled:  map[string]bool{},
		Settings: map[string]*FactorSettings{},
	}

	data, err := os.ReadFile(factorStatePath())
	if err != nil {
		return state
	}
	if json.Unmarshal(data, &state) != nil {
		return factorState{Enabled: map[string]bool{}, Settings: map[string]*FactorSettings{}}
	}
	if state.Enabled == nil {
		state.Enabled = map[string]bool{}
	}
	if state.Settings == nil {
		state.Settings = map[string]*FactorSettings{}
	}
	return state
}

func saveFactorState(state factorState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := factorStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---------- 因子数据 ----------

type factorItem struct {
	ID         uint32 `json:"id"` // 因子关卡 ID，沿用旧配置中的设计 ID
	Name       string `json:"name"`
	FactorName string `json:"factorName"`
	Mode       int    `json:"mode"`
	Enabled    bool   `json:"enabled"`
	Desc       string `json:"desc"`
	Stages     int    `json:"stages"`
}

func modeLabel(mode int) string {
	switch mode {
	case 2:
		return "普通"
	case 3:
		return "困难"
	default:
		return "简单"
	}
}

func factorDesign(id uint32) (*seerapi.FactorDesign, bool) {
	client := seerapi.Default()
	if client == nil {
		return nil, false
	}
	design, err := client.FactorDesign(id)
	if err != nil {
		return nil, false
	}
	return &design, true
}

func factorForDesign(id uint32) (*seerapi.Factor, bool) {
	client := seerapi.Default()
	if client == nil {
		return nil, false
	}
	factor, ok := client.FactorByDesign(id)
	if !ok {
		return nil, false
	}
	return &factor, true
}

func factorName(factor *seerapi.Factor) string {
	if factor == nil {
		return ""
	}
	name := strings.TrimSpace(factor.Name)
	name = strings.TrimSpace(strings.TrimSuffix(name, "因子"))
	if name == "" {
		name = fmt.Sprintf("因子%d", factor.ID)
	}
	return name
}

func factorDisplayName(name string, mode int) string {
	return fmt.Sprintf("%s-%s", modeLabel(mode), name)
}

func factorDifficulty(design *seerapi.FactorDesign, mode int) *seerapi.FactorDifficulty {
	if design == nil {
		return nil
	}
	return design.Difficulty(mode)
}

func normalizedFactorSettings(design *seerapi.FactorDesign, saved *FactorSettings) *FactorSettings {
	result := &FactorSettings{Mode: 1, FailTimes: 10}
	if saved != nil {
		result.Mode = saved.Mode
		result.Speedrun = saved.Speedrun
		result.FailTimes = saved.FailTimes
	}
	if result.Mode < 1 || result.Mode > 3 {
		result.Mode = 1
	}

	count := 0
	if difficulty := factorDifficulty(design, result.Mode); difficulty != nil {
		count = len(difficulty.Tasks)
	}
	result.Stages = make([]FactorStageConfig, count)
	for i := 0; i < count; i++ {
		if saved != nil && i < len(saved.Stages) {
			result.Stages[i] = saved.Stages[i]
		}
		if result.Stages[i].Port != "FLASH" {
			result.Stages[i].Port = "UNITY"
		}
	}
	return result
}

func buildFactorDesc(design *seerapi.FactorDesign, mode int) string {
	difficulty := factorDifficulty(design, mode)
	if difficulty == nil {
		return ""
	}

	var lines []string
	for i, rule := range difficulty.Rules {
		if rule.Tips == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, rule.Tips))
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}

	for i, task := range difficulty.Tasks {
		lines = append(lines, fmt.Sprintf("第 %d 关%s", i+1, task.Desc))
	}
	return strings.Join(lines, "\n")
}

func listFactors() []factorItem {
	client := seerapi.Default()
	if client == nil {
		return []factorItem{}
	}

	if !client.HasFactorData() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err := client.EnsureFactorData(ctx, false)
		cancel()
		if err != nil {
			studioLog("ERROR", "因子数据获取失败: "+err.Error())
		}
	}

	factors := client.Factors()
	designs := client.FactorDesigns()
	if len(factors) == 0 || len(designs) == 0 {
		return []factorItem{}
	}

	state := loadFactorState()
	designMap := make(map[uint32]*seerapi.FactorDesign, len(designs))
	for i := range designs {
		designMap[designs[i].ID] = &designs[i]
	}

	result := make([]factorItem, 0, len(designs))
	for i := range factors {
		factor := &factors[i]
		if factor.DesignID == 0 {
			continue
		}
		design, ok := designMap[factor.DesignID]
		if !ok {
			continue
		}

		key := factorKey(factor.DesignID)
		settings := normalizedFactorSettings(design, state.Settings[key])
		name := factorName(factor)
		result = append(result, factorItem{
			ID:         factor.DesignID,
			Name:       factorDisplayName(name, settings.Mode),
			FactorName: name,
			Mode:       settings.Mode,
			Enabled:    state.Enabled[key],
			Desc:       buildFactorDesc(design, settings.Mode),
			Stages:     len(settings.Stages),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})
	return result
}

func factorProgressKey(id uint32) string {
	return fmt.Sprintf("factor:%d", id)
}

// ---------- HTTP ----------

func apiFactors(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, listFactors())
}

func apiSetFactorEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	id, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id 参数无效")
		return
	}
	if _, ok := factorDesign(uint32(id)); !ok {
		writeError(w, http.StatusNotFound, "未找到对应的因子")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	factorStateMu.Lock()
	state := loadFactorState()
	state.Enabled[factorKey(uint32(id))] = body.Enabled
	err = saveFactorState(state)
	factorStateMu.Unlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

type factorSettingsResp struct {
	Mode       int                 `json:"mode"`
	Speedrun   bool                `json:"speedrun"`
	FailTimes  int                 `json:"failTimes"`
	Stages     []FactorStageConfig `json:"stages"`
	ModeStages map[string]int      `json:"modeStages"`
	Desc       string              `json:"desc"`
	Name       string              `json:"name"`
	FactorName string              `json:"factorName"`
}

func apiFactorSettings(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id 参数无效")
		return
	}

	design, ok := factorDesign(uint32(id))
	if !ok {
		writeError(w, http.StatusNotFound, "未找到对应的因子")
		return
	}
	factor, ok := factorForDesign(uint32(id))
	if !ok {
		writeError(w, http.StatusNotFound, "未找到对应的因子道具")
		return
	}
	name := factorName(factor)

	switch r.Method {
	case http.MethodGet:
		state := loadFactorState()
		settings := normalizedFactorSettings(design, state.Settings[factorKey(uint32(id))])
		writeJSON(w, factorSettingsResp{
			Mode:      settings.Mode,
			Speedrun:  settings.Speedrun,
			FailTimes: settings.FailTimes,
			Stages:    settings.Stages,
			ModeStages: map[string]int{
				"1": stageCount(design, 1),
				"2": stageCount(design, 2),
				"3": stageCount(design, 3),
			},
			Desc:       buildFactorDesc(design, settings.Mode),
			Name:       factorDisplayName(name, settings.Mode),
			FactorName: name,
		})

	case http.MethodPost:
		var body FactorSettings
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "请求格式错误")
			return
		}

		settings := normalizedFactorSettings(design, &body)

		factorStateMu.Lock()
		state := loadFactorState()
		state.Settings[factorKey(uint32(id))] = settings
		err := saveFactorState(state)
		factorStateMu.Unlock()

		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, map[string]any{
			"ok":         true,
			"name":       factorDisplayName(name, settings.Mode),
			"factorName": name,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
	}
}

func stageCount(design *seerapi.FactorDesign, mode int) int {
	if difficulty := factorDifficulty(design, mode); difficulty != nil {
		return len(difficulty.Tasks)
	}
	return 0
}
