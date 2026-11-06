package seerapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.seerapi.com"

var ErrNotFound = errors.New("seerapi: not found")

type Client struct {
	baseURL  string
	http     *http.Client
	cacheDir string
	db       *sql.DB

	pets          *tableStore[Pet]
	skills        *tableStore[Skill]
	items         *tableStore[Item]
	mintmarks     *tableStore[MintMark]
	suits         *tableStore[Suit]
	equips        *tableStore[Equip]
	factors       *tableStore[Factor]
	factorDesigns *tableStore[FactorDesign]

	inflightMu sync.Mutex
	inflight   map[string]*inflightCall
	missingMu  sync.Mutex
	missing    map[string]time.Time

	indexMu   sync.Mutex
	syncMu    sync.Mutex
	factorMu  sync.Mutex
	factorErr error
	factorTry time.Time
}

type inflightCall struct {
	done chan struct{}
	err  error
}

var (
	defaultClient *Client
	defaultMu     sync.RWMutex
)

func SetDefault(c *Client) {
	defaultMu.Lock()
	defaultClient = c
	defaultMu.Unlock()
}

func Default() *Client {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultClient
}

func New(cacheDir string) (*Client, error) {
	db, err := openDB(filepath.Join(cacheDir, "seerapi.db"))
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:  defaultBaseURL,
		cacheDir: cacheDir,
		db:       db,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
		inflight: make(map[string]*inflightCall),
		missing:  make(map[string]time.Time),
	}

	stores := newStores(db)
	c.pets = stores.pets
	c.skills = stores.skills
	c.items = stores.items
	c.mintmarks = stores.mintmarks
	c.suits = stores.suits
	c.equips = stores.equips
	c.factors = stores.factors
	c.factorDesigns = stores.factorDesigns

	// 兼容旧版 JSONL 缓存：仅在对应表为空时导入。
	_ = migrateJSONL(filepath.Join(cacheDir, "pets.jsonl"), c.pets)
	_ = migrateJSONL(filepath.Join(cacheDir, "skills.jsonl"), c.skills)
	_ = migrateJSONL(filepath.Join(cacheDir, "items.jsonl"), c.items)
	_ = migrateJSONL(filepath.Join(cacheDir, "mintmarks.jsonl"), c.mintmarks)
	_ = migrateJSONL(filepath.Join(cacheDir, "suits.jsonl"), c.suits)
	_ = migrateJSONL(filepath.Join(cacheDir, "equips.jsonl"), c.equips)

	// 旧缓存已带名称时，标记名称索引为刚刚更新，避免首次列表请求重复下载。
	if c.pets.NameCount() > 0 && c.metaValue(metaPetIndexTime) == "" {
		_ = c.setMeta(metaPetIndexTime, strconv.FormatInt(time.Now().Unix(), 10))
	}

	return c, nil
}

func (c *Client) Close() {
	if c == nil || c.db == nil {
		return
	}
	c.db.Close()
}

// ---------- HTTP ----------

func (c *Client) fetchJSON(path string, out any) error {
	return c.fetchJSONContext(context.Background(), path, out)
}

func (c *Client) fetchJSONContext(ctx context.Context, path string, out any) error {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Suffocat/1.0")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			trimmed := bytes.TrimSpace(body)
			if len(trimmed) > 0 && trimmed[0] == '<' {
				// 部分 CDN 在触发限流/风控时会以 200 返回 HTML 页面
				lastErr = fmt.Errorf("seerapi %s: 返回了非 JSON 内容（可能触发限流）", path)
				continue
			}
			if err := json.Unmarshal(body, out); err != nil {
				lastErr = fmt.Errorf("解析响应失败 %s: %w", path, err)
				continue
			}
			return nil
		case resp.StatusCode == http.StatusNotFound:
			return ErrNotFound
		case resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= 500:
			lastErr = fmt.Errorf("seerapi %s: HTTP %d", path, resp.StatusCode)
			continue
		default:
			return fmt.Errorf("seerapi %s: HTTP %d: %s", path, resp.StatusCode, string(body))
		}
	}

	return lastErr
}

func (c *Client) runOnce(key string, fn func() error) error {
	c.inflightMu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.inflightMu.Unlock()
		<-call.done
		return call.err
	}
	call := &inflightCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.inflightMu.Unlock()

	call.err = fn()
	close(call.done)

	c.inflightMu.Lock()
	delete(c.inflight, key)
	c.inflightMu.Unlock()
	return call.err
}

func (c *Client) markMissing(key string) {
	c.missingMu.Lock()
	c.missing[key] = time.Now()
	c.missingMu.Unlock()
}

func (c *Client) isMissing(key string) bool {
	c.missingMu.Lock()
	defer c.missingMu.Unlock()

	at, ok := c.missing[key]
	if !ok {
		return false
	}
	if time.Since(at) > 10*time.Minute {
		delete(c.missing, key)
		return false
	}
	return true
}

func getOrFetch[T any](c *Client, id uint32, key string, store *tableStore[T], fetch func() (T, error)) (T, error) {
	if v, ok := store.Get(id); ok {
		return v, nil
	}
	if c.isMissing(key) {
		var zero T
		return zero, ErrNotFound
	}

	var out T
	err := c.runOnce(key, func() error {
		if v, ok := store.Get(id); ok {
			out = v
			return nil
		}
		v, err := fetch()
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				c.markMissing(key)
			}
			return err
		}
		if err := store.Put(v); err != nil {
			return err
		}
		out = v
		return nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	if v, ok := store.Get(id); ok {
		return v, nil
	}
	return out, nil
}

// ---------- API 原始结构 ----------

type apiStats struct {
	Atk   int `json:"atk"`
	Def   int `json:"def"`
	SpAtk int `json:"sp_atk"`
	SpDef int `json:"sp_def"`
	Spd   int `json:"spd"`
	HP    int `json:"hp"`
}

type apiPet struct {
	ID         uint32 `json:"id"`
	Name       string `json:"name"`
	ResourceID uint32 `json:"resource_id"`
	CatchRate  int    `json:"catch_rate"`
	Type       *struct {
		ID uint32 `json:"id"`
	} `json:"type"`
	Gender *struct {
		ID uint32 `json:"id"`
	} `json:"gender"`
	BaseStats *apiStats `json:"base_stats"`
	Skill     []struct {
		LearningLevel uint32 `json:"learning_level"`
		IsSpecial     bool   `json:"is_special"`
		IsAdvanced    bool   `json:"is_advanced"`
		IsFifth       bool   `json:"is_fifth"`
		Skill         *struct {
			ID uint32 `json:"id"`
		} `json:"skill"`
	} `json:"skill"`
	Soulmark []struct {
		ID uint32 `json:"id"`
	} `json:"soulmark"`
}

type apiSkill struct {
	ID       uint32  `json:"id"`
	Name     string  `json:"name"`
	Power    uint32  `json:"power"`
	MaxPP    uint32  `json:"max_pp"`
	Accuracy uint32  `json:"accuracy"`
	Priority int     `json:"priority"`
	CritRate float64 `json:"crit_rate"`
	MustHit  bool    `json:"must_hit"`
	Category *struct {
		ID uint32 `json:"id"`
	} `json:"category"`
	Type *struct {
		ID uint32 `json:"id"`
	} `json:"type"`
}

type apiItem struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type apiMintMark struct {
	ID    uint32 `json:"id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Class *struct {
		ID uint32 `json:"id"`
	} `json:"mintmark_class"`
	ExtraAttrValue *apiStats `json:"extra_attr_value"`
}

type apiSuit struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Equips []struct {
		ID uint32 `json:"id"`
	} `json:"equips"`
}

type apiEquip struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Part *struct {
		ID uint32 `json:"id"`
	} `json:"part_type"`
}

// ---------- 对外查询 ----------

func (c *Client) Pet(id uint32) (Pet, error) {
	return getOrFetch(c, id, fmt.Sprintf("pet:%d", id), c.pets, func() (Pet, error) {
		return c.fetchPet(id)
	})
}

func (c *Client) fetchPet(id uint32) (Pet, error) {
	var raw apiPet
	if err := c.fetchJSON("/v1/pet/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
		return Pet{}, err
	}
	return raw.toPet(), nil
}

func (raw apiPet) toPet() Pet {
	pet := Pet{
		ID:         raw.ID,
		Name:       raw.Name,
		ResourceID: raw.ResourceID,
		CatchRate:  raw.CatchRate,
	}
	if raw.Type != nil {
		pet.Type = raw.Type.ID
	}
	if raw.Gender != nil {
		pet.Gender = raw.Gender.ID
	}
	if raw.BaseStats != nil {
		pet.BaseStats = toStats(*raw.BaseStats)
	}
	for _, ref := range raw.Skill {
		if ref.Skill == nil || ref.Skill.ID == 0 {
			continue
		}
		pet.Skills = append(pet.Skills, PetSkill{
			ID:       ref.Skill.ID,
			Level:    ref.LearningLevel,
			Special:  ref.IsSpecial,
			Advanced: ref.IsAdvanced,
			Fifth:    ref.IsFifth,
		})
	}
	for _, sm := range raw.Soulmark {
		pet.SoulmarkIDs = append(pet.SoulmarkIDs, sm.ID)
	}
	return pet
}

func (c *Client) Skill(id uint32) (Skill, error) {
	return getOrFetch(c, id, fmt.Sprintf("skill:%d", id), c.skills, func() (Skill, error) {
		var raw apiSkill
		if err := c.fetchJSON("/v1/skill/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return Skill{}, err
		}
		return raw.toSkill(), nil
	})
}

func (raw apiSkill) toSkill() Skill {
	skill := Skill{
		ID:       raw.ID,
		Name:     raw.Name,
		Power:    raw.Power,
		MaxPP:    raw.MaxPP,
		Accuracy: raw.Accuracy,
		Priority: raw.Priority,
		CritRate: raw.CritRate,
		MustHit:  raw.MustHit,
	}
	if raw.Category != nil {
		skill.Category = raw.Category.ID
	}
	if raw.Type != nil {
		skill.Type = raw.Type.ID
	}
	return skill
}

func (c *Client) Item(id uint32) (Item, error) {
	return getOrFetch(c, id, fmt.Sprintf("item:%d", id), c.items, func() (Item, error) {
		var raw apiItem
		if err := c.fetchJSON("/v1/item/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return Item{}, err
		}
		return raw.toItem(), nil
	})
}

func (raw apiItem) toItem() Item {
	return Item{ID: raw.ID, Name: raw.Name, Desc: raw.Desc}
}

func (c *Client) MintMark(id uint32) (MintMark, error) {
	return getOrFetch(c, id, fmt.Sprintf("mintmark:%d", id), c.mintmarks, func() (MintMark, error) {
		var raw apiMintMark
		if err := c.fetchJSON("/v1/mintmark/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return MintMark{}, err
		}
		return raw.toMintMark(), nil
	})
}

func (raw apiMintMark) toMintMark() MintMark {
	mark := MintMark{ID: raw.ID, Name: raw.Name, Desc: raw.Desc}
	if raw.Class != nil {
		mark.ClassID = raw.Class.ID
	}
	if raw.ExtraAttrValue != nil {
		mark.ExtraAttr = toStats(*raw.ExtraAttrValue)
	}
	return mark
}

func (c *Client) Suit(id uint32) (Suit, error) {
	return getOrFetch(c, id, fmt.Sprintf("suit:%d", id), c.suits, func() (Suit, error) {
		var raw apiSuit
		if err := c.fetchJSON("/v1/suit/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return Suit{}, err
		}
		return raw.toSuit(), nil
	})
}

func (raw apiSuit) toSuit() Suit {
	suit := Suit{ID: raw.ID, Name: raw.Name}
	for _, e := range raw.Equips {
		suit.EquipIDs = append(suit.EquipIDs, e.ID)
	}
	return suit
}

func (c *Client) Equip(id uint32) (Equip, error) {
	return getOrFetch(c, id, fmt.Sprintf("equip:%d", id), c.equips, func() (Equip, error) {
		var raw apiEquip
		if err := c.fetchJSON("/v1/equip/"+strconv.FormatUint(uint64(id), 10), &raw); err != nil {
			return Equip{}, err
		}
		return raw.toEquip(), nil
	})
}

func (raw apiEquip) toEquip() Equip {
	equip := Equip{ID: raw.ID, Name: raw.Name}
	if raw.Part != nil {
		equip.Part = raw.Part.ID
	}
	return equip
}

func toStats(s apiStats) Stats {
	return Stats{
		Atk:   s.Atk,
		Def:   s.Def,
		SpAtk: s.SpAtk,
		SpDef: s.SpDef,
		Spd:   s.Spd,
		HP:    s.HP,
	}
}

// ---------- 便捷接口 ----------

func (c *Client) PetName(id uint32) string {
	if c == nil {
		return ""
	}
	// 名称索引优先，避免为了取名字而拉取整只精灵的详情。
	if name, ok := c.pets.Name(id); ok {
		return name
	}
	pet, err := c.Pet(id)
	if err != nil {
		return ""
	}
	return pet.Name
}

// CachedPetName 只读取本地名称索引，不触发网络请求。
func (c *Client) CachedPetName(id uint32) (string, bool) {
	if c == nil {
		return "", false
	}
	return c.pets.Name(id)
}

func (c *Client) ItemName(id uint32) string {
	if c == nil {
		return ""
	}
	item, err := c.Item(id)
	if err != nil {
		return ""
	}
	return item.Name
}

func (c *Client) SkillInfo(id uint32) (Skill, bool) {
	if c == nil {
		return Skill{}, false
	}
	skill, err := c.Skill(id)
	return skill, err == nil
}

// EffectIDForPet 返回精灵魂印 ID。
// 游戏内精灵 EffectList 中记录的魂印 EffectID 与魂印 ID 一致。
func (c *Client) EffectIDForPet(id uint32) uint32 {
	if c == nil {
		return 0
	}
	pet, err := c.Pet(id)
	if err != nil || len(pet.SoulmarkIDs) == 0 {
		return 0
	}
	return pet.SoulmarkIDs[0]
}

// PetHasMove 判断精灵是否可以学习指定技能。
func (c *Client) PetHasMove(petID, moveID uint32) bool {
	if c == nil {
		return false
	}
	pet, err := c.Pet(petID)
	if err != nil {
		return false
	}
	for _, s := range pet.Skills {
		if s.ID == moveID {
			return true
		}
	}
	return false
}

// PetSkills 返回精灵可学习技能的详细信息，顺序与数据源一致。
func (c *Client) PetSkills(petID uint32) []Skill {
	if c == nil {
		return nil
	}
	pet, err := c.Pet(petID)
	if err != nil {
		return nil
	}

	skills := make([]Skill, 0, len(pet.Skills))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)

	for _, ref := range pet.Skills {
		ref := ref
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			skill, err := c.Skill(ref.ID)
			if err != nil {
				return
			}
			mu.Lock()
			skills = append(skills, skill)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// 并发完成后按数据源顺序还原
	byID := make(map[uint32]Skill, len(skills))
	for _, s := range skills {
		byID[s.ID] = s
	}
	ordered := make([]Skill, 0, len(skills))
	for _, ref := range pet.Skills {
		if s, ok := byID[ref.ID]; ok {
			ordered = append(ordered, s)
		}
	}
	return ordered
}

func (c *Client) ClothBySuitID(suitID uint32) Cloth {
	var cloth Cloth
	if c == nil {
		return cloth
	}

	suit, err := c.Suit(suitID)
	if err != nil {
		return cloth
	}
	cloth.Name = suit.Name
	cloth.ID = suit.ID

	for _, equipID := range suit.EquipIDs {
		equip, err := c.Equip(equipID)
		if err != nil {
			continue
		}

		switch equip.Part {
		case 0:
			cloth.Head = equipID
		case 1:
			cloth.Face = equipID
		case 2:
			cloth.Waist = equipID
		case 3:
			cloth.Body = equipID
		case 4:
			cloth.Foot = equipID
		}
	}

	return cloth
}

func (c *Client) SuitEquipIDs(suitID uint32) []uint32 {
	if c == nil {
		return nil
	}
	suit, err := c.Suit(suitID)
	if err != nil {
		return nil
	}
	return suit.EquipIDs
}

// AllPetNames 返回以精灵 ID 为下标的名字数组，未知精灵为空串。
// 名称来自轻量索引，不要求本地拥有精灵详情。
func (c *Client) AllPetNames() []string {
	if c == nil {
		return []string{}
	}

	names := c.pets.Names()
	if len(names) == 0 {
		if err := c.EnsurePetIndex(context.Background(), false); err == nil {
			names = c.pets.Names()
		}
	}
	if len(names) == 0 {
		return []string{}
	}

	var maxID uint32
	for id := range names {
		if id > maxID {
			maxID = id
		}
	}

	result := make([]string, maxID+1)
	for id, name := range names {
		result[id] = name
	}
	return result
}

func (c *Client) PetCacheCount() int {
	if c == nil {
		return 0
	}
	return c.pets.Count()
}

func (c *Client) PetIndexCount() int {
	if c == nil {
		return 0
	}
	return c.pets.NameCount()
}

// HasPetIndex 表示本地是否已有精灵名称索引。
func (c *Client) HasPetIndex() bool {
	if c == nil {
		return false
	}
	return c.pets.NameCount() > 0
}

// BaseURL 便于调试或代理。
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) SetBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	c.baseURL = u.String()
	return nil
}
