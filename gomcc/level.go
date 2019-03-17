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

type LevelStorage interface {
	Load(path string) (*Level, error)
	Save(level *Level) error
}

type WeatherType byte

const (
	WeatherSunny   = 0
	WeatherRaining = 1
	WeatherSnowing = 2
)

type LevelAppearance struct {
	TexturePackURL        string
	SideBlock, EdgeBlock  BlockID
	SideLevel, CloudLevel uint
	MaxViewDistance       uint
}

type Level struct {
	Server *Server

	Name                  string
	Width, Height, Length uint
	Blocks                []BlockID
	Spawn                 Location
	Appearance            LevelAppearance
	Weather               WeatherType
}

func NewLevel(name string, width, height, length uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		nil,
		name,
		width, height, length,
		make([]BlockID, width*height*length),
		Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
		LevelAppearance{
			SideBlock:       BlockBedrock,
			EdgeBlock:       BlockActiveWater,
			SideLevel:       height / 2,
			CloudLevel:      height + 2,
			MaxViewDistance: 0,
		},
		WeatherSunny,
	}
}

func (level *Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	blocks := make([]BlockID, len(level.Blocks))
	copy(blocks, level.Blocks)

	return &Level{
		nil,
		name,
		level.Width, level.Height, level.Length,
		blocks,
		level.Spawn,
		level.Appearance,
		level.Weather,
	}
}

func (level *Level) Volume() uint {
	return level.Width * level.Height * level.Length
}

func (level *Level) Index(x, y, z uint) uint {
	return x + level.Width*(z+level.Length*y)
}

func (level *Level) GetBlock(x, y, z uint) BlockID {
	if x < level.Width && y < level.Height && z < level.Length {
		return level.Blocks[level.Index(x, y, z)]
	}

	return BlockAir
}

func (level *Level) ForEachEntity(fn func(*Entity)) {
	if level.Server == nil {
		return
	}

	level.Server.ForEachEntity(func(entity *Entity) {
		if entity.level == level {
			fn(entity)
		}
	})
}

func (level *Level) ForEachClient(fn func(*Client)) {
	if level.Server == nil {
		return
	}

	level.Server.ForEachClient(func(client *Client) {
		if client.Entity.level == level {
			fn(client)
		}
	})
}

func (level *Level) SetBlock(x, y, z uint, block BlockID, broadcast bool) {
	if x < level.Width && y < level.Height && z < level.Length {
		level.Blocks[level.Index(x, y, z)] = block
		if broadcast {
			level.ForEachClient(func(client *Client) {
				client.sendBlockChange(x, y, z, block)
			})
		}
	}
}

func (level *Level) SetWeather(weather WeatherType) {
	if weather != level.Weather {
		level.ForEachClient(func(client *Client) {
			client.sendWeather(weather)
		})

		level.Weather = weather
	}
}
