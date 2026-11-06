package seerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// 精灵名称索引：一次下载即可获得全部 ID -> 名称，不包含技能等详情。
const petNameIndexURL = dataRepoBase + "/pet/name-index.json"

const (
	metaPetIndexHash = "pet_name_index_hash"
	metaPetIndexTime = "pet_name_index_time"
)

type petNameIndex struct {
	ByName map[string][]uint32 `json:"by_name"`
	Hash   string              `json:"hash"`
}

// EnsurePetIndex 确保本地存在精灵名称索引。
// force 为 false 时，24 小时内不会重复下载。
func (c *Client) EnsurePetIndex(ctx context.Context, force bool) error {
	if c == nil {
		return nil
	}

	c.indexMu.Lock()
	defer c.indexMu.Unlock()

	if !force && c.pets.NameCount() > 0 {
		if ts := c.metaValue(metaPetIndexTime); ts != "" {
			if parsed, err := strconv.ParseInt(ts, 10, 64); err == nil {
				if time.Since(time.Unix(parsed, 0)) < 24*time.Hour {
					return nil
				}
			}
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	body, err := fetchRaw(ctx, client, petNameIndexURL)
	if err != nil {
		return fmt.Errorf("下载精灵名称索引失败: %w", err)
	}

	var index petNameIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return fmt.Errorf("解析精灵名称索引失败: %w", err)
	}

	names := make(map[uint32]string, len(index.ByName))
	for name, ids := range index.ByName {
		for _, id := range ids {
			names[id] = name
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("精灵名称索引为空")
	}

	if err := c.pets.SetNames(names); err != nil {
		return err
	}

	_ = c.setMeta(metaPetIndexHash, index.Hash)
	_ = c.setMeta(metaPetIndexTime, strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func (c *Client) metaValue(key string) string {
	var value string
	if c.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value) != nil {
		return ""
	}
	return value
}

func (c *Client) setMeta(key, value string) error {
	_, err := c.db.Exec(`INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
