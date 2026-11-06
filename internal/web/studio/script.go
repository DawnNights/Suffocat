package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"suffocat/internal/common"
	"suffocat/internal/config"
	luabind "suffocat/internal/lua"
	"suffocat/internal/lua/builtin"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// ScriptMeta 是脚本列表展示所需的元数据。
type ScriptMeta struct {
	File     string `json:"file"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Author   string `json:"author"`
	Version  string `json:"version"`
	Desc     string `json:"desc"`
	HasSetup bool   `json:"hasSetup"`
	Enabled  bool   `json:"enabled"`
	Size     int64  `json:"size"`
	ModTime  int64  `json:"modTime"`
	Error    string `json:"error,omitempty"`
}

type cachedMeta struct {
	size    int64
	modTime int64
	meta    ScriptMeta
}

var (
	metaCacheMu sync.Mutex
	metaCache   = map[string]cachedMeta{}

	stateMu  sync.Mutex
	layoutMu sync.Mutex

	scriptPrefixes = []string{"UNITY-", "UNITY_", "FLASH-", "FLASH_"}
)

type persistedState struct {
	Order   []string        `json:"order"`
	Enabled map[string]bool `json:"enabled"`
}

func statePath() string {
	return filepath.Join(config.ConfigDir(), "state.json")
}

func loadState() persistedState {
	var state persistedState
	data, err := os.ReadFile(statePath())
	if err != nil {
		return persistedState{Enabled: map[string]bool{}}
	}
	if json.Unmarshal(data, &state) != nil {
		return persistedState{Enabled: map[string]bool{}}
	}
	if state.Enabled == nil {
		state.Enabled = map[string]bool{}
	}
	return state
}

func saveState(state persistedState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := statePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// safeScriptName 校验文件名并补全 .lua 后缀。
func safeScriptName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("文件名为空")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", errors.New("非法文件名")
	}
	if name == "." || name == ".." {
		return "", errors.New("非法文件名")
	}
	if strings.ContainsRune(name, '�') {
		return "", errors.New("文件名编码无效")
	}
	if !utf8.ValidString(name) {
		return "", errors.New("文件名编码无效")
	}
	if !strings.EqualFold(filepath.Ext(name), ".lua") {
		name += ".lua"
	}
	return name, nil
}

func stripScriptPrefix(name string) string {
	upper := strings.ToUpper(name)
	for _, prefix := range scriptPrefixes {
		if strings.HasPrefix(upper, prefix) && len(name) > len(prefix) {
			return name[len(prefix):]
		}
	}
	return name
}

func scriptDirForType(scriptType string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(scriptType)) {
	case "UNITY":
		return config.UnityLuaDir(), true
	case "FLASH":
		return config.FlashLuaDir(), true
	default:
		return "", false
	}
}

func scriptDirs() []string {
	return []string{config.UnityLuaDir(), config.FlashLuaDir(), config.LuaDir()}
}

func findScriptPath(file string) (string, bool) {
	name, err := safeScriptName(file)
	if err != nil {
		return "", false
	}
	for _, dir := range scriptDirs() {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}

	// 兜底扫描嵌套目录，方便自动整理用户手动放入的文件。
	var found string
	_ = filepath.WalkDir(config.LuaDir(), func(path string, entry os.DirEntry, err error) error {
		if err != nil || found != "" || entry.IsDir() {
			return nil
		}
		if strings.EqualFold(entry.Name(), name) {
			found = path
		}
		return nil
	})
	return found, found != ""
}

func scriptExists(file string) bool {
	_, ok := findScriptPath(file)
	return ok
}

func scriptPath(file string) (string, error) {
	path, ok := findScriptPath(file)
	if !ok {
		return "", os.ErrNotExist
	}
	return path, nil
}

func readScriptSource(file string) (string, error) {
	path, err := scriptPath(file)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseScriptSource(source string) (*luabind.LuaScript, error) {
	L := builtin.NewLuaState()
	defer L.Close()

	script := &luabind.LuaScript{}
	if err := script.FromSource(source, L); err != nil {
		return nil, err
	}
	return script, nil
}

func scriptTypeFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	script, err := parseScriptSource(string(data))
	if err != nil {
		return ""
	}
	return script.Type
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func uniqueScriptBase(name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if !scriptExists(candidate) {
			return candidate
		}
	}
	return name
}

func writeScriptFile(file, source string) (string, error) {
	oldName, err := safeScriptName(file)
	if err != nil {
		return "", err
	}
	name := stripScriptPrefix(oldName)

	destDir := ""
	script, parseErr := parseScriptSource(source)
	if parseErr == nil {
		if dir, ok := scriptDirForType(script.Type); ok {
			destDir = dir
		}
	}
	if destDir == "" {
		if path, ok := findScriptPath(name); ok {
			destDir = filepath.Dir(path)
		} else if parseErr != nil {
			return "", parseErr
		} else {
			return "", errors.New("脚本类型无效，Type 必须为 UNITY 或 FLASH")
		}
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	dest := filepath.Join(destDir, name)
	if oldPath, ok := findScriptPath(name); ok && !samePath(oldPath, dest) {
		if _, err := os.Stat(dest); err == nil {
			return "", errors.New("同名文件已存在")
		}
		if err := os.Rename(oldPath, dest); err != nil {
			return "", err
		}
		invalidateMeta(name)
	}

	if err := os.WriteFile(dest, []byte(source), 0644); err != nil {
		return "", err
	}
	if oldName != name {
		if oldPath, ok := findScriptPath(oldName); ok && !samePath(oldPath, dest) {
			_ = os.Remove(oldPath)
		}
		migrateScriptKeys(oldName, name)
	}
	invalidateMeta(name)
	invalidateMeta(oldName)
	return name, nil
}

func invalidateMeta(file string) {
	metaCacheMu.Lock()
	delete(metaCache, file)
	metaCacheMu.Unlock()
}

func parseScriptMeta(name string) ScriptMeta {
	path, ok := findScriptPath(name)
	if !ok {
		return ScriptMeta{File: name, Error: "文件不存在"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return ScriptMeta{File: name, Error: err.Error()}
	}

	metaCacheMu.Lock()
	cached, ok := metaCache[name]
	metaCacheMu.Unlock()
	if ok && cached.size == info.Size() && cached.modTime == info.ModTime().UnixNano() {
		return cached.meta
	}

	meta := ScriptMeta{File: name, Size: info.Size(), ModTime: info.ModTime().UnixNano()}

	source, err := os.ReadFile(path)
	if err != nil {
		meta.Error = err.Error()
	} else {
		L := builtin.NewLuaState()
		script := &luabind.LuaScript{}
		if err := script.FromSource(string(source), L); err != nil {
			meta.Error = err.Error()
		} else {
			meta.Type = strings.ToUpper(strings.TrimSpace(script.Type))
			meta.Name = script.Name
			meta.Author = script.Author
			meta.Version = script.Version
			meta.Desc = script.Desc
			meta.HasSetup = script.Setup != nil
		}
		L.Close()
	}

	metaCacheMu.Lock()
	metaCache[name] = cachedMeta{size: info.Size(), modTime: info.ModTime().UnixNano(), meta: meta}
	metaCacheMu.Unlock()
	return meta
}

func resolveScriptKey(key string, current map[string]bool) string {
	if current[key] {
		return key
	}
	stripped := stripScriptPrefix(key)
	if current[stripped] {
		return stripped
	}
	return ""
}

// migrateScriptKeysLocked 将旧的前缀文件名状态/配置迁移到去掉前缀后的名称。
func migrateScriptKeysLocked() {
	current := map[string]bool{}
	_ = filepath.WalkDir(config.LuaDir(), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".lua") {
			return nil
		}
		current[entry.Name()] = true
		return nil
	})

	stateMu.Lock()
	state := loadState()
	changed := false

	order := make([]string, 0, len(state.Order))
	for _, file := range state.Order {
		resolved := resolveScriptKey(file, current)
		if resolved == "" {
			changed = true
			continue
		}
		if resolved != file {
			changed = true
		}
		order = append(order, resolved)
	}
	state.Order = order

	enabled := make(map[string]bool, len(state.Enabled))
	for file, value := range state.Enabled {
		resolved := resolveScriptKey(file, current)
		if resolved == "" {
			changed = true
			continue
		}
		if resolved != file {
			changed = true
		}
		enabled[resolved] = value
	}
	state.Enabled = enabled

	if changed {
		_ = saveState(state)
	}
	stateMu.Unlock()

	for name := range current {
		newConfig := configPath(name)
		if _, err := os.Stat(newConfig); err == nil {
			continue
		}
		for _, prefix := range scriptPrefixes {
			oldConfig := configPath(prefix + name)
			if data, err := os.ReadFile(oldConfig); err == nil {
				_ = os.WriteFile(newConfig, data, 0644)
				_ = os.Remove(oldConfig)
				break
			}
		}
	}
}

// normalizeLuaLayoutLocked 将脚本按 Type 放入 flash/unity，并去除文件名前缀。
func normalizeLuaLayoutLocked() {
	_ = os.MkdirAll(config.FlashLuaDir(), 0755)
	_ = os.MkdirAll(config.UnityLuaDir(), 0755)

	type luaFile struct {
		path string
		name string
	}
	var files []luaFile
	_ = filepath.WalkDir(config.LuaDir(), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".lua") {
			return nil
		}
		files = append(files, luaFile{path: path, name: entry.Name()})
		return nil
	})

	for _, file := range files {
		dir, ok := scriptDirForType(scriptTypeFromFile(file.path))
		if !ok {
			continue
		}

		name := stripScriptPrefix(file.name)
		target := filepath.Join(dir, name)
		if samePath(file.path, target) {
			continue
		}
		if _, err := os.Stat(target); err == nil {
			name = uniqueScriptBase(name)
			target = filepath.Join(dir, name)
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		if err := os.Rename(file.path, target); err != nil {
			continue
		}
		invalidateMeta(file.name)
		invalidateMeta(name)
	}

	removeEmptyLuaDirs()
	migrateScriptKeysLocked()
}

func removeEmptyLuaDirs() {
	var dirs []string
	_ = filepath.WalkDir(config.LuaDir(), func(path string, entry os.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	// 从最深层开始删除空目录，保留 lua 根目录和 flash/unity。
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		if samePath(dir, config.LuaDir()) || samePath(dir, config.FlashLuaDir()) || samePath(dir, config.UnityLuaDir()) {
			continue
		}
		_ = os.Remove(dir)
	}
}

func listScripts() []ScriptMeta {
	layoutMu.Lock()
	defer layoutMu.Unlock()

	normalizeLuaLayoutLocked()

	state := loadState()
	orderIndex := make(map[string]int, len(state.Order))
	for i, file := range state.Order {
		orderIndex[file] = i
	}

	seen := map[string]bool{}
	scripts := make([]ScriptMeta, 0, 16)
	_ = filepath.WalkDir(config.LuaDir(), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".lua") {
			return nil
		}
		name := entry.Name()
		if seen[name] {
			return nil
		}
		seen[name] = true

		meta := parseScriptMeta(name)
		if enabled, ok := state.Enabled[name]; ok {
			meta.Enabled = enabled
		} else {
			meta.Enabled = true
		}
		scripts = append(scripts, meta)
		return nil
	})

	sort.SliceStable(scripts, func(i, j int) bool {
		oi, iok := orderIndex[scripts[i].File]
		oj, jok := orderIndex[scripts[j].File]
		if iok && jok {
			return oi < oj
		}
		if iok {
			return true
		}
		if jok {
			return false
		}
		return scripts[i].File < scripts[j].File
	})

	return scripts
}

func findScript(file string) (ScriptMeta, bool) {
	name, err := safeScriptName(file)
	if err != nil {
		return ScriptMeta{}, false
	}
	for _, s := range listScripts() {
		if s.File == name {
			return s, true
		}
	}
	return ScriptMeta{}, false
}

func setScriptEnabled(file string, enabled bool) error {
	name, err := safeScriptName(file)
	if err != nil {
		return err
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	state := loadState()
	state.Enabled[name] = enabled
	return saveState(state)
}

func saveScriptOrder(files []string) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	state := loadState()
	state.Order = files
	return saveState(state)
}

func deleteScriptFile(file string) error {
	name, err := safeScriptName(file)
	if err != nil {
		return err
	}
	path, err := scriptPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	invalidateMeta(name)

	stateMu.Lock()
	state := loadState()
	if state.Enabled != nil {
		delete(state.Enabled, name)
	}
	var order []string
	for _, f := range state.Order {
		if f != name {
			order = append(order, f)
		}
	}
	state.Order = order
	saveState(state)
	stateMu.Unlock()

	_ = os.Remove(configPath(name))
	return nil
}

func renameScriptFile(oldFile, newFile string) (string, error) {
	oldName, err := safeScriptName(oldFile)
	if err != nil {
		return "", err
	}
	newName, err := safeScriptName(newFile)
	if err != nil {
		return "", err
	}
	newName = stripScriptPrefix(newName)
	if oldName == newName {
		return newName, nil
	}

	src, err := scriptPath(oldName)
	if err != nil {
		return "", err
	}

	destDir := filepath.Dir(src)
	if dir, ok := scriptDirForType(scriptTypeFromFile(src)); ok {
		destDir = dir
	}
	dst := filepath.Join(destDir, newName)
	if scriptExists(newName) {
		return "", errors.New("同名文件已存在")
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	if err := os.Rename(src, dst); err != nil {
		return "", err
	}
	invalidateMeta(oldName)
	invalidateMeta(newName)

	migrateScriptKeys(oldName, newName)
	return newName, nil
}

func migrateScriptKeys(oldName, newName string) {
	stateMu.Lock()
	state := loadState()
	for i, file := range state.Order {
		if file == oldName {
			state.Order[i] = newName
		}
	}
	if value, ok := state.Enabled[oldName]; ok {
		delete(state.Enabled, oldName)
		state.Enabled[newName] = value
	}
	_ = saveState(state)
	stateMu.Unlock()

	if oldName == newName {
		return
	}
	oldConfig := configPath(oldName)
	newConfig := configPath(newName)
	if _, err := os.Stat(newConfig); err == nil {
		return
	}
	if data, err := os.ReadFile(oldConfig); err == nil {
		_ = os.WriteFile(newConfig, data, 0644)
		_ = os.Remove(oldConfig)
	}
}

// ---------- 脚本配置持久化 ----------

func configPath(file string) string {
	sum := sha256.Sum256([]byte(file))
	name := hex.EncodeToString(sum[:8]) + ".json"
	return filepath.Join(config.ConfigDir(), "scripts", name)
}

func loadScriptConfig(file string) []luabind.Config {
	data, err := os.ReadFile(configPath(file))
	if err != nil {
		return nil
	}
	var configs []luabind.Config
	if json.Unmarshal(data, &configs) != nil {
		return nil
	}
	for i := range configs {
		configs[i].Value = normalizeConfigValue(configs[i].Value, configs[i].Type)
	}
	return configs
}

func saveScriptConfig(file string, configs []luabind.Config) error {
	path := configPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func normalizeConfigValue(v any, t luabind.ConfigType) any {
	switch t {
	case luabind.TypeMultiSelect:
		switch arr := v.(type) {
		case []any:
			strs := make([]string, len(arr))
			for i, e := range arr {
				strs[i] = fmt.Sprint(e)
			}
			return strs
		case []string:
			return arr
		}
	case luabind.TypeNumberInput:
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return v
}

// getScriptConfigDef 执行脚本的 Setup，并合并已保存的配置值。
func getScriptConfigDef(file string) ([]luabind.Config, error) {
	source, err := readScriptSource(file)
	if err != nil {
		return nil, err
	}

	L := builtin.NewLuaState()
	defer L.Close()

	script := &luabind.LuaScript{}
	if err := script.FromSource(source, L); err != nil {
		return nil, err
	}
	if script.Setup == nil {
		return []luabind.Config{}, nil
	}

	cr := luabind.NewConfigRegister()
	if err := L.CallByParam(lua.P{Fn: script.Setup, NRet: 0, Protect: true}, luar.New(L, cr)); err != nil {
		return nil, fmt.Errorf("执行 Setup 失败: %v", err)
	}

	ordered := cr.OrderedConfigs()
	savedByKey := make(map[string]luabind.Config)
	for _, c := range loadScriptConfig(file) {
		savedByKey[c.Key] = c
	}
	for i, c := range ordered {
		if v, ok := savedByKey[c.Key]; ok && v.Type == c.Type {
			c.Value = normalizeConfigValue(v.Value, c.Type)
			ordered[i] = c
		}
	}
	return ordered, nil
}

// buildConfigTable 将配置定义与保存值合并成 Lua table。
func buildConfigTable(L *lua.LState, configs []luabind.Config) *lua.LTable {
	table := L.NewTable()
	for _, c := range configs {
		switch value := c.Value.(type) {
		case string:
			table.RawSetString(c.Key, lua.LString(value))
		case bool:
			table.RawSetString(c.Key, lua.LBool(value))
		case float64:
			table.RawSetString(c.Key, lua.LNumber(value))
		case int:
			table.RawSetString(c.Key, lua.LNumber(value))
		case int64:
			table.RawSetString(c.Key, lua.LNumber(value))
		case []any:
			table.RawSetString(c.Key, common.ToLuaList(L, value))
		case []string:
			table.RawSetString(c.Key, common.ToLuaList(L, value))
		}
	}
	return table
}
