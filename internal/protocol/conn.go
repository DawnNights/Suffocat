package protocol

import (
	"net"
	"suffocat/internal/common"
	"suffocat/internal/protocol/command"
	"sync"
	"time"
)

type SeerConn interface {
	Type() string
	UserID() uint32
	LogFn(string, string)
	RegisterHandler(uint32, func(packet SeerPacket))
	SendFn(uint32, func(*common.ByteStream))
	Recv(uint32) *FuturePacket
	SendAndRecv(uint32, func(*common.ByteStream)) *FuturePacket
	Start(func(SeerConn))
	Exit() error
	UserState() *UserState
}

func NewFlashConn(account uint32, password string, logFn func(string, string)) SeerConn {
	// 连接Flash服务器
	var err error
	var conn net.Conn
	var session string
	success := false

	for range 2 {
		// 获取登录凭证
		session, err = get_session(account, password)
		if err != nil {
			logFn("ERROR", err.Error())
			return nil
		}

		addr, err := get_flash_server_addr(account, session)
		if err != nil {
			logFn("ERROR", "[Flash]服务器获取失败")
			return nil
		}

		conn, err = net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(2 * time.Second)
		} else {
			success = true
			break
		}
	}

	if success {
		logFn("DEBUG", "[Flash]服务器连接成功")
	} else {
		logFn("ERROR", "[Flash]服务器连接失败")
		return nil
	}

	// 构建FlashClient实例
	s := &seerFlashConn{
		conn:         conn,
		algo:         common.NewAlgorithms(),
		userID:       account,
		cmdHandlers:  map[uint32]func(SeerPacket){},
		recvHandlers: map[uint32]func(SeerPacket){},
		logFn:        logFn,
		writeMu:      sync.Mutex{},
	}

	// 发送登录包
	s.SendFn(command.LOGIN_IN, func(bs *common.ByteStream) {
		bs.WriteHex(session)
		bs.WriteString("unknown")
		bs.WriteBytes(make([]byte, 57))
		bs.WriteU32(1).WriteU32(1).WriteU32(1)
		bs.WriteString("flash_taomee")
		bs.WriteBytes(make([]byte, 20))
	})

	return s
}

func NewUnityConn(account uint32, password string, logFn func(string, string)) SeerConn {
	// 连接Unity服务器
	var err error
	var conn net.Conn
	var session string
	success := false

	for range 2 {
		// 获取登录凭证
		session, err = get_session(account, password)
		if err != nil {
			logFn("ERROR", err.Error())
			return nil
		}

		addr, err := get_unity_server_addr(account, session)
		if err != nil {
			logFn("ERROR", err.Error())
			return nil
		}

		conn, err = net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(2 * time.Second)
		} else {
			success = true
			break
		}
	}

	if success {
		logFn("DEBUG", "[Unity]服务器连接成功")
	} else {
		logFn("ERROR", "[Unity]服务器连接失败")
		return nil
	}

	// 构建UnityClient实例
	s := &seerUnityConn{
		conn:         conn,
		algo:         common.NewAlgorithms(),
		userID:       account,
		cmdHandlers:  map[uint32]func(SeerPacket){},
		recvHandlers: map[uint32]func(SeerPacket){},
		logFn:        logFn,
		writeMu:      sync.Mutex{},
	}

	// 发送登录包
	s.SendFn(command.LOGIN_IN, func(bs *common.ByteStream) {
		bs.WriteHex(session)
		bs.WriteString("taomee")
		bs.WriteBytes(make([]byte, 58))
		bs.WriteU32(1704).WriteU32(1).WriteString("PC")
		bs.WriteBytes(make([]byte, 16))
		bs.WriteU16(10000).WriteU32(1).WriteU32(1).WriteU32(2)
		bs.WriteString("unity_app_taomee")
		bs.WriteBytes(make([]byte, 16))
		bs.WriteString("com.taomee.protocol.mobile")
		bs.WriteBytes(make([]byte, 202))
		bs.WriteString("MS-7D99(Micro-StarInternationalCo.,Ltd.)")
		bs.WriteBytes(make([]byte, 92))
	})

	return s
}
