package seerapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	metaFactorSyncTime      = "factor_sync_time"
	metaConfigVersion       = "game_config_version"
	metaConfigVersionLegacy = "factor_config_version"
	gameConfigVersion       = "https://seer.61.com/version/zzz_config.txt"
	factorPageSize          = 100
	defaultFactorTimes      = 3
)

type apiRef struct {
	ID uint32 `json:"id"`
}

type apiFactor struct {
	ID              uint32  `json:"id"`
	Name            string  `json:"name"`
	PetLimit        bool    `json:"pet_limit"`
	PetConsume      uint32  `json:"pet_consume"`
	SkillConsume    *uint32 `json:"skill_consume"`
	SoulmarkConsume *uint32 `json:"soulmark_consume"`
	Item            *apiRef `json:"item"`
	Design          *apiRef `json:"design"`
	NeedPet         *apiRef `json:"need_pet"`
	Pet             *apiRef `json:"pet"`
	Skill           *apiRef `json:"skill"`
	Soulmark        *apiRef `json:"soulmark"`
}

type apiFactorDesign struct {
	ID             uint32                `json:"id"`
	Times          uint32                `json:"times"`
	TimeValue      uint32                `json:"time_value"`
	ProgressValue  uint32                `json:"progress_value"`
	FailTimes      uint32                `json:"fail_times"`
	Achievement    *apiRef               `json:"achievement"`
	Difficulties   []apiFactorDifficulty `json:"difficulties"`
	Reward         *apiRef               `json:"reward"`
	MintmarkReward *apiRef               `json:"mintmark_reward"`
}

type apiFactorDifficulty struct {
	Level       string          `json:"level"`
	RewardCount uint32          `json:"reward_count"`
	Tasks       []apiFactorTask `json:"tasks"`
	BannedPets  []apiRef        `json:"banned_pets"`
	Rules       []apiFactorRule `json:"rules"`
}

type apiFactorTask struct {
	Desc        string  `json:"desc"`
	IsSinglePet bool    `json:"is_single_pet"`
	BossPet     *apiRef `json:"boss_pet"`
}

type apiFactorRule struct {
	ID       uint32 `json:"id"`
	Tips     string `json:"tips"`
	FailTips string `json:"fail_tips"`
}

type apiResultPage[T any] struct {
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Results []T    `json:"results"`
}

func (raw apiFactor) toFactor() Factor {
	factor := Factor{
		ID:         raw.ID,
		Name:       raw.Name,
		PetLimit:   raw.PetLimit,
		PetConsume: raw.PetConsume,
	}
	if raw.SkillConsume != nil {
		factor.SkillConsume = *raw.SkillConsume
	}
	if raw.SoulmarkConsume != nil {
		factor.SoulmarkConsume = *raw.SoulmarkConsume
	}
	if raw.Item != nil {
		factor.ItemID = raw.Item.ID
	}
	if raw.Design != nil {
		factor.DesignID = raw.Design.ID
	}
	if raw.NeedPet != nil {
		factor.NeedPetID = raw.NeedPet.ID
	}
	if raw.Pet != nil {
		factor.PetID = raw.Pet.ID
	}
	if raw.Skill != nil {
		factor.SkillID = raw.Skill.ID
	}
	if raw.Soulmark != nil {
		factor.SoulmarkID = raw.Soulmark.ID
	}
	return factor
}

func (raw apiFactorDesign) toFactorDesign() FactorDesign {
	design := FactorDesign{
		ID:            raw.ID,
		Times:         raw.Times,
		TimeValue:     raw.TimeValue,
		ProgressValue: raw.ProgressValue,
		FailTimes:     raw.FailTimes,
	}
	// 当前接口未返回 times，游戏内因子每日次数固定为 3。
	if design.Times == 0 {
		design.Times = defaultFactorTimes
	}
	if raw.Achievement != nil {
		design.AchievementID = raw.Achievement.ID
	}
	if raw.Reward != nil {
		design.RewardID = raw.Reward.ID
	}
	if raw.MintmarkReward != nil {
		design.MintmarkRewardID = raw.MintmarkReward.ID
	}

	design.Difficulties = make([]FactorDifficulty, 0, len(raw.Difficulties))
	for _, rawDifficulty := range raw.Difficulties {
		difficulty := FactorDifficulty{
			Level:       rawDifficulty.Level,
			RewardCount: rawDifficulty.RewardCount,
		}
		for _, rawTask := range rawDifficulty.Tasks {
			task := FactorTask{
				Desc:        rawTask.Desc,
				IsSinglePet: rawTask.IsSinglePet,
			}
			if rawTask.BossPet != nil {
				task.BossPetID = rawTask.BossPet.ID
			}
			difficulty.Tasks = append(difficulty.Tasks, task)
		}
		for _, rawRule := range rawDifficulty.Rules {
			difficulty.Rules = append(difficulty.Rules, FactorRule{
				ID:       rawRule.ID,
				Tips:     rawRule.Tips,
				FailTips: rawRule.FailTips,
			})
		}
		for _, banned := range rawDifficulty.BannedPets {
			difficulty.BannedPets = append(difficulty.BannedPets, banned.ID)
		}
		design.Difficulties = append(design.Difficulties, difficulty)
	}
	return design
}

// Factor 返回因子道具资源，本地缺失时按需请求 SeerAPI。
func (c *Client) Factor(id uint32) (Factor, error) {
	return getOrFetch(c, id, fmt.Sprintf("factor:%d", id), c.factors, func() (Factor, error) {
		var raw apiFactor
		if err := c.fetchJSON("/v1/factor/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return Factor{}, err
		}
		return raw.toFactor(), nil
	})
}

// FactorDesign 返回因子关卡资源，本地缺失时按需请求 SeerAPI。
func (c *Client) FactorDesign(id uint32) (FactorDesign, error) {
	return getOrFetch(c, id, fmt.Sprintf("factor_design:%d", id), c.factorDesigns, func() (FactorDesign, error) {
		var raw apiFactorDesign
		if err := c.fetchJSON("/v1/factor_design/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return FactorDesign{}, err
		}
		return raw.toFactorDesign(), nil
	})
}

// FactorByDesign 返回关联到指定关卡资源的因子道具。
func (c *Client) FactorByDesign(designID uint32) (Factor, bool) {
	if c == nil {
		return Factor{}, false
	}
	return c.factors.FirstBy("design_id", designID)
}

// Factors 返回本地缓存中的全部因子道具。
func (c *Client) Factors() []Factor {
	if c == nil {
		return nil
	}
	return c.factors.All()
}

// FactorDesigns 返回本地缓存中的全部因子关卡。
func (c *Client) FactorDesigns() []FactorDesign {
	if c == nil {
		return nil
	}
	return c.factorDesigns.All()
}

func (c *Client) HasFactorData() bool {
	return c != nil && c.factors.Count() > 0 && c.factorDesigns.Count() > 0
}

func (c *Client) FactorCount() int {
	if c == nil {
		return 0
	}
	return c.factors.Count()
}

// EnsureFactorData 确保本地存在因子数据。
// force 为 false 且已有数据时直接返回；版本判断由 EnsureGameData 负责。
func (c *Client) EnsureFactorData(ctx context.Context, force bool) error {
	if c == nil {
		return nil
	}

	c.factorMu.Lock()
	defer c.factorMu.Unlock()

	if !force && c.factors.Count() > 0 && c.factorDesigns.Count() > 0 {
		return nil
	}

	// 同步失败后 10 分钟内不重复请求，避免 SeerAPI 不可用时频繁重试。
	if !force && !c.factorTry.IsZero() && time.Since(c.factorTry) < 10*time.Minute {
		return c.factorErr
	}
	c.factorTry = time.Now()

	err := c.syncFactorData(ctx)
	c.factorErr = err
	if err != nil {
		return err
	}

	_ = c.setMeta(metaFactorSyncTime, strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

// EnsureGameData 在游戏配置版本变化时更新精灵与因子数据。
func (c *Client) EnsureGameData(ctx context.Context, progress func(done, total int)) error {
	if c == nil {
		return nil
	}

	version, err := c.fetchConfigVersion(ctx)
	if err != nil {
		return fmt.Errorf("检查游戏配置版本失败: %w", err)
	}
	if c.storedConfigVersion() == version && c.pets.Count() > 0 && c.HasFactorData() {
		return nil
	}

	c.syncMu.Lock()
	defer c.syncMu.Unlock()

	// 获取锁后再次确认，避免并发调用重复同步。
	if c.storedConfigVersion() == version && c.pets.Count() > 0 && c.HasFactorData() {
		return nil
	}

	if err := c.SyncMissingPetsFromDataRepo(ctx, progress); err != nil {
		return fmt.Errorf("更新精灵数据失败: %w", err)
	}

	if err := c.EnsureFactorData(ctx, true); err != nil {
		return fmt.Errorf("更新因子数据失败: %w", err)
	}

	return c.setConfigVersion(version)
}

// StoreConfigVersion 将当前游戏配置版本写入本地，用于显式同步后跳过下次启动的重复更新。
func (c *Client) StoreConfigVersion(ctx context.Context) error {
	if c == nil {
		return nil
	}
	version, err := c.fetchConfigVersion(ctx)
	if err != nil {
		return err
	}
	return c.setConfigVersion(version)
}

func (c *Client) storedConfigVersion() string {
	if version := c.metaValue(metaConfigVersion); version != "" {
		return version
	}
	version := c.metaValue(metaConfigVersionLegacy)
	if version == "" {
		return ""
	}
	_ = c.setMeta(metaConfigVersion, version)
	_, _ = c.db.Exec("DELETE FROM meta WHERE key = ?", metaConfigVersionLegacy)
	return version
}

func (c *Client) setConfigVersion(version string) error {
	if err := c.setMeta(metaConfigVersion, version); err != nil {
		return err
	}
	_, _ = c.db.Exec("DELETE FROM meta WHERE key = ?", metaConfigVersionLegacy)
	return nil
}

func (c *Client) fetchConfigVersion(ctx context.Context) (string, error) {
	requestURL := fmt.Sprintf("%s?%d", gameConfigVersion, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Suffocat/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("配置版本为空")
	}
	return version, nil
}

func (c *Client) syncFactorData(ctx context.Context) error {
	rawFactors, err := fetchAllPages[apiFactor](c, ctx, "/v1/factor", true)
	if err != nil {
		return fmt.Errorf("获取因子列表失败: %w", err)
	}
	rawDesigns, err := fetchAllPages[apiFactorDesign](c, ctx, "/v1/factor_design", true)
	if err != nil {
		return fmt.Errorf("获取因子关卡列表失败: %w", err)
	}

	factors := make([]Factor, 0, len(rawFactors))
	for _, raw := range rawFactors {
		factor := raw.toFactor()
		if factor.ID != 0 {
			factors = append(factors, factor)
		}
	}
	designs := make([]FactorDesign, 0, len(rawDesigns))
	for _, raw := range rawDesigns {
		design := raw.toFactorDesign()
		if design.ID != 0 {
			designs = append(designs, design)
		}
	}
	if len(factors) == 0 || len(designs) == 0 {
		return fmt.Errorf("因子数据为空")
	}

	return c.replaceFactorData(factors, designs)
}

func fetchAllPages[T any](c *Client, ctx context.Context, path string, expand bool) ([]T, error) {
	var all []T
	total := -1

	for offset := 0; ; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		query := fmt.Sprintf("%s?offset=%d&limit=%d&expand=%t", path, offset, factorPageSize, expand)
		var page apiResultPage[T]
		if err := c.fetchJSONContext(ctx, query, &page); err != nil {
			return nil, err
		}
		if total < 0 {
			total = page.Count
		}
		all = append(all, page.Results...)

		if len(page.Results) == 0 || len(all) >= total {
			break
		}
		offset += len(page.Results)
	}

	if total >= 0 && len(all) != total {
		return nil, fmt.Errorf("数据分页不完整: 已获取 %d/%d", len(all), total)
	}
	return all, nil
}

func (c *Client) replaceFactorData(factors []Factor, designs []FactorDesign) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{
		"factor_tasks",
		"factor_rules",
		"factor_banned_pets",
		"factor_difficulties",
		"factor_designs",
		"factors",
	} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}

	now := time.Now().Unix()
	if err := bulkPutTx(tx, c.factors, factors, nil, now); err != nil {
		return fmt.Errorf("写入因子数据失败: %w", err)
	}
	if err := bulkPutTx(tx, c.factorDesigns, designs, nil, now); err != nil {
		return fmt.Errorf("写入因子关卡数据失败: %w", err)
	}

	return tx.Commit()
}
