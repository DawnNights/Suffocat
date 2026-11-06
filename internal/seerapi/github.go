package seerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// SeerAPI 官方数据仓库中的分片数据。
// 该仓库与 API 使用同一份数据，适合首次批量导入，避免逐条请求触发限流。
const dataRepoBase = "https://raw.githubusercontent.com/SeerAPI/api-data/main/data/v1/sharded_data"
const petDataRepoBase = dataRepoBase + "/pet"

type shardIndex struct {
	ShardCount int `json:"shard_count"`
	Shards     map[string]struct {
		Path  string `json:"path"`
		Count int    `json:"count"`
	} `json:"shards"`
	ByID map[string]string `json:"by_id"`
}

// missingPetShards 对比本地已有详情的精灵 ID，返回缺失 ID 及其所在分片。
func missingPetShards(index shardIndex, existing map[uint32]bool) (map[uint32]bool, map[string]bool) {
	missing := make(map[uint32]bool)
	shards := make(map[string]bool)

	for idText, shardKey := range index.ByID {
		id, err := strconv.ParseUint(idText, 10, 32)
		if err != nil || id == 0 || existing[uint32(id)] {
			continue
		}
		if _, ok := index.Shards[shardKey]; !ok {
			continue
		}
		missing[uint32(id)] = true
		shards[shardKey] = true
	}
	return missing, shards
}

// SyncMissingPetsFromDataRepo 只下载包含缺失精灵 ID 的分片。
func (c *Client) SyncMissingPetsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	if c == nil {
		return nil
	}

	client := &http.Client{Timeout: 90 * time.Second}
	indexBody, err := fetchRaw(ctx, client, petDataRepoBase+"/id-index.json")
	if err != nil {
		return fmt.Errorf("获取精灵分片索引失败: %w", err)
	}

	var index shardIndex
	if err := json.Unmarshal(indexBody, &index); err != nil {
		return fmt.Errorf("解析精灵分片索引失败: %w", err)
	}

	missingIDs, shardSet := missingPetShards(index, c.pets.DetailIDs())
	if len(shardSet) == 0 {
		return nil
	}

	keys := make([]string, 0, len(shardSet))
	for key := range shardSet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	total := len(keys)
	if progress != nil {
		progress(0, total)
	}

	var firstErr error
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}

		shardURL := petDataRepoBase + "/" + index.Shards[key].Path
		body, err := fetchRaw(ctx, client, shardURL)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("下载分片 %s 失败: %w", key, err)
			}
			continue
		}

		var records map[string]json.RawMessage
		if err := json.Unmarshal(body, &records); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("解析分片 %s 失败: %w", key, err)
			}
			continue
		}

		pets := make([]Pet, 0, len(missingIDs))
		for recordKey, record := range records {
			if recordKey == "hash" {
				continue
			}
			var raw apiPet
			if err := json.Unmarshal(record, &raw); err != nil {
				continue
			}
			pet := raw.toPet()
			if pet.ID == 0 || !missingIDs[pet.ID] {
				continue
			}
			pets = append(pets, pet)
		}

		if err := c.pets.BulkPut(pets); err != nil {
			return err
		}
		if progress != nil {
			progress(i+1, total)
		}
	}

	return firstErr
}

// SyncPetsFromDataRepo 从 SeerAPI 官方数据仓库批量导入精灵数据。
// 只下载 18 个分片文件，远快于逐条调用 API。
func (c *Client) SyncPetsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	client := &http.Client{Timeout: 90 * time.Second}

	indexBody, err := fetchRaw(ctx, client, petDataRepoBase+"/id-index.json")
	if err != nil {
		return fmt.Errorf("获取精灵分片索引失败: %w", err)
	}

	var index shardIndex
	if err := json.Unmarshal(indexBody, &index); err != nil {
		return fmt.Errorf("解析精灵分片索引失败: %w", err)
	}

	keys := make([]string, 0, len(index.Shards))
	for key := range index.Shards {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	total := len(keys)
	if progress != nil {
		progress(0, total)
	}

	var firstErr error

	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}

		shardURL := petDataRepoBase + "/" + index.Shards[key].Path
		body, err := fetchRaw(ctx, client, shardURL)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("下载分片 %s 失败: %w", key, err)
			}
			continue
		}

		// 分片顶层是 ID -> 精灵对象，另有一个 "hash" 字符串字段。
		var records map[string]json.RawMessage
		if err := json.Unmarshal(body, &records); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("解析分片 %s 失败: %w", key, err)
			}
			continue
		}

		pets := make([]Pet, 0, len(records))
		for recordKey, record := range records {
			if recordKey == "hash" {
				continue
			}
			var raw apiPet
			if err := json.Unmarshal(record, &raw); err != nil {
				continue
			}
			pet := raw.toPet()
			if pet.ID == 0 {
				continue
			}
			pets = append(pets, pet)
		}

		if err := c.pets.BulkPut(pets); err != nil {
			return err
		}

		if progress != nil {
			progress(i+1, total)
		}
	}

	if firstErr != nil {
		return firstErr
	}
	return nil
}

// SyncAllFromDataRepo 批量导入精灵、技能、道具、刻印、套装与部件。
func (c *Client) SyncAllFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	var errs []error

	if err := c.SyncPetsFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("精灵: %w", err))
	}
	if err := c.SyncSkillsFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("技能: %w", err))
	}
	if err := c.SyncItemsFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("道具: %w", err))
	}
	if err := c.SyncMintMarksFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("刻印: %w", err))
	}
	if err := c.SyncSuitsFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("套装: %w", err))
	}
	if err := c.SyncEquipsFromDataRepo(ctx, progress); err != nil {
		errs = append(errs, fmt.Errorf("部件: %w", err))
	}

	return errors.Join(errs...)
}

func (c *Client) SyncSkillsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	return syncRepoEntity(ctx, "skill", c.skills, func(raw json.RawMessage) (Skill, bool) {
		var item apiSkill
		if json.Unmarshal(raw, &item) != nil {
			return Skill{}, false
		}
		converted := item.toSkill()
		return converted, converted.ID != 0
	}, progress)
}

func (c *Client) SyncItemsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	return syncRepoEntity(ctx, "item", c.items, func(raw json.RawMessage) (Item, bool) {
		var item apiItem
		if json.Unmarshal(raw, &item) != nil {
			return Item{}, false
		}
		converted := item.toItem()
		return converted, converted.ID != 0
	}, progress)
}

func (c *Client) SyncMintMarksFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	return syncRepoEntity(ctx, "mintmark", c.mintmarks, func(raw json.RawMessage) (MintMark, bool) {
		var item apiMintMark
		if json.Unmarshal(raw, &item) != nil {
			return MintMark{}, false
		}
		converted := item.toMintMark()
		return converted, converted.ID != 0
	}, progress)
}

func (c *Client) SyncSuitsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	return syncRepoEntity(ctx, "suit", c.suits, func(raw json.RawMessage) (Suit, bool) {
		var item apiSuit
		if json.Unmarshal(raw, &item) != nil {
			return Suit{}, false
		}
		converted := item.toSuit()
		return converted, converted.ID != 0
	}, progress)
}

func (c *Client) SyncEquipsFromDataRepo(ctx context.Context, progress func(done, total int)) error {
	return syncRepoEntity(ctx, "equip", c.equips, func(raw json.RawMessage) (Equip, bool) {
		var item apiEquip
		if json.Unmarshal(raw, &item) != nil {
			return Equip{}, false
		}
		converted := item.toEquip()
		return converted, converted.ID != 0
	}, progress)
}

func syncRepoEntity[T any](
	ctx context.Context,
	entity string,
	store *tableStore[T],
	convert func(json.RawMessage) (T, bool),
	progress func(done, total int),
) error {
	client := &http.Client{Timeout: 90 * time.Second}
	base := dataRepoBase + "/" + entity

	indexBody, err := fetchRaw(ctx, client, base+"/id-index.json")
	if err != nil {
		return fmt.Errorf("获取分片索引失败: %w", err)
	}

	var index shardIndex
	if err := json.Unmarshal(indexBody, &index); err != nil {
		return fmt.Errorf("解析分片索引失败: %w", err)
	}

	keys := make([]string, 0, len(index.Shards))
	for key := range index.Shards {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	total := len(keys)
	if progress != nil {
		progress(0, total)
	}

	var firstErr error
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}

		body, err := fetchRaw(ctx, client, base+"/"+index.Shards[key].Path)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("下载分片 %s 失败: %w", key, err)
			}
			continue
		}

		var records map[string]json.RawMessage
		if err := json.Unmarshal(body, &records); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("解析分片 %s 失败: %w", key, err)
			}
			continue
		}

		values := make([]T, 0, len(records))
		for recordKey, record := range records {
			if recordKey == "hash" {
				continue
			}
			value, ok := convert(record)
			if !ok {
				continue
			}
			values = append(values, value)
		}

		if err := store.BulkPut(values); err != nil {
			return err
		}

		if progress != nil {
			progress(i+1, total)
		}
	}

	return firstErr
}

func fetchRaw(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Suffocat/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		return body, nil
	}

	return nil, lastErr
}
