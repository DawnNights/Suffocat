package fight_editor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"suffocat/internal/seerapi"
)

// -------------- 数据接口 --------------

// allMonsters 返回以精灵 ID 为下标的名字数组，兼容原编辑器前端。
func allMonsters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	client := seerapi.Default()
	if client == nil {
		w.Write([]byte("[]"))
		return
	}

	petNames, err := json.Marshal(client.AllPetNames())
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	w.Write(petNames)
}

// getMovesById 返回指定精灵可学习的技能列表。
func getMovesById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	petID, err := strconv.ParseUint(r.URL.Query().Get("petId"), 10, 32)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}

	client := seerapi.Default()
	if client == nil {
		w.Write([]byte("[]"))
		return
	}

	skills := client.PetSkills(uint32(petID))
	moves := make([]struct {
		ID   uint32 `json:"ID"`
		Name string `json:"Name"`
	}, 0, len(skills))

	for _, skill := range skills {
		moves = append(moves, struct {
			ID   uint32 `json:"ID"`
			Name string `json:"Name"`
		}{ID: skill.ID, Name: skill.Name})
	}

	out, err := json.Marshal(moves)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	w.Write(out)
}

// getEquipsById 返回套装包含的部件 ID 列表。
func getEquipsById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	suitID, err := strconv.ParseUint(r.URL.Query().Get("suitId"), 10, 32)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}

	client := seerapi.Default()
	if client == nil {
		w.Write([]byte("[]"))
		return
	}

	ids := client.SuitEquipIDs(uint32(suitID))
	out, err := json.Marshal(ids)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	w.Write(out)
}

// ------------- 文件操作接口 --------------

func allScripts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data, err := json.Marshal(buildRootNode(rootPath))
	if err != nil {
		w.Write([]byte(`{"Files":[],"ChildNode":[]}`))
		return
	}

	w.Write(data)
}

func readScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	query := r.URL.Query()

	path, ok := planPath(query.Get("childName"), query.Get("file"))
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	scriptData, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(w, empty_yaml, strings.Replace(query.Get("file"), ".yaml", "", 1))
		return
	}

	w.Write(scriptData)
}

func writeScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("请使用POST请求"))
		return
	}

	// FormData 会自动填充 r.FormValue，无需 ParseForm()
	childName := r.FormValue("childName")
	file := r.FormValue("file")
	yamlText := r.FormValue("yaml")

	if file == "" {
		w.Write([]byte("file 参数为空"))
		return
	}

	if yamlText == "" {
		yamlText = fmt.Sprintf(empty_yaml, strings.Replace(file, ".yaml", "", 1))
	}

	path, ok := planPath(childName, file)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		w.Write([]byte("创建目录失败：" + err.Error()))
		return
	}
	if err := os.WriteFile(path, []byte(yamlText), 0644); err != nil {
		w.Write([]byte("写入失败：" + err.Error()))
		return
	}

	w.Write([]byte("ok"))
}

func deleteScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("请使用POST请求"))
		return
	}

	childName := r.FormValue("childName")
	file := r.FormValue("file")

	if file == "" {
		w.Write([]byte("file 参数为空"))
		return
	}

	path, ok := planPath(childName, file)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	if _, err := os.Stat(path); err == nil {
		err := os.Remove(path)
		if err == nil {
			w.Write([]byte("ok"))
		} else {
			w.Write([]byte(err.Error()))
		}
	} else {
		w.Write([]byte(err.Error()))
	}
}

func createChild(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path, ok := childDir(r.URL.Query().Get("childName"))
	if !ok {
		w.Write([]byte("非法目录名"))
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}

	w.Write([]byte(`ok`))
}

func deleteChild(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path, ok := childDir(r.URL.Query().Get("childName"))
	if !ok {
		w.Write([]byte("非法目录名"))
		return
	}
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			w.Write([]byte(`对象不为目录`))
			return
		}

		err := os.Remove(path)
		if err == nil {
			w.Write([]byte(`ok`))
		} else {
			w.Write([]byte(`无法删除包含脚本文件的子目录`))
		}
	} else if os.IsNotExist(err) {
		w.Write([]byte(`目录不存在`))
	} else {
		w.Write([]byte(err.Error()))
	}
}

func moveScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("请使用POST请求"))
		return
	}

	fromChild := r.FormValue("fromChild")
	toChild := r.FormValue("toChild")
	file := r.FormValue("file")

	if file == "" {
		w.Write([]byte("file 参数为空"))
		return
	}

	srcPath, ok := planPath(fromChild, file)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	dstPath, ok := planPath(toChild, file)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		w.Write([]byte("源文件不存在"))
		return
	}

	if _, err := os.Stat(dstPath); err == nil {
		w.Write([]byte("目标位置已存在同名文件"))
		return
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		w.Write([]byte("移动失败：" + err.Error()))
		return
	}

	w.Write([]byte("ok"))
}

func renameScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("请使用POST请求"))
		return
	}

	childName := r.FormValue("childName")
	oldFile := r.FormValue("oldFile")
	newFile := r.FormValue("newFile")

	if oldFile == "" || newFile == "" {
		w.Write([]byte("参数为空"))
		return
	}

	srcPath, ok := planPath(childName, oldFile)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}
	dstPath, ok := planPath(childName, newFile)
	if !ok {
		w.Write([]byte("非法文件名"))
		return
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		w.Write([]byte("源文件不存在"))
		return
	}

	if _, err := os.Stat(dstPath); err == nil {
		w.Write([]byte("同名文件已存在"))
		return
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		w.Write([]byte("重命名失败：" + err.Error()))
		return
	}

	w.Write([]byte("ok"))
}

func renameChild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("请使用POST请求"))
		return
	}

	oldName := r.FormValue("oldName")
	newName := r.FormValue("newName")

	if oldName == "" || newName == "" {
		w.Write([]byte("参数为空"))
		return
	}

	srcPath, ok := childDir(oldName)
	if !ok {
		w.Write([]byte("非法目录名"))
		return
	}
	dstPath, ok := childDir(newName)
	if !ok {
		w.Write([]byte("非法目录名"))
		return
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		w.Write([]byte("源目录不存在"))
		return
	}

	if _, err := os.Stat(dstPath); err == nil {
		w.Write([]byte("同名目录已存在"))
		return
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		w.Write([]byte("重命名失败：" + err.Error()))
		return
	}

	w.Write([]byte("ok"))
}
