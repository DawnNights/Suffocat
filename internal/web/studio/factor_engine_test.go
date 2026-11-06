package studio

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"suffocat/internal/common"
	"suffocat/internal/protocol"
)

type fakeFactorConn struct {
	port      string
	closed    chan struct{}
	closeOnce sync.Once
	exits     atomic.Int32
	state     protocol.UserState
}

func newFakeFactorConn(port string) *fakeFactorConn {
	return &fakeFactorConn{port: port, closed: make(chan struct{})}
}

func (f *fakeFactorConn) Type() string                   { return f.port }
func (f *fakeFactorConn) UserID() uint32                 { return 1 }
func (f *fakeFactorConn) LogFn(string, string)           {}
func (f *fakeFactorConn) UserState() *protocol.UserState { return &f.state }

func (f *fakeFactorConn) RegisterHandler(uint32, func(protocol.SeerPacket)) {}

func (f *fakeFactorConn) SendFn(uint32, func(*common.ByteStream)) {}

func (f *fakeFactorConn) Recv(uint32) *protocol.FuturePacket {
	return protocol.NewFuturePacket()
}

func (f *fakeFactorConn) SendAndRecv(uint32, func(*common.ByteStream)) *protocol.FuturePacket {
	return protocol.NewFuturePacket()
}

func (f *fakeFactorConn) Start(afterLogin func(protocol.SeerConn)) {
	go afterLogin(f)
	<-f.closed
}

func (f *fakeFactorConn) Exit() error {
	f.exits.Add(1)
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

func TestFactorSessionReusesConnection(t *testing.T) {
	session := newFactorSession(context.Background(), 10001, "password")
	defer session.Close()

	var mu sync.Mutex
	var created []*fakeFactorConn
	session.dial = func(port string, _ uint32, _ string, _ func(string, string)) protocol.SeerConn {
		conn := newFakeFactorConn(port)
		mu.Lock()
		created = append(created, conn)
		mu.Unlock()
		return conn
	}

	run := func(port string) {
		t.Helper()
		if err := session.Do(port, func(protocol.SeerConn) error { return nil }); err != nil {
			t.Fatalf("Do(%s) 失败: %v", port, err)
		}
	}

	run("UNITY")
	run("UNITY")
	run("UNITY")

	mu.Lock()
	if len(created) != 1 {
		count := len(created)
		mu.Unlock()
		t.Fatalf("同一端口应复用连接，实际创建了 %d 个", count)
	}
	first := created[0]
	mu.Unlock()

	run("FLASH")

	mu.Lock()
	if len(created) != 2 {
		count := len(created)
		mu.Unlock()
		t.Fatalf("切换端口后应创建新连接，实际创建了 %d 个", count)
	}
	second := created[1]
	mu.Unlock()

	if first.exits.Load() == 0 {
		t.Fatal("切换端口时应关闭旧连接")
	}

	session.Close()
	if second.exits.Load() == 0 {
		t.Fatal("任务结束时应关闭当前连接")
	}
}

func TestFactorSessionReconnectsAfterDisconnect(t *testing.T) {
	session := newFactorSession(context.Background(), 10001, "password")
	defer session.Close()

	var mu sync.Mutex
	var created []*fakeFactorConn
	session.dial = func(port string, _ uint32, _ string, _ func(string, string)) protocol.SeerConn {
		conn := newFakeFactorConn(port)
		mu.Lock()
		created = append(created, conn)
		mu.Unlock()
		return conn
	}

	if err := session.Do("UNITY", func(protocol.SeerConn) error { return nil }); err != nil {
		t.Fatalf("首次连接失败: %v", err)
	}

	session.mu.Lock()
	state := session.state
	session.mu.Unlock()
	if state == nil {
		t.Fatal("首次连接后会话状态为空")
	}

	// 模拟服务器或网络中断。
	state.conn.Exit()
	<-state.done

	if err := session.Do("UNITY", func(protocol.SeerConn) error { return nil }); err != nil {
		t.Fatalf("断线后重连失败: %v", err)
	}

	mu.Lock()
	count := len(created)
	mu.Unlock()
	if count != 2 {
		t.Fatalf("断线后应重新登录，实际创建了 %d 个连接", count)
	}
}
