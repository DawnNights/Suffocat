package seerapi

import "testing"

func TestMissingPetShards(t *testing.T) {
	index := shardIndex{
		Shards: map[string]struct {
			Path  string `json:"path"`
			Count int    `json:"count"`
		}{
			"0000": {Path: "id/shards/0000.json", Count: 2},
			"0001": {Path: "id/shards/0001.json", Count: 1},
		},
		ByID: map[string]string{
			"1": "0000",
			"2": "0000",
			"3": "0001",
			"4": "9999", // 索引中不存在的分片应被忽略
		},
	}

	missing, shards := missingPetShards(index, map[uint32]bool{1: true})
	if len(missing) != 2 || !missing[2] || !missing[3] {
		t.Fatalf("缺失 ID 计算错误: %+v", missing)
	}
	if len(shards) != 2 || !shards["0000"] || !shards["0001"] {
		t.Fatalf("分片选择错误: %+v", shards)
	}

	missing, shards = missingPetShards(index, map[uint32]bool{1: true, 2: true, 3: true})
	if len(missing) != 0 || len(shards) != 0 {
		t.Fatalf("数据完整时不应选择分片: missing=%+v shards=%+v", missing, shards)
	}

	missing, shards = missingPetShards(index, map[uint32]bool{})
	if len(missing) != 3 || len(shards) != 2 {
		t.Fatalf("空库应选择所有有效分片: missing=%+v shards=%+v", missing, shards)
	}
}
