package fight_editor

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"

	"suffocat/internal/config"
)

// rootPath 是战斗方案编辑器使用的唯一目录。
var rootPath string

//go:embed files/editor.html
var html_fight_editor []byte

//go:embed files/manager.html
var html_fight_manager []byte

//go:embed files/tailwindcss.js
var js_tailwindcss []byte

//go:embed files/js-yaml.min.js
var js_yaml_min []byte

//go:embed files/empty.txt
var empty_yaml string

//go:embed files/艾欧鲁斯王.yaml
var default_fight_script []byte

func Init() {
	rootPath = config.PlansDir()
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return
	}

	scriptFile := filepath.Join(rootPath, "艾欧鲁斯王.yaml")
	if _, err := os.Stat(scriptFile); os.IsNotExist(err) {
		os.WriteFile(scriptFile, default_fight_script, 0644)
	}

	// 静态资源
	http.HandleFunc("/fight_editor", fightEditor)
	http.HandleFunc("/fight_manager", fightManager)
	http.HandleFunc("/fight_editor/tailwindcss.js", tailWindCss)
	http.HandleFunc("/fight_editor/js-yaml.min.js", jsYamlMin)

	// 数据接口
	http.HandleFunc("/fight_editor/all_monsters", allMonsters)
	http.HandleFunc("/fight_editor/get_moves", getMovesById)
	http.HandleFunc("/fight_editor/get_equips", getEquipsById)

	// 文件操作接口
	http.HandleFunc("/fight_editor/all_scripts", allScripts)
	http.HandleFunc("/fight_editor/read_script", readScript)
	http.HandleFunc("/fight_editor/write_script", writeScript)
	http.HandleFunc("/fight_editor/delete_script", deleteScript)

	http.HandleFunc("/fight_editor/create_child", createChild)
	http.HandleFunc("/fight_editor/delete_child", deleteChild)
	http.HandleFunc("/fight_editor/move_script", moveScript)
	http.HandleFunc("/fight_editor/rename_script", renameScript)
	http.HandleFunc("/fight_editor/rename_child", renameChild)
}

func fightManager(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html_fight_manager)
}

func fightEditor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	query := r.URL.Query()

	if query.Get("file") == "" {
		w.Write([]byte("未选择文件"))
		return
	}

	path, ok := planPath(query.Get("childName"), query.Get("file"))
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	if _, err := os.Stat(path); err != nil {
		w.Write([]byte("文件读取异常：" + err.Error()))
		return
	}

	w.Write(html_fight_editor)
}

func tailWindCss(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Write(js_tailwindcss)
}

func jsYamlMin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Write(js_yaml_min)
}
