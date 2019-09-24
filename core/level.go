// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *Plugin) handleCopyLvl(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 2 {
		sender.SendMessage("Usage: " + command.Name + " <src> <dest>")
		return
	}

	src := sender.Server().FindLevel(args[0])
	if src == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	dest := sender.Server().FindLevel(args[1])
	if dest != nil {
		sender.SendMessage("Level " + args[1] + " already exists")
		return
	}

	dest = src.Clone(args[1])
	sender.Server().AddLevel(dest)
	sender.SendMessage("Level " + args[0] + " has been copied to " + args[1])
}

func (plugin *Plugin) handleGoto(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	if level == player.Level() {
		sender.SendMessage("You are already in " + level.Name)
		return
	}

	player.TeleportLevel(level)
}

func (plugin *Plugin) handleLoad(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	_, err := sender.Server().LoadLevel(args[0])
	if err != nil {
		sender.SendMessage("Could not load level " + args[0])
		return
	}

	sender.SendMessage("Level " + args[0] + " loaded")
}

func (plugin *Plugin) handleMain(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 0:
		sender.SendMessage("Main level is " + sender.Server().MainLevel.Name)

	case 1:
		level := sender.Server().FindLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		sender.Server().MainLevel = level
		sender.SendMessage("Set main level to " + level.Name)

	default:
		sender.SendMessage("Usage: " + command.Name + " <level>")
	}
}

func (plugin *Plugin) handleNewLvl(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) < 5 {
		sender.SendMessage("Usage: " + command.Name + " <name> <width> <height> <length> <theme> <args>")
		return
	}

	width, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		sender.SendMessage(args[1] + " is not a valid number")
		return
	}

	height, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		sender.SendMessage(args[2] + " is not a valid number")
		return
	}

	length, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		sender.SendMessage(args[3] + " is not a valid number")
		return
	}

	genFunc, ok := gomcc.Generators[args[4]]
	if !ok {
		sender.SendMessage("Generator " + args[4] + " not found")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level != nil {
		sender.SendMessage("Level " + args[0] + " already exists")
		return
	}

	level = gomcc.NewLevel(args[0], int(width), int(height), int(length))
	if level == nil {
		sender.SendMessage("Could not create level")
		return
	}

	generator := genFunc(args[5:]...)
	generator.Generate(level)

	sender.Server().AddLevel(level)
	sender.SendMessage("Level " + level.Name + " created")
}

func (plugin *Plugin) handlePhysics(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	var level *level
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		if player, ok := sender.(*gomcc.Player); !ok {
			sender.SendMessage("You are not a player")
			return
		} else {
			level = plugin.findLevel(player.Level().Name)
		}

	case 2:
		level = plugin.findLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		args = args[1:]

	default:
		sender.SendMessage("Usage: " + command.Name + " <level> <value>")
		return
	}

	if value, err := strconv.ParseBool(args[0]); err != nil {
		sender.SendMessage(args[0] + " is not a valid boolean")
		return
	} else {
		var oldValue bool
		plugin.db.Get(&oldValue, "SELECT physics FROM levels WHERE name = ?", level.Name)
		plugin.db.MustExec("UPDATE levels SET physics = ? WHERE name = ?", value, level.Name)
		if oldValue != value {
			if value {
				plugin.enablePhysics(level)
			} else {
				plugin.disablePhysics(level)
			}
		}

		sender.SendMessage(fmt.Sprintf("Physics set to %t", value))
	}
}

func (plugin *Plugin) handleSave(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	if args[0] == "all" {
		sender.Server().ForEachLevel(func(level *gomcc.Level) {
			sender.Server().SaveLevel(level)
		})
		sender.SendMessage("All levels have been saved")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	sender.Server().SaveLevel(level)
	sender.SendMessage("Level " + level.Name + " saved")
}

func (plugin *Plugin) handleSetSpawn(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 0:
		level := player.Level()
		level.Spawn = player.Location()
		level.Dirty = true

		player.SetSpawn()
		sender.SendMessage("Spawn location set to your current location")

	case 1:
		target := sender.Server().FindPlayer(args[0])
		if target == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		if target.Level() != player.Level() {
			sender.SendMessage(target.Name() + " is on a different level")
			return
		}

		target.Teleport(player.Location())
		target.SetSpawn()
		sender.SendMessage("Spawn location of " + player.Name() + " set to your current location")

	default:
		sender.SendMessage("Usage: " + command.Name + " <player>")
	}
}

func (plugin *Plugin) handleSpawn(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) > 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	player.Teleport(player.Level().Spawn)
}

func (plugin *Plugin) handleUnload(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	if level == sender.Server().MainLevel {
		sender.SendMessage("Level " + args[0] + " is the main level")
		return
	}

	sender.Server().UnloadLevel(level)
	sender.SendMessage("Level " + args[0] + " unloaded")
}
