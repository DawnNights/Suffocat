package seerapi

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// openDB 打开（或创建）SQLite 数据库并初始化表结构。
func openDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}

	// 单连接即可满足本地工具的使用；避免并发写入时的锁竞争。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("执行 %s 失败: %w", pragma, err)
		}
	}

	if err := prepareSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := ensureFactorColumns(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS pets (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		resource_id INTEGER NOT NULL DEFAULT 0,
		catch_rate INTEGER NOT NULL DEFAULT 0,
		type_id INTEGER NOT NULL DEFAULT 0,
		gender INTEGER NOT NULL DEFAULT 0,
		atk INTEGER NOT NULL DEFAULT 0,
		def INTEGER NOT NULL DEFAULT 0,
		sp_atk INTEGER NOT NULL DEFAULT 0,
		sp_def INTEGER NOT NULL DEFAULT 0,
		spd INTEGER NOT NULL DEFAULT 0,
		hp INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_pets_name ON pets(name);`,
	`CREATE TABLE IF NOT EXISTS pet_skills (
		pet_id INTEGER NOT NULL,
		position INTEGER NOT NULL,
		skill_id INTEGER NOT NULL,
		level INTEGER NOT NULL DEFAULT 0,
		special INTEGER NOT NULL DEFAULT 0,
		advanced INTEGER NOT NULL DEFAULT 0,
		fifth INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (pet_id, position)
	);`,
	`CREATE TABLE IF NOT EXISTS pet_soulmarks (
		pet_id INTEGER NOT NULL,
		position INTEGER NOT NULL,
		soulmark_id INTEGER NOT NULL,
		PRIMARY KEY (pet_id, position)
	);`,
	`CREATE TABLE IF NOT EXISTS skills (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		category_id INTEGER NOT NULL DEFAULT 0,
		type_id INTEGER NOT NULL DEFAULT 0,
		power INTEGER NOT NULL DEFAULT 0,
		max_pp INTEGER NOT NULL DEFAULT 0,
		accuracy INTEGER NOT NULL DEFAULT 0,
		priority INTEGER NOT NULL DEFAULT 0,
		crit_rate REAL NOT NULL DEFAULT 0,
		must_hit INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);`,
	`CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_items_name ON items(name);`,
	`CREATE TABLE IF NOT EXISTS mintmarks (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		class_id INTEGER NOT NULL DEFAULT 0,
		extra_atk INTEGER NOT NULL DEFAULT 0,
		extra_def INTEGER NOT NULL DEFAULT 0,
		extra_sp_atk INTEGER NOT NULL DEFAULT 0,
		extra_sp_def INTEGER NOT NULL DEFAULT 0,
		extra_spd INTEGER NOT NULL DEFAULT 0,
		extra_hp INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_mintmarks_name ON mintmarks(name);`,
	`CREATE TABLE IF NOT EXISTS suits (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_suits_name ON suits(name);`,
	`CREATE TABLE IF NOT EXISTS suit_equips (
		suit_id INTEGER NOT NULL,
		position INTEGER NOT NULL,
		equip_id INTEGER NOT NULL,
		PRIMARY KEY (suit_id, position)
	);`,
	`CREATE TABLE IF NOT EXISTS equips (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		part INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_equips_name ON equips(name);`,
	`CREATE TABLE IF NOT EXISTS factors (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		pet_limit INTEGER NOT NULL DEFAULT 0,
		pet_consume INTEGER NOT NULL DEFAULT 0,
		skill_consume INTEGER NOT NULL DEFAULT 0,
		soulmark_consume INTEGER NOT NULL DEFAULT 0,
		item_id INTEGER NOT NULL DEFAULT 0,
		design_id INTEGER NOT NULL DEFAULT 0,
		need_pet_id INTEGER NOT NULL DEFAULT 0,
		pet_id INTEGER NOT NULL DEFAULT 0,
		skill_id INTEGER NOT NULL DEFAULT 0,
		soulmark_id INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE INDEX IF NOT EXISTS idx_factors_name ON factors(name);`,
	`CREATE INDEX IF NOT EXISTS idx_factors_design ON factors(design_id);`,
	`CREATE TABLE IF NOT EXISTS factor_designs (
		id INTEGER PRIMARY KEY,
		times INTEGER NOT NULL DEFAULT 3,
		time_value INTEGER NOT NULL DEFAULT 0,
		progress_value INTEGER NOT NULL DEFAULT 0,
		fail_times INTEGER NOT NULL DEFAULT 0,
		achievement_id INTEGER NOT NULL DEFAULT 0,
		reward_id INTEGER NOT NULL DEFAULT 0,
		mintmark_reward_id INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`,
	`CREATE TABLE IF NOT EXISTS factor_difficulties (
		design_id INTEGER NOT NULL,
		position INTEGER NOT NULL,
		level TEXT NOT NULL DEFAULT '',
		reward_count INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (design_id, position)
	);`,
	`CREATE TABLE IF NOT EXISTS factor_tasks (
		design_id INTEGER NOT NULL,
		difficulty INTEGER NOT NULL,
		position INTEGER NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		is_single_pet INTEGER NOT NULL DEFAULT 0,
		boss_pet_id INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (design_id, difficulty, position)
	);`,
	`CREATE TABLE IF NOT EXISTS factor_rules (
		design_id INTEGER NOT NULL,
		difficulty INTEGER NOT NULL,
		position INTEGER NOT NULL,
		rule_id INTEGER NOT NULL DEFAULT 0,
		tips TEXT NOT NULL DEFAULT '',
		fail_tips TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (design_id, difficulty, position)
	);`,
	`CREATE TABLE IF NOT EXISTS factor_banned_pets (
		design_id INTEGER NOT NULL,
		difficulty INTEGER NOT NULL,
		position INTEGER NOT NULL,
		pet_id INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (design_id, difficulty, position)
	);`,
	`CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
}

type rowScanner interface {
	Scan(dest ...any) error
}

// childTable 描述一对多子表。columns 同时用于读取和写入，
// 最后一列固定为 position，用于保持数据源中的顺序。
type childTable[T any] struct {
	table        string
	parentColumn string
	orderBy      string
	columns      []string
	valuesFn     func(T) [][]any
	load         func(rowScanner, *T) error
}

func (c *childTable[T]) selectSQL() string {
	orderBy := c.orderBy
	if orderBy == "" {
		orderBy = "position"
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? ORDER BY %s",
		strings.Join(c.columns, ", "), c.table, c.parentColumn, orderBy)
}

func (c *childTable[T]) insertSQL() string {
	columns := append([]string{c.parentColumn}, c.columns...)
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		c.table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
}

func (c *childTable[T]) deleteSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE %s = ?", c.table, c.parentColumn)
}

func (c *childTable[T]) loadRows(db *sql.DB, id uint32, value *T) error {
	rows, err := db.Query(c.selectSQL(), id)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := c.load(rows, value); err != nil {
			return err
		}
	}
	return rows.Err()
}

// tableStore 是单表实体的通用访问层。数据按字段存储，
// 可变长子数据通过 childTable 写入独立子表。
type tableStore[T any] struct {
	db       *sql.DB
	table    string
	columns  []string
	idFn     func(T) uint32
	scanFn   func(rowScanner) (T, error)
	valuesFn func(T) []any
	children []childTable[T]
}

func (s *tableStore[T]) selectSQL() string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE id = ? AND updated_at > 0",
		strings.Join(s.columns, ", "), s.table)
}

func (s *tableStore[T]) upsertSQL() string {
	placeholders := make([]string, len(s.columns)+1)
	for i := range placeholders {
		placeholders[i] = "?"
	}

	updates := make([]string, 0, len(s.columns))
	for _, column := range s.columns {
		if column == "id" {
			continue
		}
		updates = append(updates, column+" = excluded."+column)
	}
	updates = append(updates, "updated_at = excluded.updated_at")

	return fmt.Sprintf("INSERT INTO %s (%s, updated_at) VALUES (%s) ON CONFLICT(id) DO UPDATE SET %s",
		s.table,
		strings.Join(s.columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "),
	)
}

func (s *tableStore[T]) Get(id uint32) (T, bool) {
	var zero T

	value, err := s.scanFn(s.db.QueryRow(s.selectSQL(), id))
	if err != nil {
		return zero, false
	}
	for i := range s.children {
		if err := s.children[i].loadRows(s.db, id, &value); err != nil {
			return zero, false
		}
	}
	return value, true
}

// FirstBy 按指定列查询第一条完整记录。
func (s *tableStore[T]) FirstBy(column string, value any) (T, bool) {
	var zero T

	known := false
	for _, name := range s.columns {
		if name == column {
			known = true
			break
		}
	}
	if !known {
		return zero, false
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? AND updated_at > 0 LIMIT 1",
		strings.Join(s.columns, ", "), s.table, column)
	result, err := s.scanFn(s.db.QueryRow(query, value))
	if err != nil {
		return zero, false
	}
	for i := range s.children {
		if err := s.children[i].loadRows(s.db, s.idFn(result), &result); err != nil {
			return zero, false
		}
	}
	return result, true
}

// All 返回全部完整记录。先读取并关闭父表游标，再加载子表，
// 避免单连接模式下父子查询互相阻塞。
func (s *tableStore[T]) All() []T {
	rows, err := s.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE updated_at > 0 ORDER BY id",
		strings.Join(s.columns, ", "), s.table))
	if err != nil {
		return nil
	}

	var result []T
	for rows.Next() {
		value, err := s.scanFn(rows)
		if err != nil {
			continue
		}
		result = append(result, value)
	}
	rows.Close()

	for i := range result {
		for j := range s.children {
			if err := s.children[j].loadRows(s.db, s.idFn(result[i]), &result[i]); err != nil {
				break
			}
		}
	}
	return result
}

func (s *tableStore[T]) Put(value T) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	writer, err := s.writer(tx)
	if err != nil {
		return err
	}
	defer writer.Close()

	if err := writer.Put(value, time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *tableStore[T]) BulkPut(values []T) error {
	if len(values) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	writer, err := s.writer(tx)
	if err != nil {
		return err
	}
	defer writer.Close()

	now := time.Now().Unix()
	for _, value := range values {
		if s.idFn(value) == 0 {
			continue
		}
		if err := writer.Put(value, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

type childStatements struct {
	delete *sql.Stmt
	insert *sql.Stmt
}

type storeWriter[T any] struct {
	store    *tableStore[T]
	parent   *sql.Stmt
	children []childStatements
}

func (s *tableStore[T]) writer(tx *sql.Tx) (*storeWriter[T], error) {
	parent, err := tx.Prepare(s.upsertSQL())
	if err != nil {
		return nil, err
	}

	writer := &storeWriter[T]{store: s, parent: parent}
	for i := range s.children {
		child := &s.children[i]

		deleteStmt, err := tx.Prepare(child.deleteSQL())
		if err != nil {
			writer.Close()
			return nil, err
		}
		insertStmt, err := tx.Prepare(child.insertSQL())
		if err != nil {
			deleteStmt.Close()
			writer.Close()
			return nil, err
		}
		writer.children = append(writer.children, childStatements{delete: deleteStmt, insert: insertStmt})
	}
	return writer, nil
}

func (w *storeWriter[T]) Put(value T, now int64) error {
	args := append(w.store.valuesFn(value), now)
	if _, err := w.parent.Exec(args...); err != nil {
		return err
	}

	id := w.store.idFn(value)
	for i := range w.store.children {
		if _, err := w.children[i].delete.Exec(id); err != nil {
			return err
		}
		for _, row := range w.store.children[i].valuesFn(value) {
			args := append([]any{id}, row...)
			if _, err := w.children[i].insert.Exec(args...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *storeWriter[T]) Close() {
	if w.parent != nil {
		w.parent.Close()
	}
	for _, child := range w.children {
		child.delete.Close()
		child.insert.Close()
	}
}

func (s *tableStore[T]) Count() int {
	var count int
	if s.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE updated_at > 0`, s.table)).Scan(&count) != nil {
		return 0
	}
	return count
}

func (s *tableStore[T]) UpdatedAt(id uint32) (time.Time, bool) {
	var ts int64
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT updated_at FROM %s WHERE id = ? AND updated_at > 0`, s.table),
		id,
	).Scan(&ts)
	if err != nil || ts <= 0 {
		return time.Time{}, false
	}
	return time.Unix(ts, 0), true
}

// Name 读取名称列。对于精灵，索引行只有名称、没有详情。
func (s *tableStore[T]) Name(id uint32) (string, bool) {
	var name string
	err := s.db.QueryRow(fmt.Sprintf(`SELECT name FROM %s WHERE id = ?`, s.table), id).Scan(&name)
	if err != nil || name == "" {
		return "", false
	}
	return name, true
}

// Names 返回全部非空名称。
func (s *tableStore[T]) Names() map[uint32]string {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT id, name FROM %s WHERE name != ''`, s.table))
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make(map[uint32]string)
	for rows.Next() {
		var id uint32
		var name string
		if rows.Scan(&id, &name) == nil {
			result[id] = name
		}
	}
	return result
}

// SetNames 批量写入名称索引；保留已有的详情数据与更新时间。
func (s *tableStore[T]) SetNames(names map[uint32]string) error {
	if len(names) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(`INSERT INTO %s (id, name, updated_at) VALUES (?, ?, 0)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name`, s.table))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, name := range names {
		if _, err := stmt.Exec(id, name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *tableStore[T]) IDs() []uint32 {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT id FROM %s ORDER BY id`, s.table))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var ids []uint32
	for rows.Next() {
		var id uint32
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// DetailIDs 返回已写入完整详情的记录 ID 集合。
func (s *tableStore[T]) DetailIDs() map[uint32]bool {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT id FROM %s WHERE updated_at > 0`, s.table))
	if err != nil {
		return map[uint32]bool{}
	}
	defer rows.Close()

	ids := map[uint32]bool{}
	for rows.Next() {
		var id uint32
		if rows.Scan(&id) == nil {
			ids[id] = true
		}
	}
	return ids
}

func (s *tableStore[T]) NameCount() int {
	var count int
	if s.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE name != ''`, s.table)).Scan(&count) != nil {
		return 0
	}
	return count
}

// ---------- 旧版 JSON 缓存迁移 ----------

type legacyCache struct {
	tables    []string
	pets      []Pet
	skills    []Skill
	items     []Item
	mintmarks []MintMark
	suits     []Suit
	equips    []Equip
	names     map[string]map[uint32]string
	stamps    map[string]map[uint32]int64
}

func prepareSchema(db *sql.DB) error {
	legacy, err := readLegacyCache(db)
	if err != nil {
		return err
	}

	// 新库或已完成迁移：只需保证表结构存在。
	if len(legacy.tables) == 0 {
		return execSchema(db)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧的 JSON 表；SQLite 的 DDL 支持事务，迁移失败会整体回滚。
	for _, table := range legacy.tables {
		if _, err := tx.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			return fmt.Errorf("删除旧表 %s 失败: %w", table, err)
		}
	}
	if legacy.hasTable("pets") {
		for _, table := range []string{"pet_skills", "pet_soulmarks"} {
			if _, err := tx.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				return err
			}
		}
	}
	if legacy.hasTable("suits") {
		if _, err := tx.Exec("DROP TABLE IF EXISTS suit_equips"); err != nil {
			return err
		}
	}

	for _, stmt := range schemaStatements {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
	}

	stores := newStores(db)
	now := time.Now().Unix()
	if err := bulkPutTx(tx, stores.pets, legacy.pets, legacy.stamps["pets"], now); err != nil {
		return fmt.Errorf("迁移精灵数据失败: %w", err)
	}
	if err := bulkPutTx(tx, stores.skills, legacy.skills, legacy.stamps["skills"], now); err != nil {
		return fmt.Errorf("迁移技能数据失败: %w", err)
	}
	if err := bulkPutTx(tx, stores.items, legacy.items, legacy.stamps["items"], now); err != nil {
		return fmt.Errorf("迁移道具数据失败: %w", err)
	}
	if err := bulkPutTx(tx, stores.mintmarks, legacy.mintmarks, legacy.stamps["mintmarks"], now); err != nil {
		return fmt.Errorf("迁移刻印数据失败: %w", err)
	}
	if err := bulkPutTx(tx, stores.suits, legacy.suits, legacy.stamps["suits"], now); err != nil {
		return fmt.Errorf("迁移套装数据失败: %w", err)
	}
	if err := bulkPutTx(tx, stores.equips, legacy.equips, legacy.stamps["equips"], now); err != nil {
		return fmt.Errorf("迁移部件数据失败: %w", err)
	}

	for table, names := range legacy.names {
		if len(names) == 0 {
			continue
		}
		stmt, err := tx.Prepare(fmt.Sprintf(`INSERT INTO %s (id, name, updated_at) VALUES (?, ?, 0)
			ON CONFLICT(id) DO UPDATE SET name = excluded.name`, table))
		if err != nil {
			return err
		}
		for id, name := range names {
			if _, err := stmt.Exec(id, name); err != nil {
				stmt.Close()
				return err
			}
		}
		stmt.Close()
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 回收旧 JSON 占用的空间，失败不影响使用。
	_, _ = db.Exec("VACUUM")
	return nil
}

func execSchema(db *sql.DB) error {
	for _, stmt := range schemaStatements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
	}
	return nil
}

// ensureFactorColumns 为旧版 factor_designs 表补充新增的进度字段。
// 补列后清除因子同步时间，强制下一次同步写入真实值。
func ensureFactorColumns(db *sql.DB) error {
	columns := []struct {
		name       string
		definition string
	}{
		{"times", "INTEGER NOT NULL DEFAULT 3"},
		{"time_value", "INTEGER NOT NULL DEFAULT 0"},
		{"progress_value", "INTEGER NOT NULL DEFAULT 0"},
		{"fail_times", "INTEGER NOT NULL DEFAULT 0"},
	}

	added := false
	for _, column := range columns {
		if hasColumn(db, "factor_designs", column.name) {
			continue
		}
		if _, err := db.Exec("ALTER TABLE factor_designs ADD COLUMN " + column.name + " " + column.definition); err != nil {
			return fmt.Errorf("补充 factor_designs.%s 失败: %w", column.name, err)
		}
		added = true
	}

	if added {
		if _, err := db.Exec("DELETE FROM meta WHERE key = ?", metaFactorSyncTime); err != nil {
			return err
		}
	}
	return nil
}

func bulkPutTx[T any](tx *sql.Tx, store *tableStore[T], values []T, stamps map[uint32]int64, now int64) error {
	if len(values) == 0 {
		return nil
	}

	writer, err := store.writer(tx)
	if err != nil {
		return err
	}
	defer writer.Close()

	for _, value := range values {
		id := store.idFn(value)
		if id == 0 {
			continue
		}
		updatedAt := now
		if stamp := stamps[id]; stamp > 0 {
			updatedAt = stamp
		}
		if err := writer.Put(value, updatedAt); err != nil {
			return err
		}
	}
	return nil
}

func readLegacyCache(db *sql.DB) (*legacyCache, error) {
	cache := &legacyCache{
		names:  map[string]map[uint32]string{},
		stamps: map[string]map[uint32]int64{},
	}

	pets, petNames, stamps, err := readLegacyTable[Pet](db, "pets", &cache.tables)
	if err != nil {
		return nil, err
	}
	cache.pets = pets
	cache.names["pets"] = petNames
	cache.stamps["pets"] = stamps

	if cache.skills, _, stamps, err = readLegacyTable[Skill](db, "skills", &cache.tables); err != nil {
		return nil, err
	} else {
		cache.stamps["skills"] = stamps
	}
	if cache.items, _, stamps, err = readLegacyTable[Item](db, "items", &cache.tables); err != nil {
		return nil, err
	} else {
		cache.stamps["items"] = stamps
	}
	if cache.mintmarks, _, stamps, err = readLegacyTable[MintMark](db, "mintmarks", &cache.tables); err != nil {
		return nil, err
	} else {
		cache.stamps["mintmarks"] = stamps
	}
	if cache.suits, _, stamps, err = readLegacyTable[Suit](db, "suits", &cache.tables); err != nil {
		return nil, err
	} else {
		cache.stamps["suits"] = stamps
	}
	if cache.equips, _, stamps, err = readLegacyTable[Equip](db, "equips", &cache.tables); err != nil {
		return nil, err
	} else {
		cache.stamps["equips"] = stamps
	}

	return cache, nil
}

func (c *legacyCache) hasTable(name string) bool {
	for _, table := range c.tables {
		if table == name {
			return true
		}
	}
	return false
}

func readLegacyTable[T any](db *sql.DB, table string, tables *[]string) ([]T, map[uint32]string, map[uint32]int64, error) {
	if !hasColumn(db, table, "data") {
		return nil, nil, nil, nil
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT id, name, data, updated_at FROM %s`, table))
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	names := map[uint32]string{}
	stamps := map[uint32]int64{}
	var values []T

	for rows.Next() {
		var id uint32
		var name string
		var data sql.NullString
		var updatedAt int64
		if err := rows.Scan(&id, &name, &data, &updatedAt); err != nil {
			return nil, nil, nil, err
		}

		if data.Valid && strings.TrimSpace(data.String) != "" {
			var value T
			if json.Unmarshal([]byte(data.String), &value) == nil {
				values = append(values, value)
				if updatedAt > 0 {
					stamps[id] = updatedAt
				}
			}
		} else if name != "" {
			names[id] = name
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	*tables = append(*tables, table)
	return values, names, stamps, nil
}

func hasColumn(db *sql.DB, table, column string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultV   sql.NullString
			pk         int
		)
		if rows.Scan(&cid, &name, &columnType, &notNull, &defaultV, &pk) == nil && name == column {
			return true
		}
	}
	return false
}

// migrateJSONL 在表为空时导入旧版 JSONL 缓存。
func migrateJSONL[T any](path string, store *tableStore[T]) error {
	if store.Count() > 0 {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var values []T
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var value T
		if json.Unmarshal(line, &value) == nil {
			values = append(values, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return store.BulkPut(values)
}
