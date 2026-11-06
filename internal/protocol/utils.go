package protocol

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"suffocat/internal/common"
	"suffocat/internal/protocol/command"
	"time"
)

func get_current_timestamp() int64 {
	return time.Now().UnixMilli()
}

func jquery_mock_callback() string {
	return fmt.Sprintf("jQuery1.7.20%16d_%d", rand.Int63(), get_current_timestamp()-1000)
}

func get_session(account uint32, password string) (string, error) {
	token := md5.Sum([]byte(password))

	text, err := common.HttpGets("https://account-co.61.com/index.php?" + url.Values{
		"r":           {"userIdentity/authenticate"},
		"callback":    {jquery_mock_callback()},
		"account":     {fmt.Sprintf("%d", account)},
		"rememberAcc": {"false"},
		"passwd":      {hex.EncodeToString(token[:])},
		"rememberPwd": {"true"},
		"vericode":    {""},
		"game":        {"02"},
		"tad":         {"none"},
		"_":           {fmt.Sprintf("%d/authenticate/login", get_current_timestamp())},
		"gid":         {"206"},
	}.Encode())

	if err != nil {
		return "", errors.New("网络连接异常")
	}

	session := common.StringBetween(text, `"session":"`, `"}`)
	if session == "" {
		return "", errors.New("账号或密码错误")
	}

	return session, nil
}

func parse_server_list(data []byte) []string {
	var servers []string
	i := 0
	for i < len(data) {
		if data[i] >= '0' && data[i] <= '9' {
			start := i
			for data[i] == '.' || (data[i] >= '0' && data[i] <= '9') {
				i++
			}

			ip := string(data[start:i])
			i += 14 - len(ip)

			port := binary.BigEndian.Uint32(data[i : i+4])
			i = i + 4

			if strings.Count(ip, ".") == 3 && port > 0 && port < 65536 {
				servers = append(servers, fmt.Sprintf("%s:%d", ip, port))
			}
		} else {
			i++
		}
	}
	return servers
}

func get_flash_server_addr(account uint32, session string) (string, error) {
	buf := common.NewByteStream([]byte{})
	buf.WriteHex("000000253100000069")
	buf.WriteU32(account)
	buf.WriteU32(0)
	buf.WriteHex(session)
	buf.WriteHex("00000000")

	addrs, err := common.HttpGets("https://seerlogin.61.com/ip.txt")
	if err != nil {
		return "", err
	}

	conn, err := net.Dial("tcp", strings.Split(addrs, "|")[0])
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write(buf.ReadAll())
	if err != nil {
		return "", err
	}

	data := make([]byte, 2048)
	n, err := conn.Read(data)
	if err != nil {
		return "", err
	}

	server_list := parse_server_list(data[17:n])
	if len(server_list) == 0 {
		fmt.Println(hex.EncodeToString(data[17:n]))
		return "", errors.New("[Flash]服务器解析失败")
	}

	// return server_list[rand.Intn(len(server_list))], nil
	return server_list[0], nil
}

func get_unity_server_addr(account uint32, session string) (string, error) {
	buf := common.NewByteStream([]byte{})
	buf.WriteHex("000000253100000069")
	buf.WriteU32(account)
	buf.WriteU32(0)
	buf.WriteHex(session)
	buf.WriteHex("00000000")
	addrs, err := common.HttpGets("https://seer-login-ip.61.com/unity-ip.txt")
	if err != nil {
		return "", err
	}

	conn, err := net.Dial("tcp", strings.Split(addrs, "|")[0])
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write(buf.ReadAll())
	if err != nil {
		return "", err
	}

	data := make([]byte, 2048)
	n, err := conn.Read(data)
	if err != nil {
		return "", err
	}

	server_list := parse_server_list(data[17:n])
	if len(server_list) == 0 {
		fmt.Println(hex.EncodeToString(data[17:n]))
		return "", errors.New("[Unity]服务器解析失败")
	}
	return server_list[rand.Intn(len(server_list))], nil
}

func packet_delay(packet SeerPacket) {
	switch packet.Cmd {
	case command.FIGHT_NPC_MONSTER,
		command.CHALLENGE_BOSS,
		command.MIBAO_FIGHT,
		command.VANISH_AIR_BRICK_CHALLENGE,
		command.FIGHT_TRY_USE_PET,
		command.FIGHT_PET_SKIN_TRIAL:
		time.Sleep(3 * time.Second)
	case command.YINZI_START_FIGHT:
		time.Sleep(9 * time.Second)
	case command.FIGHT_H5_PVE_BOSS:
		switch packet.Body.ShowU32() {
		case 115, 116:
			time.Sleep(8 * time.Second)
		default:
			time.Sleep(6 * time.Second)
		}
	default:
		time.Sleep(200 * time.Millisecond)
	}
}
