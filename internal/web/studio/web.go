package studio

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var indexHTML []byte

func Init() {
	http.HandleFunc("/studio/", servePage)

	// 脚本管理
	http.HandleFunc("/studio/scripts", apiListScripts)
	http.HandleFunc("/studio/script", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			apiReadScript(w, r)
		case http.MethodPost:
			apiSaveScript(w, r)
		case http.MethodDelete:
			apiDeleteScript(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
		}
	})
	http.HandleFunc("/studio/rename", apiRenameScript)
	http.HandleFunc("/studio/enabled", apiSetEnabled)
	http.HandleFunc("/studio/order", apiSaveOrder)
	http.HandleFunc("/studio/config", apiConfig)
	http.HandleFunc("/studio/delete_unchecked", apiDeleteUnchecked)
	http.HandleFunc("/studio/import", apiImport)

	// 运行控制
	http.HandleFunc("/studio/run", apiRun)
	http.HandleFunc("/studio/run_single", apiRunSingle)
	http.HandleFunc("/studio/stop", apiStop)
	http.HandleFunc("/studio/status", apiStatus)

	// 数据缓存
	http.HandleFunc("/studio/data/status", apiDataStatus)
	http.HandleFunc("/studio/data/sync", apiDataSync)

	// 因子任务
	http.HandleFunc("/studio/factors", apiFactors)
	http.HandleFunc("/studio/factor/enabled", apiSetFactorEnabled)
	http.HandleFunc("/studio/factor/settings", apiFactorSettings)
	http.HandleFunc("/studio/factor/run", apiRunFactor)
	http.HandleFunc("/studio/factor/run_category", apiRunFactorCategory)
}

func servePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/studio/" && r.URL.Path != "/studio" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}
