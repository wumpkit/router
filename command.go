package router

import (
	"context"

	"github.com/wumpkit/wumpkit"
)

// commandhandler executes a command in the middleware chain
type CommandHandler func(ctx Context) error

// autocompleter handles autocomplete interactions
type Autocompleter interface {
	Autocomplete(ctx context.Context, interaction *wumpkit.AutoCompleteInteraction) error
}

// autocompletehandler executes autocomplete interactions
type AutocompleteHandler func(ctx context.Context, interaction *wumpkit.AutoCompleteInteraction) error

// excludedin defines bitmask of contexts where command execution is excluded
type ExcludedIn uint32

const (
	// excludeguild excludes execution in guild channels
	ExcludeGuild ExcludedIn = 1 << iota
	// excludedm excludes execution in direct messages
	ExcludeDM
	// excludegroupdm excludes execution in group direct messages
	ExcludeGroupDM
	// excludetextchannel excludes execution in guild text channels
	ExcludeTextChannel
	// excludevoicechannel excludes execution in guild voice channels
	ExcludeVoiceChannel
	// excludestagechannel excludes execution in stage voice channels
	ExcludeStageChannel
	// excludethread excludes execution in threads
	ExcludeThread
	// excludeforum excludes execution in forum channels
	ExcludeForum
	// excludeannouncement excludes execution in guild announcement channels
	ExcludeAnnouncement
)

// disabledin defines execution surfaces where a command is disabled
type DisabledIn uint8

const (
	// disabledinslash disables slash command execution
	DisabledInSlash DisabledIn = 1 << iota
	// disabledinmessage disables message or prefix command execution
	DisabledInMessage
	// disabledinboth disables slash and message command execution
	DisabledInBoth = DisabledInSlash | DisabledInMessage
)

// command represents any supported discord command type
type Command any

// chatcommand represents a hybrid prefix and slash command
type ChatCommand interface {
	Config() ChatCommandConfig
	Run(ctx Context) error
}

// usercontextmenucommand represents a user context menu command
type UserContextMenuCommand interface {
	Config() UserContextMenuCommandConfig
	Run(ctx *UserContextMenuCommandContext) error
}

// messagecontextmenucommand represents a message context menu command
type MessageContextMenuCommand interface {
	Config() MessageContextMenuCommandConfig
	Run(ctx *MessageContextMenuCommandContext) error
}

// commonconfig holds shared metadata across command types
type CommonConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// chatcommandconfig holds configuration metadata for a chat command
type ChatCommandConfig struct {
	Name            string
	Description     string
	Aliases         []string
	Args            []ArgDef
	Flags           []FlagDef
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	GuildID         *wumpkit.Snowflake
	NSFW            bool
	DisabledIn      DisabledIn
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
	Category        *Category
	Subcommands     []ChatCommand
}

// config returns the chat command configuration
func (c ChatCommandConfig) Config() ChatCommandConfig {
	return c
}

// usercontextmenucommandconfig holds configuration for a user context menu command
type UserContextMenuCommandConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// config returns the user context menu command configuration
func (u UserContextMenuCommandConfig) Config() UserContextMenuCommandConfig {
	return u
}

// messagecontextmenucommandconfig holds configuration for a message context menu command
type MessageContextMenuCommandConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// config returns the message context menu command configuration
func (m MessageContextMenuCommandConfig) Config() MessageContextMenuCommandConfig {
	return m
}

// buttoncommand represents an interactive button component command
type ButtonCommand interface {
	Config() ButtonCommandConfig
	Run(ctx *ButtonCommandContext) error
}

// buttoncommandconfig holds configuration for a button command
type ButtonCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// config returns the button command configuration
func (b ButtonCommandConfig) Config() ButtonCommandConfig {
	return b
}

// selectmenucommand represents an interactive select menu command
type SelectMenuCommand interface {
	Config() SelectMenuCommandConfig
	Run(ctx *SelectMenuCommandContext) error
}

// selectmenucommandconfig holds configuration for a select menu command
type SelectMenuCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// config returns the select menu command configuration
func (s SelectMenuCommandConfig) Config() SelectMenuCommandConfig {
	return s
}

// modalcommand represents a modal submission command
type ModalCommand interface {
	Config() ModalCommandConfig
	Run(ctx *ModalCommandContext) error
}

// modalcommandconfig holds configuration for a modal command
type ModalCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// config returns the modal command configuration
func (m ModalCommandConfig) Config() ModalCommandConfig {
	return m
}

// extractcommonconfig extracts shared metadata across all command types
func ExtractCommonConfig(cmd any) CommonConfig {
	switch c := cmd.(type) {
	case ChatCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.Name,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	case UserContextMenuCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.Name,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	case MessageContextMenuCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.Name,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	case ButtonCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.CustomID,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	case SelectMenuCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.CustomID,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	case ModalCommand:
		cfg := c.Config()
		return CommonConfig{
			Name:            cfg.CustomID,
			GuildID:         cfg.GuildID,
			UserPermissions: cfg.UserPermissions,
			AppPermissions:  cfg.AppPermissions,
			Cooldowns:       cfg.Cooldowns,
			ExcludedIn:      cfg.ExcludedIn,
		}
	default:
		return CommonConfig{}
	}
}
