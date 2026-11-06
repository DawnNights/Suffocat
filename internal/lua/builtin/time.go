package builtin

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

type date struct {
	Year   int
	Mouth  int
	Day    int
	Hour   int
	Minute int
	Second int
}

func getDate() date {
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	return date{
		Year:   int(now.Year()),
		Mouth:  int(now.Month()),
		Day:    int(now.Day()),
		Hour:   int(now.Hour()),
		Minute: int(now.Minute()),
		Second: int(now.Second()),
	}
}

// time.Date() -> table
func timeDate(L *lua.LState) int {
	d := getDate()
	tbl := L.NewTable()
	tbl.RawSetString("Year", lua.LNumber(d.Year))
	tbl.RawSetString("Mouth", lua.LNumber(d.Mouth))
	tbl.RawSetString("Day", lua.LNumber(d.Day))
	tbl.RawSetString("Hour", lua.LNumber(d.Hour))
	tbl.RawSetString("Minute", lua.LNumber(d.Minute))
	tbl.RawSetString("Second", lua.LNumber(d.Second))
	L.Push(tbl)
	return 1
}

// time.Sleep(second)
func timeSleep(L *lua.LState) int {
	second := L.CheckInt(1)
	d := time.Duration(second) * time.Second
	ctx := L.Context()
	if ctx != nil {
		select {
		case <-time.After(d):
		case <-ctx.Done():
		}
	} else {
		time.Sleep(d)
	}
	return 0
}

// time.Unix() -> integer
func timeUnix(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

// time.UnixMilli() -> integer
func timeUnixMilli(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().UnixMilli()))
	return 1
}
