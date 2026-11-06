package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"suffocat/internal/config"
	"suffocat/internal/seerapi"
	"suffocat/internal/web"
	"suffocat/internal/web/fight_editor"
	"suffocat/internal/web/studio"
)

func main() {
	addr := flag.String("addr", ":4604", "HTTP 监听地址")
	root := flag.String("root", ".", "项目根目录（运行时数据统一位于 data/）")
	resetCache := flag.Bool("reset-cache", false, "启动前删除本地 SQLite 缓存")
	flag.Parse()

	if err := config.SetRoot(*root); err != nil {
		log.Fatalf("设置根目录失败: %v", err)
	}
	if err := config.EnsureDirs(); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	if *resetCache {
		for _, name := range []string{"seerapi.db-wal", "seerapi.db-shm", "seerapi.db"} {
			if err := os.Remove(filepath.Join(config.DBDir(), name)); err != nil && !os.IsNotExist(err) {
				log.Printf("删除缓存文件 %s 失败: %v", name, err)
			}
		}
		log.Println("已清空本地数据缓存")
	}

	client, err := seerapi.New(config.DBDir())
	if err != nil {
		log.Fatalf("初始化数据缓存失败: %v", err)
	}
	defer client.Close()
	seerapi.SetDefault(client)

	if !client.HasPetIndex() {
		// 首次运行时只拉取一份轻量的精灵名称索引（单个文件），
		// 精灵详情、技能等仍按需获取。
		go func() {
			if err := client.EnsurePetIndex(context.Background(), false); err != nil {
				log.Printf("精灵名称索引获取失败（可稍后重试）: %v", err)
			}
		}()
	}

	// 启动前完成游戏数据检查与更新，失败直接退出，避免带着不完整的数据运行。
	log.Println("检查游戏配置版本...")
	if err := client.EnsureGameData(context.Background(), func(done, total int) {
		log.Printf("游戏数据更新: %d/%d", done, total)
	}); err != nil {
		client.Close()
		log.Fatalf("游戏数据检查失败: %v", err)
	}

	go web.BroadCaster()
	http.HandleFunc("/ws", web.WsHandler)

	fight_editor.Init()
	studio.Init()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/studio/", http.StatusFound)
	})

	fmt.Printf("Suffocat 已启动: http://127.0.0.1%s/studio/\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
