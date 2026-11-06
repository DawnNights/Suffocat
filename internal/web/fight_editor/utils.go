package fight_editor

import (
	"os"
	"path/filepath"
	"strings"
)

type childNode struct {
	Name  string
	Files []string
}

type rootNode struct {
	Files     []string
	ChildNode []childNode
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml"
}

// validSegment 拒绝路径分隔符与上跳目录，保证文件操作限制在方案目录内。
func validSegment(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}

func planPath(childName, file string) (string, bool) {
	if file == "" || !validSegment(file) {
		return "", false
	}
	if childName != "" && !validSegment(childName) {
		return "", false
	}
	return filepath.Join(rootPath, childName, file), true
}

func childDir(childName string) (string, bool) {
	if childName == "" || !validSegment(childName) {
		return "", false
	}
	return filepath.Join(rootPath, childName), true
}

func buildRootNode(root string) *rootNode {
	node := &rootNode{
		Files:     []string{},
		ChildNode: []childNode{},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		// 根目录出错，返回空结构
		return node
	}

	for _, entry := range entries {
		fullPath := filepath.Join(root, entry.Name())

		// 根目录下的 yaml 文件
		if !entry.IsDir() {
			if isYAML(entry.Name()) {
				node.Files = append(node.Files, entry.Name())
			}
			continue
		}

		// 子目录节点（只一层）
		child := childNode{
			Name:  entry.Name(),
			Files: []string{}, // 已处理但为空
		}

		subEntries, err := os.ReadDir(fullPath)
		if err == nil {
			for _, sub := range subEntries {
				if !sub.IsDir() && isYAML(sub.Name()) {
					child.Files = append(child.Files, sub.Name())
				}
			}
		}
		node.ChildNode = append(node.ChildNode, child)
	}

	return node
}
