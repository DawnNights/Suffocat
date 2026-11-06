package protocol

import (
	"encoding/binary"
	"fmt"
	"net"
	"suffocat/internal/common"
	"suffocat/internal/protocol/command"
	"sync"
	"sync/atomic"
)

type seerFlashConn struct {
	// 与游戏服务器的TCP连接
	conn net.Conn
	// 游戏加密解密算法处理结构
	algo *common.Algorithms
	// 用户米米号
	userID uint32
	// 封包序列号
	result int
	// 封包处理函数表
	cmdHandlers map[uint32]func(SeerPacket)
	// 响应包回调表
	recvHandlers map[uint32]func(SeerPacket)
	// 日志输出函数
	logFn func(level, message string)
	// 线程锁
	writeMu sync.Mutex
	// 是否由本地主动关闭
	closed atomic.Bool
	// 本次登录期间的角色状态缓存
	state UserState
}

func (s *seerFlashConn) UserState() *UserState {
	return &s.state
}

func (s *seerFlashConn) Type() string {
	return "FLASH"
}

func (s *seerFlashConn) UserID() uint32 {
	return s.userID
}

func (s *seerFlashConn) LogFn(level, message string) {
	s.logFn(level, message)
}

func (s *seerFlashConn) SendFn(cmd uint32, fn func(bs *common.ByteStream)) {
	bs := common.NewByteStream([]byte{})
	if fn != nil {
		fn(bs)
	}

	s.result = s.algo.CalculateResult(cmd, bs.Bytes())
	packet := SeerPacket{
		Length:   uint32(17 + bs.Length()),
		Version:  0x31,
		UserId:   s.userID,
		Sequence: s.result,
		Cmd:      cmd,
		Body:     bs,
	}

	s.writeMu.Lock()
	packet_delay(packet)
	s.conn.Write(s.algo.Encrypt(packet.UnMake()))
	s.writeMu.Unlock()
}

func (s *seerFlashConn) Recv(cmd uint32) *FuturePacket {
	fp := NewFuturePacket()
	s.recvHandlers[cmd] = func(packet SeerPacket) {
		fp.Set(packet)
	}

	return fp
}

func (s *seerFlashConn) SendAndRecv(cmd uint32, fn func(bs *common.ByteStream)) *FuturePacket {
	fp := s.Recv(cmd)
	s.SendFn(cmd, fn)
	return fp
}

func (s *seerFlashConn) RegisterHandler(cmd uint32, handler func(packet SeerPacket)) {
	if handler == nil {
		delete(s.cmdHandlers, cmd)
	} else {
		s.cmdHandlers[cmd] = handler
	}
}

func (s *seerFlashConn) Start(afterLogin func(SeerConn)) {
	// 注册登录事件处理
	is_login := false
	s.RegisterHandler(command.LOGIN_IN, func(packet SeerPacket) {
		if is_login {
			s.LogFn("ERROR", "您的账号已在其它端口登录")
		} else {
			is_login = true
			s.LogFn("DEBUG", "[Flash]服务器登录成功")
			s.algo.InitKey(packet.UnMake(), int(s.userID))
			s.algo.UpdateResult(packet.Sequence)
			s.SendFn(command.SYSTEM_TIME, nil)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.LogFn("ERROR", fmt.Sprintf("afterLogin发生严重异常(panic): %v", r))
					}
				}()
				afterLogin(s)
			}()
		}
	})

	// 注册心跳事件处理
	s.RegisterHandler(command.SYSTEM_TIME, func(packet SeerPacket) {
		s.SendFn(command.SYSTEM_TIME_CHECK, func(bs *common.ByteStream) {
			bs.WriteBytes(packet.Body.ReadAll())
		})
	})

	// 开始循环读取并处理封包
	buffer := []byte{}
	for {
		data := make([]byte, 2048)
		n, err := s.conn.Read(data)
		if err != nil {
			if s.closed.Load() {
				s.LogFn("DEBUG", "[Flash]连接已关闭")
			} else {
				s.LogFn("ERROR", "与Flash服务器断开连接")
			}
			return
		}

		buffer = append(buffer, data[:n]...)
		for len(buffer) > 4 {
			length := binary.BigEndian.Uint32(buffer[:4])

			if length > uint32(len(buffer)) {
				break
			}

			cipher := make([]byte, length)
			copy(cipher, buffer[:length])
			buffer = buffer[length:]

			bs := common.NewByteStream(s.algo.Decrypt(cipher))
			packet := SeerPacket{
				Length:   bs.ReadU32(),
				Version:  bs.ReadU8(),
				Cmd:      bs.ReadU32(),
				UserId:   bs.ReadU32(),
				Sequence: int(bs.ReadU32()),
				Body:     common.NewByteStream(bs.ReadAll()),
			}

			handler, ok := s.cmdHandlers[packet.Cmd]
			if ok {
				func() {
					defer func() {
						if r := recover(); r != nil {
							s.LogFn("ERROR", fmt.Sprintf("封包处理(cmd=%d)发生异常(panic): %v", packet.Cmd, r))
						}
					}()
					handler(packet)
				}()
			}

			handler, ok = s.recvHandlers[packet.Cmd]
			if ok {
				delete(s.recvHandlers, packet.Cmd)
				func() {
					defer func() {
						if r := recover(); r != nil {
							s.LogFn("ERROR", fmt.Sprintf("响应包处理(cmd=%d)发生异常(panic): %v", packet.Cmd, r))
						}
					}()
					handler(packet)
				}()
			}
		}
	}
}

func (s *seerFlashConn) Exit() error {
	s.closed.Store(true)
	return s.conn.Close()
}
