package protocol

import "testing"

func TestUserStateInitialization(t *testing.T) {
	var state UserState
	if state.Known() {
		t.Fatal("零值状态不应被视为已初始化")
	}

	state.Init(11, []uint32{1, 2, 3})
	if !state.Known() {
		t.Fatal("初始化后状态应可用")
	}

	title, known := state.Title()
	if !known || title != 11 {
		t.Fatalf("称号缓存错误: title=%d known=%v", title, known)
	}

	match, known := state.ClothMatches([]uint32{3, 1, 2})
	if !known || !match {
		t.Fatalf("相同套装应按集合匹配: match=%v known=%v", match, known)
	}

	match, _ = state.ClothMatches([]uint32{1, 2})
	if match {
		t.Fatal("部件数量不同不应视为相同套装")
	}

	match, _ = state.ClothMatches([]uint32{1, 2, 4})
	if match {
		t.Fatal("部件不同不应视为相同套装")
	}
}

func TestUserStateUpdates(t *testing.T) {
	var state UserState
	state.Init(1, []uint32{1, 2})

	state.SetTitle(2)
	if title, _ := state.Title(); title != 2 {
		t.Fatalf("称号更新失败: %d", title)
	}

	state.SetClothes([]uint32{4, 5, 6})
	if match, _ := state.ClothMatches([]uint32{6, 4, 5}); !match {
		t.Fatal("套装更新后应匹配新部件")
	}
}
