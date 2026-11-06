package common

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func Md5Hex(str string) string {
	result := md5.Sum([]byte(str))
	return hex.EncodeToString(result[:])
}

type Algorithms struct {
	key    []byte
	result int
}

func NewAlgorithms() *Algorithms {
	return &Algorithms{
		key:    []byte("!crAckmE4nOthIng:-)"),
		result: 0,
	}
}

func (a *Algorithms) UpdateResult(result int) {
	a.result = result
}

func (a *Algorithms) Encrypt(plain []byte) []byte {
	cipherLen := len(plain) + 1
	if len(plain) <= 4 {
		return nil
	}
	plain = plain[4:] // 跳过前4个字节

	cipher := make([]byte, len(plain)+1)
	j := 0
	needBecomeZero := false

	for i := 0; i < len(plain); i++ {
		if j == 1 && needBecomeZero {
			j = 0
			needBecomeZero = false
		}
		if j == len(a.key) {
			j = 0
			needBecomeZero = true
		}
		cipher[i] = plain[i] ^ a.key[j]
		j++
	}

	cipher[len(cipher)-1] = 0

	for i := len(cipher) - 1; i > 0; i-- {
		cipher[i] = byte(((uint16(cipher[i]) << 5) & 0xFF) | (uint16(cipher[i-1]) >> 3))
	}
	cipher[0] = byte(((uint16(cipher[0]) << 5) & 0xFF) | 3)

	result := int(a.key[len(plain)%len(a.key)]) * 13 % len(cipher)
	cipher = append(cipher[result:], cipher[:result]...)

	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(cipherLen))
	out = append(out, cipher...)
	return out
}

func (a *Algorithms) Decrypt(cipher []byte) []byte {
	plainLen := len(cipher) - 1
	if len(cipher) <= 4 {
		return nil
	}
	cipher = cipher[4:]

	result := int(a.key[(len(cipher)-1)%len(a.key)]) * 13 % len(cipher)
	cipher = append(cipher[len(cipher)-result:], cipher[:len(cipher)-result]...)

	plain := make([]byte, len(cipher)-1)
	for i := 0; i < len(cipher)-1; i++ {
		plain[i] = byte(((cipher[i] >> 5) & 0xFF) | ((cipher[i+1] << 3) & 0xFF))
	}

	j := 0
	needBecomeZero := false
	for i := 0; i < len(plain); i++ {
		if j == 1 && needBecomeZero {
			j = 0
			needBecomeZero = false
		}
		if j == len(a.key) {
			j = 0
			needBecomeZero = true
		}
		plain[i] ^= a.key[j]
		j++
	}

	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(plainLen))
	out = append(out, plain...)
	return out
}

func (a *Algorithms) InitKey(packetData []byte, userID int) {
	if len(packetData) < 4 {
		return
	}
	lastFour := packetData[len(packetData)-4:]
	lastUint := binary.BigEndian.Uint32(lastFour)
	xorResult := lastUint ^ uint32(userID)
	xorStr := fmt.Sprintf("%d", xorResult)

	hash := md5.Sum([]byte(xorStr))
	md5Str := hex.EncodeToString(hash[:])
	newKey := md5Str[:10]

	a.key = []byte(newKey)
	// fmt.Println("Updated encryption key to:", string(a.key))
}

func (a *Algorithms) MSerial(_a, b, c, d int) int {
	serialNumber := _a + c + (_a / (-3)) + (b % 17) + (d % 23) + 120
	return serialNumber

}

func (a *Algorithms) CalculateResult(cmdId uint32, body []byte) int {
	if cmdId <= 1000 {
		return 0 // 不更新内部状态，写 0
	}
	crc8Val := 0
	for _, b := range body {
		crc8Val ^= int(b)
	}
	newResult := a.MSerial(a.result, len(body), crc8Val, int(cmdId))
	a.result = newResult
	return newResult
}
