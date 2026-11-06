package builtin

import (
	"context"

	lua "github.com/yuin/gopher-lua"
)

type stateCtx struct {
	logFn func(string, string)
}

type ctxKey struct{}

func getStateCtx(L *lua.LState) *stateCtx {
	ctx := L.Context()
	if ctx == nil {
		return nil
	}
	v := ctx.Value(ctxKey{})
	if v == nil {
		return nil
	}
	return v.(*stateCtx)
}

func ensureStateCtx(L *lua.LState) *stateCtx {
	if s := getStateCtx(L); s != nil {
		return s
	}
	s := &stateCtx{
		logFn: func(string, string) {},
	}
	ctx := L.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	L.SetContext(context.WithValue(ctx, ctxKey{}, s))
	return s
}

func builtinInit(L *lua.LState) *lua.LState {
	jsonTable := L.NewTable()
	jsonTable.RawSetString("dumps", L.NewFunction(jsonDumps))
	jsonTable.RawSetString("loads", L.NewFunction(jsonLoads))
	L.SetGlobal("json", jsonTable)

	httpTable := L.NewTable()
	httpTable.RawSetString("get", L.NewFunction(httpGet))
	httpTable.RawSetString("post", L.NewFunction(httpPost))
	L.SetGlobal("http", httpTable)

	timeTable := L.NewTable()
	timeTable.RawSetString("Date", L.NewFunction(timeDate))
	timeTable.RawSetString("Sleep", L.NewFunction(timeSleep))
	timeTable.RawSetString("Unix", L.NewFunction(timeUnix))
	timeTable.RawSetString("UnixMilli", L.NewFunction(timeUnixMilli))
	L.SetGlobal("time", timeTable)

	fightscriptTable := L.NewTable()
	fightscriptTable.RawSetString("New", L.NewFunction(fightscriptNew))
	fightscriptTable.RawSetString("LoadLocal", L.NewFunction(fightscriptLoadLocal))
	L.SetGlobal("fightscript", fightscriptTable)

	loggerTable := L.NewTable()
	loggerTable.RawSetString("Debug", L.NewFunction(loggerDebug))
	loggerTable.RawSetString("Info", L.NewFunction(loggerInfo))
	loggerTable.RawSetString("Warn", L.NewFunction(loggerWarn))
	loggerTable.RawSetString("Error", L.NewFunction(loggerError))
	L.SetGlobal("logger", loggerTable)

	bitTable := L.NewTable()
	bitTable.RawSetString("band", L.NewFunction(bitBand))
	bitTable.RawSetString("bor", L.NewFunction(bitBor))
	bitTable.RawSetString("bxor", L.NewFunction(bitBxor))
	bitTable.RawSetString("bnot", L.NewFunction(bitBnot))
	bitTable.RawSetString("lshift", L.NewFunction(bitLshift))
	bitTable.RawSetString("rshift", L.NewFunction(bitRshift))
	bitTable.RawSetString("arshift", L.NewFunction(bitArshift))
	L.SetGlobal("bit", bitTable)

	return L
}

func SetLogger(L *lua.LState, fn func(string, string)) {
	ensureStateCtx(L).logFn = fn
}

func NewLuaState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)

	builtinInit(L)
	return L
}
