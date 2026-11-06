package protocol

import "sync"

// UserState 缓存本次登录期间的角色装扮与称号状态。
// 状态随连接创建，重新登录后会自动重新获取，避免使用过期数据。
type UserState struct {
	mu         sync.Mutex
	titleKnown bool
	title      uint32
	clothKnown bool
	clothes    []uint32
}

// Known 返回称号和装扮是否都已经获取。
func (s *UserState) Known() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.titleKnown && s.clothKnown
}

// Init 写入登录后首次获取到的角色状态。
func (s *UserState) Init(title uint32, clothes []uint32) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.title = title
	s.titleKnown = true
	s.clothes = append(s.clothes[:0], clothes...)
	s.clothKnown = true
}

func (s *UserState) Title() (uint32, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.title, s.titleKnown
}

func (s *UserState) SetTitle(title uint32) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.title = title
	s.titleKnown = true
}

// ClothMatches 判断当前装扮是否与给定的部件列表一致。
func (s *UserState) ClothMatches(clothes []uint32) (match, known bool) {
	if s == nil {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return sameIDs(s.clothes, clothes), s.clothKnown
}

func (s *UserState) SetClothes(clothes []uint32) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clothes = append(s.clothes[:0], clothes...)
	s.clothKnown = true
}

// sameIDs 按集合比较两个部件列表，忽略顺序。
func sameIDs(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[uint32]int, len(a))
	for _, id := range a {
		counts[id]++
	}
	for _, id := range b {
		if counts[id] == 0 {
			return false
		}
		counts[id]--
	}
	return true
}
