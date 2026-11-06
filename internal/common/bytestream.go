package common

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type ByteStream struct {
	data []byte
}

// 创建新的字节流
func NewByteStream(data []byte) *ByteStream {
	return &ByteStream{data: data}
}

// 写入 uint8
func (bs *ByteStream) WriteU8(n uint8) *ByteStream {
	bs.data = append(bs.data, n)
	return bs
}

// 写入 uint16（大端）
func (bs *ByteStream) WriteU16(n uint16) *ByteStream {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], n)
	bs.data = append(bs.data, buf[:]...)
	return bs
}

// 写入 uint32（大端）
func (bs *ByteStream) WriteU32(n uint32) *ByteStream {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], n)
	bs.data = append(bs.data, buf[:]...)
	return bs
}

func (bs *ByteStream) WriteU32List(list ...uint32) *ByteStream {
	for _, n := range list {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], n)
		bs.data = append(bs.data, buf[:]...)
	}
	return bs
}

// 写入字节数组
func (bs *ByteStream) WriteBytes(b []byte) *ByteStream {
	bs.data = append(bs.data, b...)
	return bs
}

// 写入字符串
func (bs *ByteStream) WriteString(s string) *ByteStream {
	bs.data = append(bs.data, []byte(s)...)
	return bs
}

// 写入十六进制字符串
func (bs *ByteStream) WriteHex(str string) *ByteStream {
	decoded, err := hex.DecodeString(str)
	if err != nil {
		bs.data = append(bs.data, []byte(str)...)
	} else {
		bs.data = append(bs.data, decoded...)
	}
	return bs
}

// ---- 读取操作（无 readIndex，直接切片前移） ----

// 读取 uint8
func (bs *ByteStream) ReadU8() uint8 {
	if len(bs.data) < 1 {
		panic("ByteStream: read out of range (U8)")
	}
	val := bs.data[0]
	bs.data = bs.data[1:]
	return val
}

func (bs *ByteStream) ReadU8List(size uint32) []uint8 {
	var list []uint8
	for range size {
		list = append(list, bs.ReadU8())
	}
	return list
}

// 读取 uint16（大端）
func (bs *ByteStream) ReadU16() uint16 {
	if len(bs.data) < 2 {
		panic("ByteStream: read out of range (U16)")
	}
	val := binary.BigEndian.Uint16(bs.data[:2])
	bs.data = bs.data[2:]
	return val
}

// 读取 uint32（大端）
func (bs *ByteStream) ReadU32() uint32 {
	if len(bs.data) < 4 {
		panic("ByteStream: read out of range (U32)")
	}
	val := binary.BigEndian.Uint32(bs.data[:4])
	bs.data = bs.data[4:]
	return val
}

// 查看（不移动游标）uint32
func (bs *ByteStream) ShowU32() uint32 {
	if len(bs.data) < 4 {
		panic("ByteStream: read out of range (ShowU32)")
	}
	return binary.BigEndian.Uint32(bs.data[:4])
}

// 读取指定长度的字节
func (bs *ByteStream) ReadBytes(length int) []byte {
	if len(bs.data) < length {
		panic(fmt.Sprintf("ByteStream: read out of range (need %d, have %d)", length, len(bs.data)))
	}
	val := bs.data[:length]
	bs.data = bs.data[length:]
	return val
}

// 读取指定长度的字节为十六进制字符串
func (bs *ByteStream) ReadHex(length int) string {
	if len(bs.data) < length {
		panic(fmt.Sprintf("ByteStream: read out of range (need %d, have %d)", length, len(bs.data)))
	}
	val := hex.EncodeToString(bs.data[:length])
	bs.data = bs.data[length:]
	return val
}

// 读取剩余全部字节
func (bs *ByteStream) ReadAll() []byte {
	val := bs.data
	bs.data = nil
	return val
}

// ---- 工具方法 ----

// Bytes 返回当前所有数据的副本
func (bs *ByteStream) Bytes() []byte {
	cp := make([]byte, len(bs.data))
	copy(cp, bs.data)
	return cp
}

// Length 返回当前数据长度
func (bs *ByteStream) Length() int {
	return len(bs.data)
}

// IsEmpty 返回当前数据是否为空
func (bs *ByteStream) IsEmpty() bool {
	return len(bs.data) == 0
}

// Clear 清空数据
func (bs *ByteStream) Clear() {
	bs.data = nil
}

// Hex 返回十六进制字符串
func (bs *ByteStream) Hex() string {
	return hex.EncodeToString(bs.data)
}

// Skip 跳过指定长度
func (bs *ByteStream) Skip(length int) *ByteStream {
	if len(bs.data) < length {
		panic(fmt.Sprintf("ByteStream: skip out of range (need %d, have %d)", length, len(bs.data)))
	}
	bs.data = bs.data[length:]
	return bs
}
