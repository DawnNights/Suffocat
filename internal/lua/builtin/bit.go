package builtin

import lua "github.com/yuin/gopher-lua"

func bitBand(L *lua.LState) int {
	n := L.GetTop()
	if n == 0 {
		L.Push(lua.LNumber(0))
		return 1
	}
	result := uint32(L.CheckInt(1))
	for i := 2; i <= n; i++ {
		result &= uint32(L.CheckInt(i))
	}
	L.Push(lua.LNumber(result))
	return 1
}

func bitBor(L *lua.LState) int {
	n := L.GetTop()
	if n == 0 {
		L.Push(lua.LNumber(0))
		return 1
	}
	result := uint32(L.CheckInt(1))
	for i := 2; i <= n; i++ {
		result |= uint32(L.CheckInt(i))
	}
	L.Push(lua.LNumber(result))
	return 1
}

func bitBxor(L *lua.LState) int {
	n := L.GetTop()
	if n == 0 {
		L.Push(lua.LNumber(0))
		return 1
	}
	result := uint32(L.CheckInt(1))
	for i := 2; i <= n; i++ {
		result ^= uint32(L.CheckInt(i))
	}
	L.Push(lua.LNumber(result))
	return 1
}

func bitBnot(L *lua.LState) int {
	a := uint32(L.CheckInt(1))
	L.Push(lua.LNumber(^a))
	return 1
}

func bitLshift(L *lua.LState) int {
	a := uint32(L.CheckInt(1))
	n := uint(L.CheckInt(2))
	L.Push(lua.LNumber(a << n))
	return 1
}

func bitRshift(L *lua.LState) int {
	a := uint32(L.CheckInt(1))
	n := uint(L.CheckInt(2))
	L.Push(lua.LNumber(a >> n))
	return 1
}

func bitArshift(L *lua.LState) int {
	a := int32(L.CheckInt(1))
	n := uint(L.CheckInt(2))
	L.Push(lua.LNumber(a >> n))
	return 1
}
