package common

import (
	"io"
	"net/http"
	"strings"
)

func HttpGets(url string) (string, error) {
	body, err := HttpGet(url)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func HttpGet(url string) ([]byte, error) {
	rsp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func StringBetween(text string, start string, end string) string {
	s := strings.Index(text, start)
	if s == -1 {
		return ""
	}

	e := strings.Index(text[s:], end)
	if e == -1 {
		return ""
	}

	return text[s+len(start) : s+e]
}
