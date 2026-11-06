package config

import (
	"os"
	"path/filepath"
)

// RootDir 是运行数据根目录，默认当前工作目录。
// 可通过 -root 启动参数指定。
var RootDir = "."

func SetRoot(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	RootDir = abs
	return nil
}

// DataDir 是全部运行时数据的根目录。
func DataDir() string { return filepath.Join(RootDir, "data") }

// LuaDir 存放明文 Lua 脚本，按端口分为 flash / unity 两个子目录。
func LuaDir() string      { return filepath.Join(DataDir(), "lua") }
func FlashLuaDir() string { return filepath.Join(LuaDir(), "flash") }
func UnityLuaDir() string { return filepath.Join(LuaDir(), "unity") }

// PlansDir 存放战斗方案 YAML。
func PlansDir() string { return filepath.Join(DataDir(), "scripts") }

// DBDir 存放 SQLite 缓存。
func DBDir() string { return filepath.Join(DataDir(), "db") }

func ConfigDir() string { return filepath.Join(DataDir(), "config") }

// EnsureDirs 创建运行所需的目录。
func EnsureDirs() error {
	for _, dir := range []string{
		LuaDir(),
		FlashLuaDir(),
		UnityLuaDir(),
		PlansDir(),
		DBDir(),
		ConfigDir(),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
