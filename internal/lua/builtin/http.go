package builtin

import (
	"context"
	"io"
	"net/http"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// httpGet 发起 HTTP GET 请求，返回响应体字符串（非字符串内容返回空串）。
// 用法：http.get(url [, header])
func httpGet(L *lua.LState) int {
	url := L.CheckString(1)

	ctx := L.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}

	// 可选 header 参数
	if L.GetTop() >= 2 {
		if tbl, ok := L.Get(2).(*lua.LTable); ok {
			tbl.ForEach(func(key, value lua.LValue) {
				req.Header.Set(key.String(), value.String())
			})
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}

	L.Push(lua.LString(string(body)))
	return 1
}

// httpPost 发起 HTTP POST 请求，返回响应体字符串（非字符串内容返回空串）。
// 用法：http.post(url, data [, header])
func httpPost(L *lua.LState) int {
	url := L.CheckString(1)
	data := L.CheckString(2)

	ctx := L.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data))
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 可选 header 参数
	if L.GetTop() >= 3 {
		if tbl, ok := L.Get(3).(*lua.LTable); ok {
			tbl.ForEach(func(key, value lua.LValue) {
				req.Header.Set(key.String(), value.String())
			})
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}

	L.Push(lua.LString(string(body)))
	return 1
}
