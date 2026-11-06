package protocol

import (
	"encoding/hex"
	"fmt"
	"suffocat/internal/common"
	"suffocat/internal/protocol/command"
	"time"
)

type SeerPacket struct {
	Length   uint32
	Version  uint8
	Cmd      uint32
	UserId   uint32
	Sequence int
	Body     *common.ByteStream
}

func (sp SeerPacket) UnMake() []byte {
	bs := common.NewByteStream([]byte{})
	bs.WriteU32(sp.Length)
	bs.WriteU8(sp.Version)
	bs.WriteU32(sp.Cmd)
	bs.WriteU32(sp.UserId)
	bs.WriteU32(uint32(sp.Sequence))
	bs.WriteBytes(sp.Body.Bytes())
	return bs.ReadAll()
}

func (sp SeerPacket) String() string {
	cmd, ok := command.MapCommand[sp.Cmd]
	if !ok || cmd == "" {
		cmd = fmt.Sprintf("%d", sp.Cmd)
	}

	hexBody := hex.EncodeToString(sp.Body.Bytes())
	if len(hexBody) > 48 {
		hexBody = hexBody[:48] + "..."
	}

	return fmt.Sprintf("{Length: %d, Version: 0x%X, Cmd: %s, UserId: %d, Result: %d, Body: %s}", sp.Length, sp.Version, cmd, sp.UserId, sp.Sequence, hexBody)
}

type FuturePacket struct {
	ch chan SeerPacket
}

func NewFuturePacket() *FuturePacket {
	return &FuturePacket{ch: make(chan SeerPacket, 1)}
}

func (f *FuturePacket) Wait() SeerPacket {
	select {
	case packet := <-f.ch:
		return packet
	case <-time.After(10 * time.Second):
		panic("Recv Packet timeout")
	}
}

func (f *FuturePacket) Set(p SeerPacket) {
	f.ch <- p
}
