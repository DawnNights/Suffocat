package studio

import (
	"context"
	"fmt"
	"sync"

	luabind "suffocat/internal/lua"
	"suffocat/internal/lua/builtin"
	"suffocat/internal/lua/fight"
	"suffocat/internal/protocol"
	"suffocat/internal/web"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var (
	engineRunning bool
	engineCancel  context.CancelFunc
	engineMu      sync.Mutex

	currentConn   protocol.SeerConn
	currentConnMu sync.Mutex
)

func isRunning() bool {
	engineMu.Lock()
	defer engineMu.Unlock()
	return engineRunning
}

func stopEngine() {
	engineMu.Lock()
	if engineRunning && engineCancel != nil {
		engineCancel()
	}
	engineMu.Unlock()

	currentConnMu.Lock()
	if currentConn != nil {
		currentConn.Exit()
	}
	currentConnMu.Unlock()
}

// startEngineTask 启动一个互斥的后台任务，统一处理日志清理与结束事件。
func startEngineTask(fn func(context.Context)) error {
	engineMu.Lock()
	if engineRunning {
		engineMu.Unlock()
		return fmt.Errorf("已有任务在运行")
	}
	ctx, cancel := context.WithCancel(context.Background())
	engineRunning = true
	engineCancel = cancel
	engineMu.Unlock()

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				studioLog("ERROR", fmt.Sprintf("任务执行发生严重异常(panic): %v", r))
				currentConnMu.Lock()
				if currentConn != nil {
					currentConn.Exit()
				}
				currentConnMu.Unlock()
			}
			engineMu.Lock()
			engineRunning = false
			engineCancel = nil
			engineMu.Unlock()
			clearFightLogBuf()
			clearProgressBuf()
			web.WsSend("studio:end", "{}")
		}()

		clearLogBuf()
		clearFightLogBuf()
		clearProgressBuf()
		fn(ctx)
	}()

	return nil
}

// runScripts 运行所有已勾选脚本。filterType 为空时运行全部，
// 否则只运行指定端口（UNITY / FLASH）的脚本。
func runScripts(account, password, filterType string) error {
	return startEngineTask(func(ctx context.Context) {
		var userID uint32
		fmt.Sscan(account, &userID)

		var unityScripts, flashScripts []ScriptMeta
		for _, meta := range listScripts() {
			if !meta.Enabled {
				continue
			}
			if meta.Error != "" {
				studioLog("WARN", fmt.Sprintf("脚本 %s 解析失败，已跳过: %s", meta.File, meta.Error))
				continue
			}
			switch meta.Type {
			case "UNITY":
				unityScripts = append(unityScripts, meta)
			case "FLASH":
				flashScripts = append(flashScripts, meta)
			default:
				studioLog("WARN", fmt.Sprintf("脚本 %s 的类型(%s)无效，已跳过", meta.File, meta.Type))
			}
		}

		switch filterType {
		case "UNITY":
			flashScripts = nil
		case "FLASH":
			unityScripts = nil
		}

		total := len(unityScripts) + len(flashScripts)
		if total == 0 {
			studioLog("WARN", "没有选中要执行的脚本")
			return
		}

		progressIdx := 0

		if len(unityScripts) > 0 {
			studioLog("INFO", fmt.Sprintf("=== 开始运行 UNITY 脚本 (%d个) ===", len(unityScripts)))
			runScriptBatch(ctx, unityScripts, "UNITY", userID, password, total, &progressIdx)
		}

		if len(flashScripts) > 0 {
			select {
			case <-ctx.Done():
				studioLog("INFO", "用户中止运行")
				return
			default:
			}
			studioLog("INFO", fmt.Sprintf("=== 开始运行 FLASH 脚本 (%d个) ===", len(flashScripts)))
			runScriptBatch(ctx, flashScripts, "FLASH", userID, password, total, &progressIdx)
		}

		studioLog("INFO", "全部脚本执行完毕")
	})
}

// runScriptBatch 使用同一个连接批量执行脚本。
func runScriptBatch(ctx context.Context, scripts []ScriptMeta, connType string, userID uint32, password string, total int, progressIdx *int) {
	var conn protocol.SeerConn
	if connType == "FLASH" {
		conn = protocol.NewFlashConn(userID, password, studioLog)
	} else {
		conn = protocol.NewUnityConn(userID, password, studioLog)
	}

	if conn == nil {
		studioLog("ERROR", fmt.Sprintf("创建%s连接失败，已跳过%d个脚本，请尝试重新运行", connType, len(scripts)))
		for _, meta := range scripts {
			msg := progressMsg{Index: *progressIdx, File: meta.File, Type: meta.Type, Status: "failed", Total: total}
			appendProgress(msg)
			web.WsSend("studio:progress", mustJSON(msg))
			*progressIdx++
		}
		return
	}

	fight.SetFightLogger(&WebFightLogger{})

	conn.Start(func(sc protocol.SeerConn) {
		currentConnMu.Lock()
		currentConn = sc
		currentConnMu.Unlock()
		defer func() {
			currentConnMu.Lock()
			currentConn = nil
			currentConnMu.Unlock()
		}()
		defer sc.Exit()
		defer func() {
			if r := recover(); r != nil {
				studioLog("ERROR", fmt.Sprintf("连接回调发生严重异常(panic): %v", r))
			}
		}()

		for _, meta := range scripts {
			select {
			case <-ctx.Done():
				studioLog("INFO", "用户中止运行")
				msg := progressMsg{Index: *progressIdx, File: meta.File, Type: meta.Type, Status: "stopped", Total: total}
				appendProgress(msg)
				web.WsSend("studio:progress", mustJSON(msg))
				return
			default:
			}

			msg := progressMsg{Index: *progressIdx, File: meta.File, Type: meta.Type, Status: "running", Total: total}
			appendProgress(msg)
			web.WsSend("studio:progress", mustJSON(msg))

			ok := executeOneScript(ctx, sc, meta)
			status := "success"
			if !ok {
				status = "failed"
			}

			*progressIdx++
			msg2 := progressMsg{Index: *progressIdx - 1, File: meta.File, Type: meta.Type, Status: status, Total: total}
			appendProgress(msg2)
			web.WsSend("studio:progress", mustJSON(msg2))
		}
	})
}

// runSingleScript 单独执行一个脚本。
func runSingleScript(account, password, file string) error {
	meta, ok := findScript(file)
	if !ok {
		return fmt.Errorf("未找到脚本: %s", file)
	}
	if meta.Type != "UNITY" && meta.Type != "FLASH" {
		return fmt.Errorf("脚本类型无效: %s", meta.Type)
	}
	if meta.Error != "" {
		return fmt.Errorf("脚本解析失败: %s", meta.Error)
	}

	return startEngineTask(func(ctx context.Context) {
		var userID uint32
		fmt.Sscan(account, &userID)

		studioLog("INFO", fmt.Sprintf("=== 单独执行脚本: %s ===", meta.Name))
		progressIdx := 0
		runScriptBatch(ctx, []ScriptMeta{meta}, meta.Type, userID, password, 1, &progressIdx)
		studioLog("INFO", "脚本执行完毕")
	})
}

func executeOneScript(ctx context.Context, sc protocol.SeerConn, meta ScriptMeta) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			studioLog("ERROR", fmt.Sprintf("脚本 [%s] 执行发生严重异常(panic): %v", meta.File, r))
			sc.Exit()
			ok = false
		}
	}()

	L := builtin.NewLuaState()
	defer L.Close()
	L.SetContext(ctx)
	builtin.SetLogger(L, studioLog)

	source, err := readScriptSource(meta.File)
	if err != nil {
		studioLog("ERROR", fmt.Sprintf("读取脚本 %s 失败: %v", meta.File, err))
		return false
	}

	script := &luabind.LuaScript{}
	if err := script.FromSource(source, L); err != nil {
		studioLog("ERROR", fmt.Sprintf("解析脚本 %s 失败: %v", meta.File, err))
		return false
	}

	ordered := []luabind.Config{}
	if script.Setup != nil {
		cr := luabind.NewConfigRegister()
		if err := L.CallByParam(lua.P{Fn: script.Setup, NRet: 0, Protect: true}, luar.New(L, cr)); err != nil {
			studioLog("ERROR", fmt.Sprintf("脚本 %s 执行 Setup 失败: %v", meta.File, err))
			return false
		}
		ordered = cr.OrderedConfigs()

		savedByKey := make(map[string]luabind.Config)
		for _, c := range loadScriptConfig(meta.File) {
			savedByKey[c.Key] = c
		}
		for i, c := range ordered {
			if v, ok := savedByKey[c.Key]; ok && v.Type == c.Type {
				c.Value = normalizeConfigValue(v.Value, c.Type)
				ordered[i] = c
			}
		}
	}

	table := buildConfigTable(L, ordered)
	err = L.CallByParam(lua.P{
		Fn:      script.Main,
		NRet:    1,
		Protect: true,
	}, luar.New(L, luabind.NewLuaSeer(sc, L)), table)
	if err != nil {
		studioLog("ERROR", fmt.Sprintf("%s脚本 [%s] 运行异常: %v", script.Type, script.Name, err))
		return false
	}

	ret, isBool := L.Get(-1).(lua.LBool)
	if !isBool {
		studioLog("ERROR", fmt.Sprintf("%s脚本 [%s] Main 返回值不是 boolean", script.Type, script.Name))
		return false
	}
	if bool(ret) {
		studioLog("DEBUG", fmt.Sprintf("%s脚本 [%s] 执行成功", script.Type, script.Name))
		return true
	}
	studioLog("WARN", fmt.Sprintf("%s脚本 [%s] 执行失败", script.Type, script.Name))
	return false
}
