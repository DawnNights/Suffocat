package studio

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"suffocat/internal/common"
	luabind "suffocat/internal/lua"
	"suffocat/internal/lua/builtin"
	"suffocat/internal/lua/fight"
	"suffocat/internal/lua/petbag"
	"suffocat/internal/protocol"
	"suffocat/internal/seerapi"
	"suffocat/internal/web"

	lua "github.com/yuin/gopher-lua"
)

func apiRunFactor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		ID       uint32 `json:"id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if body.Account == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "请输入米米号和密码")
		return
	}
	if _, ok := factorDesign(body.ID); !ok {
		writeError(w, http.StatusNotFound, "未找到对应的因子关卡")
		return
	}
	if _, ok := factorForDesign(body.ID); !ok {
		writeError(w, http.StatusNotFound, "未找到对应的因子道具")
		return
	}

	if err := runFactor(body.Account, body.Password, body.ID); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]string{"ok": "started"})
}

func apiRunFactorCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if body.Account == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "请输入米米号和密码")
		return
	}

	if err := runFactorCategory(body.Account, body.Password); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]string{"ok": "started"})
}

// runFactor 执行单个因子任务。
func runFactor(account, password string, id uint32) error {
	return startEngineTask(func(ctx context.Context) {
		var userID uint32
		fmt.Sscan(account, &userID)

		key := factorProgressKey(id)
		msg := progressMsg{Index: 0, File: key, Type: "FACTOR", Status: "running", Total: 1}
		appendProgress(msg)
		web.WsSend("studio:progress", mustJSON(msg))

		session := newFactorSession(ctx, userID, password)
		defer session.Close()

		ok := executeFactor(ctx, session, id)
		status := "success"
		if !ok {
			status = "failed"
		}
		if ctx.Err() != nil {
			status = "stopped"
		}

		final := progressMsg{Index: 0, File: key, Type: "FACTOR", Status: status, Total: 1}
		appendProgress(final)
		web.WsSend("studio:progress", mustJSON(final))
	})
}

// runFactorCategory 执行所有已启用的因子任务。
func runFactorCategory(account, password string) error {
	return startEngineTask(func(ctx context.Context) {
		var userID uint32
		fmt.Sscan(account, &userID)

		var enabled []factorItem
		for _, factor := range listFactors() {
			if factor.Enabled {
				enabled = append(enabled, factor)
			}
		}
		if len(enabled) == 0 {
			studioLog("WARN", "没有勾选任何因子")
			return
		}

		session := newFactorSession(ctx, userID, password)
		defer session.Close()

		total := len(enabled)
		for i, factor := range enabled {
			if ctx.Err() != nil {
				studioLog("INFO", "用户中止运行")
				return
			}

			key := factorProgressKey(factor.ID)
			msg := progressMsg{Index: i, File: key, Type: "FACTOR", Status: "running", Total: total}
			appendProgress(msg)
			web.WsSend("studio:progress", mustJSON(msg))

			ok := executeFactor(ctx, session, factor.ID)
			status := "success"
			if !ok {
				status = "failed"
			}
			if ctx.Err() != nil {
				status = "stopped"
			}

			final := progressMsg{Index: i, File: key, Type: "FACTOR", Status: status, Total: total}
			appendProgress(final)
			web.WsSend("studio:progress", mustJSON(final))
		}

		studioLog("INFO", "全部因子任务执行完毕")
	})
}

type factorCheckResult struct {
	complete    bool
	remainTimes int
	consume     [3]uint32
	maxConsume  uint32
}

func executeFactor(ctx context.Context, session *factorSession, id uint32) bool {
	design, ok := factorDesign(id)
	if !ok {
		studioLog("ERROR", "未找到对应的因子关卡数据")
		return false
	}
	factor, ok := factorForDesign(id)
	if !ok {
		studioLog("ERROR", "未找到对应的因子道具数据")
		return false
	}
	state := loadFactorState()
	settings := normalizedFactorSettings(design, state.Settings[factorKey(id)])
	difficulty := factorDifficulty(design, settings.Mode)
	if difficulty == nil || len(difficulty.Tasks) == 0 {
		studioLog("WARN", fmt.Sprintf("因子[%s]没有可挑战的关卡", factorName(factor)))
		return false
	}
	stageTotal := len(difficulty.Tasks)

	L := builtin.NewLuaState()
	defer L.Close()
	builtin.SetLogger(L, studioLog)
	fight.SetFightLogger(&WebFightLogger{})

	firstPort := "UNITY"
	if len(settings.Stages) > 0 {
		firstPort = stagePort(settings.Stages[0])
	}

	var check factorCheckResult
	err := session.Do(firstPort, func(sc protocol.SeerConn) error {
		ls := luabind.NewLuaSeer(sc, L)
		check = checkFactor(sc, ls, factor, design)
		return nil
	})
	if err != nil {
		studioLog("ERROR", err.Error())
		return false
	}
	if check.complete || check.remainTimes <= 0 {
		return true
	}

	studioLog("INFO", fmt.Sprintf("=== 开始因子: %s (%s) ===", factorName(factor), modeLabel(settings.Mode)))
	failTimes := settings.FailTimes

	for run := 0; run < check.remainTimes; run++ {
		if ctx.Err() != nil {
			studioLog("INFO", "用户中止运行")
			return false
		}

		var swept bool
		var progress, currentMode uint32

		err := session.Do(firstPort, func(sc protocol.SeerConn) error {
			if settings.Speedrun {
				body := factorSend(sc, 41283, id, uint32(settings.Mode+4))
				if !body.IsEmpty() {
					swept = true
					return nil
				}
			}

			body := factorSend(sc, 46046, 1, design.ProgressValue)
			if body.Length() < 7 {
				return fmt.Errorf("读取因子进度失败")
			}
			body.Skip(4).Skip(1)
			progress = uint32(body.ReadU8())
			currentMode = uint32(body.ReadU8())
			return nil
		})
		if err != nil {
			studioLog("ERROR", err.Error())
			return false
		}

		if swept {
			studioLog("INFO", fmt.Sprintf("[%s] 扫荡完成", factorName(factor)))
			continue
		}
		if currentMode != 0 && currentMode != uint32(settings.Mode) {
			studioLog("WARN", fmt.Sprintf("正在挑战[%s]因子的[%s]模式，请放弃挑战后再运行",
				factorName(factor), modeLabel(int(currentMode))))
			return false
		}
		if progress >= uint32(stageTotal) {
			continue
		}

		groups := groupFactorStages(settings, int(progress)+1, stageTotal)
		for _, group := range groups {
			if ctx.Err() != nil {
				studioLog("INFO", "用户中止运行")
				return false
			}

			err := session.Do(group.Port, func(sc protocol.SeerConn) error {
				ls := luabind.NewLuaSeer(sc, L)
				for _, stage := range group.Stages {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					studioLog("INFO", fmt.Sprintf("开始挑战%s-%s第%d关",
						modeLabel(settings.Mode), factorName(factor), stage))
					if !executeFactorStage(ctx, sc, L, ls, id, difficulty, settings, stage, &failTimes) {
						return fmt.Errorf("第%d关挑战失败", stage)
					}
				}
				return nil
			})
			if err != nil {
				studioLog("ERROR", err.Error())
				return false
			}
		}
	}

	studioLog("INFO", fmt.Sprintf("因子[%s]今日任务执行完毕", factorName(factor)))
	return true
}

func checkFactor(sc protocol.SeerConn, ls *luabind.LuaSeer, factor *seerapi.Factor, design *seerapi.FactorDesign) factorCheckResult {
	var result factorCheckResult

	petID := factor.PetID
	if petID == 0 {
		petID = factor.NeedPetID
	}

	if petID != 0 && ls.HasPet(petID) {
		var pet *petbag.BagPet
		if found := ls.PetBag().FindByID(petID); found != nil {
			pet = found
		} else {
			pet = ls.PetStorage().FindByID(petID)
		}

		if pet != nil {
			info := pet.PetInfoRaw()
			result.consume[0] = factor.PetConsume
			if factor.SoulmarkID != 0 {
				for _, effect := range info.EffectList {
					if effect.EffectID == factor.SoulmarkID {
						result.consume[1] = factor.SoulmarkConsume
					}
				}
			}
			if info.SkillInfos[4].ID != 0 {
				result.consume[2] = factor.SkillConsume
			}
		}
	}

	petDone := result.consume[0] != 0 || factor.PetConsume == 0
	soulmarkDone := result.consume[1] != 0 || factor.SoulmarkConsume == 0
	skillDone := result.consume[2] != 0 || factor.SkillConsume == 0
	if petDone && soulmarkDone && skillDone {
		studioLog("INFO", fmt.Sprintf("检测到精灵[%s]及其魂印、第五技能已获得，跳过此次执行", factorName(factor)))
		result.complete = true
		return result
	}

	result.maxConsume = factor.PetConsume + factor.SoulmarkConsume + factor.SkillConsume
	itemID := factor.ItemID
	if itemID == 0 {
		itemID = factor.ID
	}
	current := result.consume[0] + result.consume[1] + result.consume[2] + ls.ItemCount(itemID)
	if current >= result.maxConsume {
		studioLog("INFO", fmt.Sprintf("当前已累计获得%s: %d/%d，请自行前往游戏中兑换精灵",
			factorName(factor), current, result.maxConsume))
		result.complete = true
		return result
	}

	remain := int(design.Times) - int(readFactorU32(sc, design.TimeValue))
	if readFactorProgressFlag(sc, design.ProgressValue) {
		remain++
	}
	if remain < 0 {
		remain = 0
	}
	result.remainTimes = remain
	if remain <= 0 {
		studioLog("INFO", fmt.Sprintf("今日[%s]因子挑战已完成", factorName(factor)))
	}
	return result
}

func readFactorU32(sc protocol.SeerConn, value uint32) uint32 {
	body := factorSend(sc, 46046, 1, value)
	if body.Length() < 8 {
		return 0
	}
	return body.Skip(4).ReadU32()
}

func readFactorProgressFlag(sc protocol.SeerConn, value uint32) bool {
	body := factorSend(sc, 46046, 1, value)
	if body.Length() < 7 {
		return false
	}
	body.Skip(4).Skip(1).Skip(1)
	return body.ReadU8() != 0
}

func executeFactorStage(
	ctx context.Context,
	sc protocol.SeerConn,
	L *lua.LState,
	ls *luabind.LuaSeer,
	designID uint32,
	difficulty *seerapi.FactorDifficulty,
	settings *FactorSettings,
	stage int,
	failTimes *int,
) bool {
	ruleBody := factorSend(sc, 41284, designID, uint32(settings.Mode))
	if ruleBody.Length() < 4 {
		studioLog("WARN", "读取关卡规则失败")
		return false
	}
	rule := ruleBody.ReadU32()
	if rule != 0 {
		studioLog("WARN", factorRuleTip(difficulty, rule))
		return false
	}

	if stage < 1 || stage > len(settings.Stages) {
		studioLog("WARN", fmt.Sprintf("缺少第%d关的挑战配置", stage))
		return false
	}
	stageConfig := settings.Stages[stage-1]
	if stageConfig.Plan == "" {
		studioLog("WARN", fmt.Sprintf("请设置第%d关的挑战方案", stage))
		return false
	}

	script, ok := builtin.LoadPlan(stageConfig.Plan)
	if !ok {
		studioLog("WARN", fmt.Sprintf("读取第%d关挑战方案失败: %s", stage, stageConfig.Plan))
		return false
	}

	driver := ls.ExecFightScript(script)
	if driver == nil {
		studioLog("WARN", "战斗方案映射失败")
		return false
	}

	for {
		if *failTimes < 0 {
			studioLog("WARN", "失败次数过多，请重新选择战斗方案")
			return false
		}
		if ctx.Err() != nil {
			return false
		}

		if script.ScriptConfig.HPLimit {
			ls.PetHPLimit()
		} else {
			ls.PetCure()
		}

		sc.SendAndRecv(41282, func(bs *common.ByteStream) {
			bs.WriteU32List(designID, uint32(settings.Mode))
		})

		readyFight := fight.NewReadyFight(L, sc)
		readyFight.BindDriver(driver)

		done := make(chan bool, 1)
		go func() {
			done <- readyFight.Start()
		}()

		select {
		case win := <-done:
			if win {
				return true
			}
			*failTimes--
		case <-ctx.Done():
			return false
		}
	}
}

func factorRuleTip(difficulty *seerapi.FactorDifficulty, ruleID uint32) string {
	if difficulty == nil {
		return fmt.Sprintf("关卡规则未满足 (规则ID %d)", ruleID)
	}
	for _, rule := range difficulty.Rules {
		if rule.ID == ruleID {
			return rule.Tips
		}
	}

	index := int(ruleID) - 1
	if index >= 0 && index < len(difficulty.Rules) {
		return difficulty.Rules[index].Tips
	}
	return fmt.Sprintf("关卡规则未满足 (规则ID %d)", ruleID)
}

type factorStageGroup struct {
	Port   string
	Stages []int
}

func stagePort(config FactorStageConfig) string {
	if config.Port == "FLASH" {
		return "FLASH"
	}
	return "UNITY"
}

func groupFactorStages(settings *FactorSettings, from, to int) []factorStageGroup {
	var groups []factorStageGroup
	for stage := from; stage <= to; stage++ {
		if stage < 1 || stage > len(settings.Stages) {
			continue
		}
		port := stagePort(settings.Stages[stage-1])
		if len(groups) == 0 || groups[len(groups)-1].Port != port {
			groups = append(groups, factorStageGroup{Port: port})
		}
		groups[len(groups)-1].Stages = append(groups[len(groups)-1].Stages, stage)
	}
	return groups
}

func factorSend(sc protocol.SeerConn, cmd uint32, args ...uint32) *common.ByteStream {
	return sc.SendAndRecv(cmd, func(bs *common.ByteStream) {
		bs.WriteU32List(args...)
	}).Wait().Body
}

// factorSession 维护因子任务期间的游戏登录状态。
// 同一端口下复用连接，只有切换端口或连接断开时才重新登录。
type factorDialFunc func(port string, userID uint32, password string, logFn func(string, string)) protocol.SeerConn

type factorSession struct {
	ctx      context.Context
	userID   uint32
	password string
	dial     factorDialFunc

	mu    sync.Mutex
	state *factorConnState
}

type factorConnState struct {
	port     string
	conn     protocol.SeerConn
	ready    chan struct{}
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func newFactorSession(ctx context.Context, userID uint32, password string) *factorSession {
	return &factorSession{ctx: ctx, userID: userID, password: password, dial: dialFactorConn}
}

func dialFactorConn(port string, userID uint32, password string, logFn func(string, string)) protocol.SeerConn {
	if port == "FLASH" {
		return protocol.NewFlashConn(userID, password, logFn)
	}
	return protocol.NewUnityConn(userID, password, logFn)
}

// Do 在当前登录连接上执行操作。
func (s *factorSession) Do(port string, fn func(protocol.SeerConn) error) error {
	if err := s.ensure(port); err != nil {
		return err
	}

	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	if state == nil {
		return fmt.Errorf("游戏连接未就绪")
	}

	result := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("连接操作发生异常: %v", r)
			}
			result <- err
		}()
		err = fn(state.conn)
	}()

	select {
	case err := <-result:
		return err
	case <-state.done:
		return fmt.Errorf("与游戏服务器的连接已断开")
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *factorSession) ensure(port string) error {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	if state != nil {
		if state.port == port {
			select {
			case <-state.done:
				studioLog("WARN", fmt.Sprintf("%s连接已断开，正在重新登录", port))
				s.discard(state)
			default:
				return nil
			}
		} else {
			studioLog("INFO", fmt.Sprintf("切换登录端口: %s -> %s", state.port, port))
			s.discard(state)
		}
	}

	return s.connect(port)
}

func (s *factorSession) connect(port string) error {
	conn := s.dial(port, s.userID, s.password, studioLog)
	if conn == nil {
		return fmt.Errorf("创建%s连接失败", port)
	}

	state := &factorConnState{
		port:  port,
		conn:  conn,
		ready: make(chan struct{}),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}

	s.mu.Lock()
	s.state = state
	s.mu.Unlock()

	go func() {
		defer close(state.done)
		conn.Start(func(sc protocol.SeerConn) {
			s.mu.Lock()
			current := s.state
			s.mu.Unlock()
			if current != state {
				sc.Exit()
				return
			}
			close(state.ready)
			<-state.stop
			sc.Exit()
		})
	}()

	select {
	case <-state.ready:
		currentConnMu.Lock()
		currentConn = conn
		currentConnMu.Unlock()
		return nil
	case <-state.done:
		s.discard(state)
		return fmt.Errorf("%s连接失败或已断开", port)
	case <-s.ctx.Done():
		s.discard(state)
		return s.ctx.Err()
	}
}

// Close 结束任务并退出登录。
func (s *factorSession) Close() {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	s.discard(state)
}

func (s *factorSession) discard(state *factorConnState) {
	if state == nil {
		return
	}

	s.mu.Lock()
	if s.state == state {
		s.state = nil
	}
	s.mu.Unlock()

	state.stopOnce.Do(func() { close(state.stop) })
	state.conn.Exit()

	currentConnMu.Lock()
	if currentConn == state.conn {
		currentConn = nil
	}
	currentConnMu.Unlock()
}
