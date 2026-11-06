package studio

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"suffocat/internal/config"
)

func withTestRoot(t *testing.T) string {
	t.Helper()

	oldRoot := config.RootDir
	root := t.TempDir()
	if err := config.SetRoot(root); err != nil {
		t.Fatal(err)
	}
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		config.RootDir = oldRoot
		metaCacheMu.Lock()
		metaCache = map[string]cachedMeta{}
		metaCacheMu.Unlock()
	})
	return root
}

func testScriptSource(scriptType, name string) string {
	return fmt.Sprintf(`local function Main(seer, config)
    return true
end

return {
    Type = %q,
    Name = %q,
    Main = Main,
}
`, scriptType, name)
}

func TestNormalizeScriptLayoutAndState(t *testing.T) {
	withTestRoot(t)

	flashPath := filepath.Join(config.UnityLuaDir(), "FLASH-测试FLASH.lua")
	if err := os.WriteFile(flashPath, []byte(testScriptSource("FLASH", "测试FLASH")), 0644); err != nil {
		t.Fatal(err)
	}
	unityPath := filepath.Join(config.FlashLuaDir(), "UNITY_测试UNITY.lua")
	if err := os.WriteFile(unityPath, []byte(testScriptSource("UNITY", "测试UNITY")), 0644); err != nil {
		t.Fatal(err)
	}

	if err := saveState(persistedState{
		Order:   []string{"FLASH-测试FLASH.lua", "UNITY_测试UNITY.lua"},
		Enabled: map[string]bool{"UNITY_测试UNITY.lua": true},
	}); err != nil {
		t.Fatal(err)
	}

	scripts := listScripts()
	if len(scripts) != 2 {
		t.Fatalf("脚本数量错误: %d", len(scripts))
	}

	byName := map[string]ScriptMeta{}
	for _, script := range scripts {
		byName[script.File] = script
	}
	if meta, ok := byName["测试FLASH.lua"]; !ok || meta.Type != "FLASH" {
		t.Fatalf("FLASH 脚本归档错误: %+v", byName)
	}
	if meta, ok := byName["测试UNITY.lua"]; !ok || meta.Type != "UNITY" {
		t.Fatalf("UNITY 脚本归档错误: %+v", byName)
	}

	if _, err := os.Stat(filepath.Join(config.FlashLuaDir(), "测试FLASH.lua")); err != nil {
		t.Fatalf("FLASH 脚本未移动到 flash 目录: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config.UnityLuaDir(), "测试UNITY.lua")); err != nil {
		t.Fatalf("UNITY 脚本未移动到 unity 目录: %v", err)
	}

	state := loadState()
	if len(state.Order) != 2 || state.Order[0] != "测试FLASH.lua" || state.Order[1] != "测试UNITY.lua" {
		t.Fatalf("排序状态未迁移: %+v", state.Order)
	}
	if !state.Enabled["测试UNITY.lua"] || state.Enabled["UNITY_测试UNITY.lua"] {
		t.Fatalf("启用状态未迁移: %+v", state.Enabled)
	}
}

func TestWriteScriptPlacesByType(t *testing.T) {
	withTestRoot(t)

	name, err := writeScriptFile("UNITY-新脚本.lua", testScriptSource("UNITY", "新脚本"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "新脚本.lua" {
		t.Fatalf("文件名前缀未去除: %q", name)
	}
	if _, err := os.Stat(filepath.Join(config.UnityLuaDir(), "新脚本.lua")); err != nil {
		t.Fatalf("UNITY 脚本未写入 unity 目录: %v", err)
	}

	// 修改 Type 后再次保存，应自动移动到 flash 目录。
	name, err = writeScriptFile("新脚本.lua", testScriptSource("FLASH", "新脚本"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "新脚本.lua" {
		t.Fatalf("文件名错误: %q", name)
	}
	if _, err := os.Stat(filepath.Join(config.FlashLuaDir(), "新脚本.lua")); err != nil {
		t.Fatalf("FLASH 脚本未移动到 flash 目录: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config.UnityLuaDir(), "新脚本.lua")); !os.IsNotExist(err) {
		t.Fatalf("旧的 UNITY 文件未移除: %v", err)
	}
}
