// Copyright 2017-2019 Andrew Goulas
// https://www.structinf.com
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package gomcc

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"strings"
)

const (
	CpeClickDistance = iota
	CpeCustomBlocks
	CpeHeldBlock
	CpeExtPlayerList
	CpeLongerMessages
	CpeSelectionCuboid
	CpeChangeModel
	CpeEnvWeatherType
	CpeHackControl
	CpeMessageTypes
	CpePlayerClick
	CpeBulkBlockUpdate
	CpeEnvMapAspect
	CpeEntityProperty
	CpeExtEntityPositions
	CpeTwoWayPing
	CpeInstantMOTD
	CpeFastMap

	CpeMax   = CpeFastMap
	CpeCount = CpeMax + 1
)

type ExtEntry struct {
	Name    string
	Version int
}

var Extensions = [CpeCount]ExtEntry{
	{"ClickDistance", 1},
	{"CustomBlocks", 1},
	{"HeldBlock", 1},
	{"ExtPlayerList", 2},
	{"LongerMessages", 1},
	{"SelectionCuboid", 1},
	{"ChangeModel", 1},
	{"EnvWeatherType", 1},
	{"HackControl", 1},
	{"MessageTypes", 1},
	{"PlayerClick", 1},
	{"BulkBlockUpdate", 1},
	{"EnvMapAspect", 1},
	{"EntityProperty", 1},
	{"ExtEntityPositions", 1},
	{"TwoWayPing", 1},
	{"InstantMOTD", 1},
	{"FastMap", 1},
}

const (
	packetTypeIdentification            = 0x00
	packetTypePing                      = 0x01
	packetTypeLevelInitialize           = 0x02
	packetTypeLevelDataChunk            = 0x03
	packetTypeLevelFinalize             = 0x04
	packetTypeSetBlockClient            = 0x05
	packetTypeSetBlock                  = 0x06
	packetTypeAddEntity                 = 0x07
	packetTypePlayerTeleport            = 0x08
	packetTypePositionOrientationUpdate = 0x09
	packetTypePositionUpdate            = 0x0a
	packetTypeOrientationUpdate         = 0x0b
	packetTypeRemoveEntity              = 0x0c
	packetTypeMessage                   = 0x0d
	packetTypeKick                      = 0x0e
	packetTypeSetPermission             = 0x0f

	packetTypeExtInfo                 = 0x10
	packetTypeExtEntry                = 0x11
	packetTypeSetClickDistance        = 0x12
	packetTypeCustomBlockSupportLevel = 0x13
	packetTypeHoldThis                = 0x14
	packetTypeExtAddPlayerName        = 0x16
	packetTypeExtRemovePlayerName     = 0x18
	packetTypeMakeSelection           = 0x1a
	packetTypeRemoveSelection         = 0x1b
	packetTypeChangeModel             = 0x1d
	packetTypeEnvSetWeatherType       = 0x1f
	packetTypeHackControl             = 0x20
	packetTypeExtAddEntity2           = 0x21
	packetTypePlayerClicked           = 0x22
	packetTypeBulkBlockUpdate         = 0x26
	packetTypeSetMapEnvUrl            = 0x28
	packetTypeSetMapEnvProperty       = 0x29
	packetTypeSetEntityProperty       = 0x2a
	packetTypeTwoWayPing              = 0x2b
)

func padString(str string) [64]byte {
	var result [64]byte
	copy(result[:], str)
	if len(str) < 64 {
		copy(result[len(str):], bytes.Repeat([]byte{' '}, 64-len(str)))
	}

	return result
}

func trimString(str [64]byte) string {
	return strings.TrimRight(string(str[:]), " ")
}

type Packet struct {
	buf bytes.Buffer
}

func (packet *Packet) position(location Location, extPos bool) {
	if extPos {
		binary.Write(&packet.buf, binary.BigEndian, &struct{ X, Y, Z int32 }{
			int32(location.X * 32),
			int32(location.Y * 32),
			int32(location.Z * 32),
		})
	} else {
		binary.Write(&packet.buf, binary.BigEndian, &struct{ X, Y, Z int16 }{
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
		})
	}
}

func (packet *Packet) motd(player *Player, motd string) {
	userType := byte(0x00)
	if player.operator {
		userType = 0x64
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID        byte
		ProtocolVersion byte
		Name            [64]byte
		MOTD            [64]byte
		UserType        byte
	}{
		packetTypeIdentification,
		0x07,
		padString(player.server.Config.Name),
		padString(motd),
		userType,
	})
}

func (packet *Packet) ping() {
	packet.buf.WriteByte(packetTypePing)
}

func (packet *Packet) levelInitialize() {
	packet.buf.WriteByte(packetTypeLevelInitialize)
}

func (packet *Packet) levelInitializeExt(size uint) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		Size     int32
	}{packetTypeLevelInitialize, int32(size)})
}

func (packet *Packet) levelDataChunk(blocks []byte, percent byte) {
	data := struct {
		PacketID        byte
		ChunkLength     int16
		ChunkData       [1024]byte
		PercentComplete byte
	}{
		packetTypeLevelDataChunk,
		int16(len(blocks)),
		[1024]byte{},
		percent,
	}

	copy(data.ChunkData[:], blocks)
	binary.Write(&packet.buf, binary.BigEndian, data)
}

func (packet *Packet) levelFinalize(x, y, z uint) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		X, Y, Z  int16
	}{packetTypeLevelFinalize, int16(x), int16(y), int16(z)})
}

func (packet *Packet) setBlock(x, y, z uint, block BlockID) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID  byte
		X, Y, Z   int16
		BlockType byte
	}{packetTypeSetBlock, int16(x), int16(y), int16(z), byte(block)})
}

func (packet *Packet) addEntity(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		PlayerID byte
		Name     [64]byte
	}{packetTypeAddEntity, id, padString(entity.DisplayName)})

	packet.position(location, extPos)
	binary.Write(&packet.buf, binary.BigEndian, &struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *Packet) teleport(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		PlayerID byte
	}{packetTypePlayerTeleport, id})

	packet.position(location, extPos)
	binary.Write(&packet.buf, binary.BigEndian, &struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *Packet) positionOrientationUpdate(entity *Entity) {
	location := entity.location
	lastLocation := entity.lastLocation
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID   byte
		PlayerID   byte
		X, Y, Z    byte
		Yaw, Pitch byte
	}{
		packetTypePositionOrientationUpdate,
		entity.id,
		byte((location.X - lastLocation.X) * 32),
		byte((location.Y - lastLocation.Y) * 32),
		byte((location.Z - lastLocation.Z) * 32),
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *Packet) positionUpdate(entity *Entity) {
	location := entity.location
	lastLocation := entity.lastLocation
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		PlayerID byte
		X, Y, Z  byte
	}{
		packetTypePositionUpdate,
		entity.id,
		byte((location.X - lastLocation.X) * 32),
		byte((location.Y - lastLocation.Y) * 32),
		byte((location.Z - lastLocation.Z) * 32),
	})
}

func (packet *Packet) orientationUpdate(entity *Entity) {
	location := entity.location
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID   byte
		PlayerID   byte
		Yaw, Pitch byte
	}{
		packetTypeOrientationUpdate,
		entity.id,
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *Packet) removeEntity(entity *Entity, self bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		PlayerID byte
	}{packetTypeRemoveEntity, id})
}

func (packet *Packet) message(msgType int, message string) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		PlayerID byte
		Message  [64]byte
	}{packetTypeMessage, byte(msgType), padString(message)})
}

func (packet *Packet) kick(reason string) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		Reason   [64]byte
	}{packetTypeKick, padString(reason)})
}

func (packet *Packet) userType(player *Player) {
	userType := byte(0x00)
	if player.operator {
		userType = 0x64
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		UserType byte
	}{packetTypeSetPermission, userType})
}

func (packet *Packet) extInfo() {
	binary.Write(&packet.buf, binary.BigEndian, struct {
		PacketID       byte
		AppName        [64]byte
		ExtensionCount int16
	}{packetTypeExtInfo, padString(ServerSoftware), int16(len(Extensions))})
}

func (packet *Packet) extEntry(entry *ExtEntry) {
	binary.Write(&packet.buf, binary.BigEndian, struct {
		PacketID byte
		ExtName  [64]byte
		Version  int32
	}{packetTypeExtEntry, padString(entry.Name), int32(entry.Version)})
}

func (packet *Packet) clickDistance(player *Player) {
	binary.Write(&packet.buf, binary.BigEndian, struct {
		PacketID byte
		Distance int16
	}{packetTypeSetClickDistance, int16(player.clickDistance * 32)})
}

func (packet *Packet) customBlockSupportLevel(level byte) {
	binary.Write(&packet.buf, binary.BigEndian, struct {
		PacketID     byte
		SupportLevel byte
	}{packetTypeCustomBlockSupportLevel, level})
}

func (packet *Packet) holdThis(block BlockID, lock bool) {
	preventChange := byte(0)
	if lock {
		preventChange = 1
	}

	binary.Write(&packet.buf, binary.BigEndian, struct {
		PacketID      byte
		BlockToHold   byte
		PreventChange byte
	}{packetTypeHoldThis, byte(block), preventChange})
}

func (packet *Packet) extAddPlayerName(entity *Entity, self bool) {
	id := int16(entity.id)
	if self {
		id = 0xff
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID   byte
		NameID     int16
		PlayerName [64]byte
		ListName   [64]byte
		GroupName  [64]byte
		GroupRank  byte
	}{
		packetTypeExtAddPlayerName,
		id,
		padString(entity.name),
		padString(entity.ListName),
		padString(entity.GroupName),
		entity.GroupRank,
	})
}

func (packet *Packet) extRemovePlayerName(entity *Entity, self bool) {
	id := int16(entity.id)
	if self {
		id = 0xff
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		NameID   int16
	}{packetTypeExtRemovePlayerName, id})
}

func (packet *Packet) makeSelection(id int, label string, box AABB, color color.RGBA) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID               byte
		SelectionID            byte
		Label                  [64]byte
		StartX, StartY, StartZ int16
		EndX, EndY, Endz       int16
		R, G, B, Opacity       int16
	}{
		packetTypeMakeSelection,
		byte(id),
		padString(label),
		int16(box.Min.X), int16(box.Min.Y), int16(box.Min.Z),
		int16(box.Max.X), int16(box.Max.Y), int16(box.Max.Z),
		int16(color.R), int16(color.G), int16(color.B), int16(color.A),
	})
}

func (packet *Packet) removeSelection(id int) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID    byte
		SelectionID byte
	}{packetTypeRemoveSelection, byte(id)})
}

func (packet *Packet) changeModel(entity *Entity, self bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID  byte
		EntityID  byte
		ModelName [64]byte
	}{packetTypeChangeModel, id, padString(entity.Model)})
}

func (packet *Packet) envWeatherType(level *Level) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID    byte
		WeatherType byte
	}{packetTypeEnvSetWeatherType, byte(level.Weather)})
}

func (packet *Packet) hackControl(config *HackConfig) {
	data := struct {
		PacketID        byte
		Flying          byte
		NoClip          byte
		Speeding        byte
		SpawnControl    byte
		ThirdPersonView byte
		JumpHeight      int16
	}{packetTypeHackControl, 0, 0, 0, 0, 0, int16(config.JumpHeight)}

	if config.Flying {
		data.Flying = 1
	}
	if config.NoClip {
		data.NoClip = 1
	}
	if config.Speeding {
		data.Speeding = 1
	}
	if config.SpawnControl {
		data.SpawnControl = 1
	}
	if config.ThirdPersonView {
		data.ThirdPersonView = 1
	}

	binary.Write(&packet.buf, binary.BigEndian, &data)
}

func (packet *Packet) extAddEntity2(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID    byte
		EntityID    byte
		DisplayName [64]byte
		SkinName    [64]byte
	}{
		packetTypeExtAddEntity2,
		id,
		padString(entity.DisplayName),
		padString(entity.SkinName),
	})

	packet.position(location, extPos)
	binary.Write(&packet.buf, binary.BigEndian, &struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *Packet) bulkBlockUpdate(indices []int32, blocks []byte) {
	data := struct {
		PacketID byte
		Count    byte
		Indices  [256]int32
		Blocks   [256]byte
	}{
		packetTypeBulkBlockUpdate,
		byte(len(indices)),
		[256]int32{},
		[256]byte{},
	}

	copy(data.Indices[:], indices)
	copy(data.Blocks[:], blocks)
	binary.Write(&packet.buf, binary.BigEndian, &data)
}

func (packet *Packet) mapEnvUrl(level *Level) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID       byte
		TexturePackURL [64]byte
	}{packetTypeSetMapEnvUrl, padString(level.TexturePack)})
}

func (packet *Packet) mapEnvProperty(id byte, value int32) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		Type     byte
		Value    int32
	}{packetTypeSetMapEnvProperty, id, value})
}

func (packet *Packet) entityProperty(entity *Entity, self bool, prop byte, value int32) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID byte
		EntityID byte
		Type     byte
		Value    int32
	}{packetTypeSetEntityProperty, id, prop, value})
}

func (packet *Packet) twoWayPing(dir byte, data int16) {
	binary.Write(&packet.buf, binary.BigEndian, &struct {
		PacketID  byte
		Direction byte
		Data      int16
	}{packetTypeTwoWayPing, dir, data})
}

type levelStream struct {
	player  *Player
	buf     [1024]byte
	index   int
	percent byte
}

func (stream *levelStream) send() {
	var packet Packet
	packet.levelDataChunk(stream.buf[:stream.index], stream.percent)
	stream.player.sendPacket(packet)
	stream.index = 0
}

func (stream *levelStream) Close() {
	if stream.index > 0 {
		stream.send()
	}
}

func (stream *levelStream) Write(p []byte) (int, error) {
	offset := 0
	count := len(p)
	for count > 0 {
		size := min(1024-stream.index, count)
		copy(stream.buf[stream.index:], p[offset:offset+size])

		stream.index += size
		offset += size
		count -= size

		if stream.index == 1024 {
			stream.send()
		}
	}

	return len(p), nil
}
