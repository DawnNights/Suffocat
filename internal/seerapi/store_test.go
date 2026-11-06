package seerapi

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := openDB(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPetStoreRoundTrip(t *testing.T) {
	store := newPetStore(openTestDB(t))

	pet := Pet{
		ID:          1,
		Name:        "测试精灵",
		ResourceID:  2,
		CatchRate:   3,
		Type:        4,
		Gender:      5,
		BaseStats:   Stats{Atk: 1, Def: 2, SpAtk: 3, SpDef: 4, Spd: 5, HP: 6},
		Skills:      []PetSkill{{ID: 10, Level: 1, Special: true}, {ID: 20, Level: 2, Advanced: true, Fifth: true}},
		SoulmarkIDs: []uint32{100, 200},
	}

	if err := store.Put(pet); err != nil {
		t.Fatalf("写入精灵失败: %v", err)
	}

	got, ok := store.Get(pet.ID)
	if !ok {
		t.Fatal("读取精灵失败")
	}
	if !reflect.DeepEqual(got, pet) {
		t.Fatalf("精灵数据不一致:\n got=%+v\nwant=%+v", got, pet)
	}

	// 更新时应替换子表，而不是累积旧技能。
	pet.Skills = []PetSkill{{ID: 30, Level: 9}}
	pet.SoulmarkIDs = []uint32{300}
	if err := store.Put(pet); err != nil {
		t.Fatalf("更新精灵失败: %v", err)
	}
	got, _ = store.Get(pet.ID)
	if !reflect.DeepEqual(got, pet) {
		t.Fatalf("更新后数据不一致:\n got=%+v\nwant=%+v", got, pet)
	}

	if count := store.Count(); count != 1 {
		t.Fatalf("精灵数量错误: %d", count)
	}
}

func TestSuitStoreRoundTrip(t *testing.T) {
	store := newSuitStore(openTestDB(t))

	suit := Suit{ID: 7, Name: "测试套装", EquipIDs: []uint32{101, 102, 103}}
	if err := store.Put(suit); err != nil {
		t.Fatalf("写入套装失败: %v", err)
	}

	got, ok := store.Get(suit.ID)
	if !ok {
		t.Fatal("读取套装失败")
	}
	if !reflect.DeepEqual(got, suit) {
		t.Fatalf("套装数据不一致: got=%+v want=%+v", got, suit)
	}
}

func TestNameIndexKeepsDetails(t *testing.T) {
	store := newPetStore(openTestDB(t))

	pet := Pet{ID: 1, Name: "旧名字", BaseStats: Stats{HP: 100}}
	if err := store.Put(pet); err != nil {
		t.Fatalf("写入精灵失败: %v", err)
	}
	if err := store.SetNames(map[uint32]string{1: "新名字", 2: "仅有名字"}); err != nil {
		t.Fatalf("写入名称索引失败: %v", err)
	}

	if name, ok := store.Name(2); !ok || name != "仅有名字" {
		t.Fatalf("名称索引读取失败: %q %v", name, ok)
	}
	if _, ok := store.Get(2); ok {
		t.Fatal("名称索引不应被当作完整详情")
	}
	got, ok := store.Get(1)
	if !ok || got.Name != "新名字" || got.BaseStats.HP != 100 {
		t.Fatalf("名称更新破坏了详情: %+v", got)
	}
	if store.NameCount() != 2 {
		t.Fatalf("名称数量错误: %d", store.NameCount())
	}
}

func TestGetOrFetchDoesNotExpire(t *testing.T) {
	db := openTestDB(t)
	store := newPetStore(db)

	pet := Pet{ID: 1, Name: "静态精灵", BaseStats: Stats{HP: 100}}
	if err := store.Put(pet); err != nil {
		t.Fatal(err)
	}
	// 即使记录非常旧，静态数据也不应触发重新请求。
	if _, err := db.Exec(`UPDATE pets SET updated_at = 1 WHERE id = ?`, pet.ID); err != nil {
		t.Fatal(err)
	}

	called := false
	got, err := getOrFetch(&Client{}, pet.ID, "pet:1", store, func() (Pet, error) {
		called = true
		return Pet{}, nil
	})
	if err != nil {
		t.Fatalf("读取缓存失败: %v", err)
	}
	if called {
		t.Fatal("静态数据不应因时间久远而重新请求")
	}
	if !reflect.DeepEqual(got, pet) {
		t.Fatalf("缓存数据不一致: %+v", got)
	}
}

func TestFactorStoreRoundTrip(t *testing.T) {
	db := openTestDB(t)

	factorStore := newFactorStore(db)
	factor := Factor{
		ID:              2400021,
		Name:            "墨纪拉因子",
		PetLimit:        true,
		PetConsume:      540,
		SkillConsume:    108,
		SoulmarkConsume: 270,
		ItemID:          2400021,
		DesignID:        5,
		PetID:           4246,
		SkillID:         35220,
		SoulmarkID:      1466,
	}
	if err := factorStore.Put(factor); err != nil {
		t.Fatalf("写入因子失败: %v", err)
	}

	gotFactor, ok := factorStore.Get(factor.ID)
	if !ok || !reflect.DeepEqual(gotFactor, factor) {
		t.Fatalf("因子数据不一致: ok=%v got=%+v", ok, gotFactor)
	}
	byDesign, ok := factorStore.FirstBy("design_id", factor.DesignID)
	if !ok || byDesign.ID != factor.ID {
		t.Fatalf("按关卡 ID 查询因子失败: %+v", byDesign)
	}

	designStore := newFactorDesignStore(db)
	design := FactorDesign{
		ID:               5,
		Times:            3,
		TimeValue:        200723,
		ProgressValue:    200722,
		FailTimes:        200720,
		AchievementID:    3145004,
		RewardID:         2400021,
		MintmarkRewardID: 42055,
		Difficulties: []FactorDifficulty{
			{
				Level:       "easy",
				RewardCount: 12,
				Tasks: []FactorTask{
					{Desc: "第一关", BossPetID: 4189},
					{Desc: "第二关", BossPetID: 4078, IsSinglePet: true},
				},
				Rules: []FactorRule{
					{ID: 2, Tips: "六回合内击败", FailTips: "超时"},
				},
			},
			{
				Level:       "hard",
				RewardCount: 36,
				Tasks: []FactorTask{
					{Desc: "困难第一关", BossPetID: 4200},
				},
				BannedPets: []uint32{3022, 3460},
				Rules: []FactorRule{
					{ID: 1, Tips: "不可重复上阵"},
					{ID: 3, Tips: "禁用精灵"},
				},
			},
		},
	}
	if err := designStore.Put(design); err != nil {
		t.Fatalf("写入因子关卡失败: %v", err)
	}

	gotDesign, ok := designStore.Get(design.ID)
	if !ok || !reflect.DeepEqual(gotDesign, design) {
		t.Fatalf("因子关卡数据不一致:\n got=%+v\nwant=%+v", gotDesign, design)
	}

	all := designStore.All()
	if len(all) != 1 || !reflect.DeepEqual(all[0], design) {
		t.Fatalf("读取全部因子关卡失败: %+v", all)
	}

	easy := gotDesign.Difficulty(1)
	if easy == nil || len(easy.Tasks) != 2 || easy.Tasks[0].Desc != "第一关" {
		t.Fatalf("简单难度数据错误: %+v", easy)
	}
	hard := gotDesign.Difficulty(3)
	if hard == nil || len(hard.BannedPets) != 2 || len(hard.Rules) != 2 {
		t.Fatalf("困难难度数据错误: %+v", hard)
	}
}

func TestStoredConfigVersionMigration(t *testing.T) {
	c := &Client{db: openTestDB(t)}

	if err := c.setMeta(metaConfigVersionLegacy, "legacy-version"); err != nil {
		t.Fatal(err)
	}
	if got := c.storedConfigVersion(); got != "legacy-version" {
		t.Fatalf("旧版本号读取失败: %q", got)
	}
	if got := c.metaValue(metaConfigVersion); got != "legacy-version" {
		t.Fatalf("版本号未迁移到新键: %q", got)
	}
	if got := c.metaValue(metaConfigVersionLegacy); got != "" {
		t.Fatalf("旧版本键未清理: %q", got)
	}

	if err := c.setMeta(metaConfigVersion, "new-version"); err != nil {
		t.Fatal(err)
	}
	if err := c.setMeta(metaConfigVersionLegacy, "stale-version"); err != nil {
		t.Fatal(err)
	}
	if got := c.storedConfigVersion(); got != "new-version" {
		t.Fatalf("应优先使用新版本键: %q", got)
	}
}

func TestEnsureFactorColumns(t *testing.T) {
	db := openTestDB(t)

	if _, err := db.Exec(`DROP TABLE factor_designs`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE factor_designs (
		id INTEGER PRIMARY KEY,
		achievement_id INTEGER NOT NULL DEFAULT 0,
		reward_id INTEGER NOT NULL DEFAULT 0,
		mintmark_reward_id INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO meta (key, value) VALUES (?, ?)`, metaFactorSyncTime, "123"); err != nil {
		t.Fatal(err)
	}

	if err := ensureFactorColumns(db); err != nil {
		t.Fatalf("补充因子进度字段失败: %v", err)
	}
	for _, column := range []string{"times", "time_value", "progress_value", "fail_times"} {
		if !hasColumn(db, "factor_designs", column) {
			t.Fatalf("缺少字段 %s", column)
		}
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM meta WHERE key = ?`, metaFactorSyncTime).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("补充字段后应清除因子同步时间，强制重新同步")
	}
}

func TestLegacyJSONMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	legacy, err := sql.Open("sqlite", filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("创建旧库失败: %v", err)
	}

	if _, err := legacy.Exec(`CREATE TABLE pets (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		data TEXT,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		t.Fatalf("创建旧精灵表失败: %v", err)
	}
	if _, err := legacy.Exec(`CREATE TABLE skills (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		data TEXT NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`); err != nil {
		t.Fatalf("创建旧技能表失败: %v", err)
	}

	pet := Pet{
		ID:          1,
		Name:        "迁移精灵",
		BaseStats:   Stats{Atk: 9, HP: 88},
		Skills:      []PetSkill{{ID: 10, Level: 3}},
		SoulmarkIDs: []uint32{55},
	}
	petData, _ := json.Marshal(pet)
	if _, err := legacy.Exec(`INSERT INTO pets (id, name, data, updated_at) VALUES (?, ?, ?, ?)`,
		pet.ID, pet.Name, string(petData), 100); err != nil {
		t.Fatalf("写入旧精灵失败: %v", err)
	}
	if _, err := legacy.Exec(`INSERT INTO pets (id, name, data, updated_at) VALUES (?, ?, NULL, 0)`,
		uint32(2), "索引精灵"); err != nil {
		t.Fatalf("写入旧名称索引失败: %v", err)
	}

	skill := Skill{ID: 10, Name: "迁移技能", Power: 80, MaxPP: 5, MustHit: true}
	skillData, _ := json.Marshal(skill)
	if _, err := legacy.Exec(`INSERT INTO skills (id, name, data, updated_at) VALUES (?, ?, ?, ?)`,
		skill.ID, skill.Name, string(skillData), 100); err != nil {
		t.Fatalf("写入旧技能失败: %v", err)
	}
	legacy.Close()

	db, err := openDB(path)
	if err != nil {
		t.Fatalf("迁移旧库失败: %v", err)
	}
	defer db.Close()

	if hasColumn(db, "pets", "data") {
		t.Fatal("迁移后不应保留 data 列")
	}

	petStore := newPetStore(db)
	gotPet, ok := petStore.Get(pet.ID)
	if !ok || !reflect.DeepEqual(gotPet, pet) {
		t.Fatalf("精灵迁移结果错误: ok=%v got=%+v want=%+v", ok, gotPet, pet)
	}
	if name, ok := petStore.Name(2); !ok || name != "索引精灵" {
		t.Fatalf("名称索引迁移失败: %q %v", name, ok)
	}
	if _, ok := petStore.Get(2); ok {
		t.Fatal("名称索引行不应包含详情")
	}
	if updatedAt, ok := petStore.UpdatedAt(pet.ID); !ok || !updatedAt.Equal(time.Unix(100, 0)) {
		t.Fatalf("迁移应保留更新时间: %v %v", updatedAt, ok)
	}

	skillStore := newSkillStore(db)
	gotSkill, ok := skillStore.Get(skill.ID)
	if !ok || !reflect.DeepEqual(gotSkill, skill) {
		t.Fatalf("技能迁移结果错误: ok=%v got=%+v want=%+v", ok, gotSkill, skill)
	}
	if updatedAt, ok := skillStore.UpdatedAt(skill.ID); !ok || !updatedAt.Equal(time.Unix(100, 0)) {
		t.Fatalf("技能迁移应保留更新时间: %v %v", updatedAt, ok)
	}
}
