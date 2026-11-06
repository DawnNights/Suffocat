package builtin

import lua "github.com/yuin/gopher-lua"

func getLogFn(L *lua.LState) func(string, string) {
	if s := getStateCtx(L); s != nil {
		return s.logFn
	}
	return func(string, string) {}
}

func loggerDebug(L *lua.LState) int {
	getLogFn(L)("DEBUG", L.CheckString(1))
	return 0
}

func loggerInfo(L *lua.LState) int {
	getLogFn(L)("INFO", L.CheckString(1))
	return 0
}

func loggerWarn(L *lua.LState) int {
	getLogFn(L)("WARN", L.CheckString(1))
	return 0
}

func loggerError(L *lua.LState) int {
	getLogFn(L)("ERROR", L.CheckString(1))
	return 0
}
