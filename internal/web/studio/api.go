package studio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	luabind "suffocat/internal/lua"
	"suffocat/internal/lua/builtin"
)

// ---------- 通用响应 ----------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(v)
}

// ---------- 脚本列表与内容 ----------

func apiListScripts(w http.ResponseWriter, r *http.Request) {
	scripts := listScripts()

	if filter := r.URL.Query().Get("type"); filter == "UNITY" || filter == "FLASH" {
		filtered := make([]ScriptMeta, 0, len(scripts))
		for _, s := range scripts {
			if s.Type == filter {
				filtered = append(filtered, s)
			}
		}
		scripts = filtered
	}
	writeJSON(w, scripts)
}

func apiReadScript(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	source, err := readScriptSource(file)
	if err != nil {
		writeError(w, http.StatusNotFound, "读取失败: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(source))
}

func apiSaveScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var req struct {
		File   string `json:"file"`
		Script string `json:"script"`
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
			return
		}
	} else {
		r.ParseForm()
		req.File = r.FormValue("file")
		req.Script = r.FormValue("script")
	}

	if req.File == "" {
		writeError(w, http.StatusBadRequest, "file 参数为空")
		return
	}

	name, err := writeScriptFile(req.File, req.Script)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败: "+err.Error())
		return
	}

	writeJSON(w, map[string]any{"ok": true, "file": name})
}

func apiDeleteScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 DELETE")
		return
	}

	file := r.URL.Query().Get("file")
	if file == "" {
		writeError(w, http.StatusBadRequest, "缺少 file 参数")
		return
	}

	if err := deleteScriptFile(file); err != nil {
		writeError(w, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func apiRenameScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var req struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "请求格式错误")
			return
		}
	} else {
		r.ParseForm()
		req.Old = r.FormValue("old")
		req.New = r.FormValue("new")
	}

	if req.Old == "" || req.New == "" {
		writeError(w, http.StatusBadRequest, "参数缺失")
		return
	}

	name, err := renameScriptFile(req.Old, req.New)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "重命名失败: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "file": name})
}

func apiSetEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	file := r.URL.Query().Get("file")
	if file == "" {
		writeError(w, http.StatusBadRequest, "缺少 file 参数")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := setScriptEnabled(file, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func apiSaveOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var body struct {
		Files []string `json:"files"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := saveScriptOrder(body.Files); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func apiDeleteUnchecked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	filter := r.URL.Query().Get("type")
	deleted := 0

	for _, meta := range listScripts() {
		if meta.Enabled {
			continue
		}
		if (filter == "UNITY" || filter == "FLASH") && meta.Type != filter {
			continue
		}
		if err := deleteScriptFile(meta.File); err == nil {
			deleted++
		}
	}

	writeJSON(w, map[string]any{"ok": true, "deleted": deleted})
}

// ---------- 配置 ----------

type ConfigDTO struct {
	Value any `json:"value"`
}

func apiConfig(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		writeError(w, http.StatusBadRequest, "缺少 file 参数")
		return
	}

	switch r.Method {
	case http.MethodGet:
		defs, err := getScriptConfigDef(file)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, defs)

	case http.MethodPost:
		var values map[string]ConfigDTO
		if err := decodeJSON(r, &values); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 解析失败")
			return
		}

		defs, err := getScriptConfigDef(file)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}

		for i := range defs {
			if dto, ok := values[defs[i].Key]; ok {
				defs[i].Value = normalizeConfigValue(dto.Value, defs[i].Type)
			}
		}

		if err := saveScriptConfig(file, defs); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
	}
}

// ---------- 运行控制 ----------

func apiRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Type     string `json:"type"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if body.Account == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "请输入米米号和密码")
		return
	}
	if body.Type == "ALL" {
		body.Type = ""
	}
	if body.Type != "" && body.Type != "UNITY" && body.Type != "FLASH" {
		writeError(w, http.StatusBadRequest, "脚本类型无效")
		return
	}

	if err := runScripts(body.Account, body.Password, body.Type); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]string{"ok": "started"})
}

func apiRunSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		File     string `json:"file"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if body.Account == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "请输入米米号和密码")
		return
	}
	if body.File == "" {
		writeError(w, http.StatusBadRequest, "请指定脚本文件")
		return
	}

	if err := runSingleScript(body.Account, body.Password, body.File); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]string{"ok": "started"})
}

func apiStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}
	stopEngine()
	writeJSON(w, map[string]any{"ok": true})
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, getStatus())
}

// ---------- 数据同步 ----------

func apiDataStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, getStatus())
}

func apiDataSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}
	started := startSync()
	writeJSON(w, map[string]any{"ok": true, "started": started})
}

// ---------- 导入 ----------

func apiImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "请使用 POST")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "解析上传数据失败: "+err.Error())
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "未选择文件")
		return
	}

	type importResult struct {
		File   string `json:"file"`
		Type   string `json:"type"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]importResult, 0, len(files))

	for _, fh := range files {
		result := importResult{File: fh.Filename}

		if !strings.EqualFold(filepath.Ext(fh.Filename), ".lua") {
			result.Status = "error"
			result.Error = "只支持 .lua 明文脚本"
			results = append(results, result)
			continue
		}

		file, err := fh.Open()
		if err != nil {
			result.Status = "error"
			result.Error = "读取文件失败: " + err.Error()
			results = append(results, result)
			continue
		}
		raw, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			result.Status = "error"
			result.Error = "读取文件失败: " + err.Error()
			results = append(results, result)
			continue
		}

		L := builtin.NewLuaState()
		script := &luabind.LuaScript{}
		parseErr := script.FromSource(string(raw), L)
		L.Close()
		if parseErr != nil {
			result.Status = "error"
			result.Error = "脚本解析失败: " + parseErr.Error()
			results = append(results, result)
			continue
		}

		baseName := filepath.Base(strings.ReplaceAll(fh.Filename, "\\", "/"))
		name, err := uniqueScriptName(baseName)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		if _, err := writeScriptFile(name, string(raw)); err != nil {
			result.Status = "error"
			result.Error = "保存失败: " + err.Error()
			results = append(results, result)
			continue
		}

		result.File = name
		result.Type = strings.ToUpper(strings.TrimSpace(script.Type))
		result.Status = "ok"
		results = append(results, result)
	}

	writeJSON(w, results)
}

func uniqueScriptName(base string) (string, error) {
	name, err := safeScriptName(base)
	if err != nil {
		return "", err
	}
	name = stripScriptPrefix(name)

	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	candidate := name
	for i := 1; i < 1000; i++ {
		if !scriptExists(candidate) {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
	return "", fmt.Errorf("同名文件过多")
}
