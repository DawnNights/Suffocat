package studio

import (
	"context"
	"fmt"
	"sync"

	"suffocat/internal/seerapi"
)

var (
	syncMu         sync.Mutex
	syncRunning    bool
	syncDoneCount  int
	syncTotalCount int
)

// startSync 在后台从 SeerAPI 官方数据仓库批量刷新全部数据。
func startSync() bool {
	syncMu.Lock()
	if syncRunning {
		syncMu.Unlock()
		return false
	}
	syncRunning = true
	syncDoneCount = 0
	syncTotalCount = 0
	syncMu.Unlock()

	go func() {
		defer func() {
			syncMu.Lock()
			syncRunning = false
			syncMu.Unlock()
			studioLog("INFO", "数据同步结束")
		}()

		client := seerapi.Default()
		if client == nil {
			studioLog("ERROR", "数据客户端未初始化")
			return
		}

		studioLog("INFO", "开始同步数据...")
		err := client.SyncAllFromDataRepo(context.Background(), func(done, total int) {
			syncMu.Lock()
			syncDoneCount = done
			syncTotalCount = total
			syncMu.Unlock()
		})
		if err != nil {
			studioLog("ERROR", "数据同步失败: "+err.Error())
			return
		}
		if err := client.EnsureFactorData(context.Background(), true); err != nil {
			studioLog("ERROR", "因子数据同步失败: "+err.Error())
			return
		}
		if err := client.StoreConfigVersion(context.Background()); err != nil {
			studioLog("WARN", "保存游戏配置版本失败: "+err.Error())
		}
		studioLog("INFO", fmt.Sprintf("数据同步完成，本地精灵缓存数量: %d", client.PetCacheCount()))
	}()

	return true
}
