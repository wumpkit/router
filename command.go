package router

import (
	"context"

	"github.com/wumpkit-go/wumpkit"
)

// CommandHandler is the execution function for a command in the middleware chain.
type CommandHandler func(ctx Context) error

// Autocompleter is an optional interface a command can implement to handle autocomplete interactions.
type Autocompleter interface {
	Autocomplete(ctx context.Context, interaction *wumpkit.AutoCompleteInteraction) error
}

// AutocompleteHandler is the execution function for autocomplete interactions.
type AutocompleteHandler func(ctx context.Context, interaction *wumpkit.AutoCompleteInteraction) error

// ExcludedIn defines a bitmask of contexts where command execution is excluded.
type ExcludedIn uint32

const (
	// ExcludeGuild excludes execution in guild channels.
	ExcludeGuild ExcludedIn = 1 << iota
	// ExcludeDM excludes execution in direct messages.
	ExcludeDM
	// ExcludeGroupDM excludes execution in group direct messages.
	ExcludeGroupDM
	// ExcludeTextChannel excludes execution in guild text channels.
	ExcludeTextChannel
	// ExcludeVoiceChannel excludes execution in guild voice channels.
	ExcludeVoiceChannel
	// ExcludeStageChannel excludes execution in stage voice channels.
	ExcludeStageChannel
	// ExcludeThread excludes execution in threads.
	ExcludeThread
	// ExcludeForum excludes execution in forum channels.
	ExcludeForum
	// ExcludeAnnouncement excludes execution in guild announcement channels.
	ExcludeAnnouncement
)

// DisabledIn defines the execution surfaces where a command is disabled.
type DisabledIn uint8

const (
	// DisabledInSlash disables the command from slash command execution.
	DisabledInSlash DisabledIn = 1 << iota
	// DisabledInMessage disables the command from message/prefix command execution.
	DisabledInMessage
	// DisabledInBoth disables the command from both slash and message command execution.
	DisabledInBoth = DisabledInSlash | DisabledInMessage
)

// Command represents any supported Discord command type.
type Command any

// ChatCommand represents a hybrid chat (prefix and slash) command.
type ChatCommand interface {
	Config() ChatCommandConfig
	Run(ctx Context) error
}

// UserContextMenuCommand represents a user context menu command.
type UserContextMenuCommand interface {
	Config() UserContextMenuCommandConfig
	Run(ctx Context) error
}

// MessageContextMenuCommand represents a message context menu command.
type MessageContextMenuCommand interface {
	Config() MessageContextMenuCommandConfig
	Run(ctx Context) error
}

// CommonConfig holds metadata common to all command types (Chat, User Context Menu, Message Context Menu).
type CommonConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// ChatCommandConfig holds configuration metadata for a chat command.
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

// Config returns the configuration itself, satisfying the ChatCommand interface.
func (c ChatCommandConfig) Config() ChatCommandConfig {
	return c
}

// UserContextMenuCommandConfig holds configuration metadata for a user context menu command.
type UserContextMenuCommandConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// Config returns the configuration itself, satisfying the UserContextMenuCommand interface.
func (u UserContextMenuCommandConfig) Config() UserContextMenuCommandConfig {
	return u
}

// MessageContextMenuCommandConfig holds configuration metadata for a message context menu command.
type MessageContextMenuCommandConfig struct {
	Name            string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// Config returns the configuration itself, satisfying the MessageContextMenuCommand interface.
func (m MessageContextMenuCommandConfig) Config() MessageContextMenuCommandConfig {
	return m
}

// ButtonCommand represents an interactive button component command.
type ButtonCommand interface {
	Config() ButtonCommandConfig
	Run(ctx Context) error
}

// ButtonCommandConfig holds configuration metadata for a button command.
type ButtonCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// Config returns the configuration itself, satisfying the ButtonCommand interface.
func (b ButtonCommandConfig) Config() ButtonCommandConfig {
	return b
}

// SelectMenuCommand represents an interactive select menu component command.
type SelectMenuCommand interface {
	Config() SelectMenuCommandConfig
	Run(ctx Context) error
}

// SelectMenuCommandConfig holds configuration metadata for a select menu command.
type SelectMenuCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// Config returns the configuration itself, satisfying the SelectMenuCommand interface.
func (s SelectMenuCommandConfig) Config() SelectMenuCommandConfig {
	return s
}

// ModalCommand represents a modal submission command.
type ModalCommand interface {
	Config() ModalCommandConfig
	Run(ctx Context) error
}

// ModalCommandConfig holds configuration metadata for a modal submission command.
type ModalCommandConfig struct {
	CustomID        string
	GuildID         *wumpkit.Snowflake
	UserPermissions wumpkit.Permissions
	AppPermissions  wumpkit.Permissions
	Cooldowns       []CooldownRule
	ExcludedIn      ExcludedIn
}

// Config returns the configuration itself, satisfying the ModalCommand interface.
func (m ModalCommandConfig) Config() ModalCommandConfig {
	return m
}

// ExtractCommonConfig extracts shared metadata across all command types.
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

// chatcommandhandler handles chat command execution
type ChatCommandHandler func(ctx Context) error

// buttonhandler handles button component interactions
type ButtonHandler func(ctx *ButtonCommandContext) error

// selectmenuhandler handles select menu component interactions
type SelectMenuHandler func(ctx *SelectMenuCommandContext) error

// modalhandler handles modal submission interactions
type ModalHandler func(ctx *ModalCommandContext) error

// usercontextmenuhandler handles user context menu interactions
type UserContextMenuHandler func(ctx *UserContextMenuCommandContext) error

// messagecontextmenuhandler handles message context menu interactions
type MessageContextMenuHandler func(ctx *MessageContextMenuCommandContext) error

// funcChatCommand adapts a function and configuration to satisfy ChatCommand
type FuncChatCommand struct {
	ChatCommandConfig
	handler ChatCommandHandler
}

// run executes the adapted chat command handler
func (f *FuncChatCommand) Run(ctx Context) error {
	if f.handler != nil {
		return f.handler(ctx)
	}
	return nil
}

// newFuncChatCommand creates a new functional chat command
func NewFuncChatCommand(cfg ChatCommandConfig, handler ChatCommandHandler) *FuncChatCommand {
	return &FuncChatCommand{
		ChatCommandConfig: cfg,
		handler:           handler,
	}
}

// funcButtonCommand adapts a button handler function and configuration
type FuncButtonCommand struct {
	ButtonCommandConfig
	handler ButtonHandler
}

// run executes the adapted button handler
func (f *FuncButtonCommand) Run(ctx Context) error {
	if f.handler != nil {
		if bCtx, ok := AsButtonContext(ctx); ok {
			return f.handler(bCtx)
		}
		return f.handler(&ButtonCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     f,
			},
		})
	}
	return nil
}

// newFuncButtonCommand creates a new functional button command
func NewFuncButtonCommand(cfg ButtonCommandConfig, handler ButtonHandler) *FuncButtonCommand {
	return &FuncButtonCommand{
		ButtonCommandConfig: cfg,
		handler:             handler,
	}
}

// funcSelectMenuCommand adapts a select menu handler function and configuration
type FuncSelectMenuCommand struct {
	SelectMenuCommandConfig
	handler SelectMenuHandler
}

// run executes the adapted select menu handler
func (f *FuncSelectMenuCommand) Run(ctx Context) error {
	if f.handler != nil {
		if sCtx, ok := AsSelectMenuContext(ctx); ok {
			return f.handler(sCtx)
		}
		return f.handler(&SelectMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     f,
			},
		})
	}
	return nil
}

// newFuncSelectMenuCommand creates a new functional select menu command
func NewFuncSelectMenuCommand(cfg SelectMenuCommandConfig, handler SelectMenuHandler) *FuncSelectMenuCommand {
	return &FuncSelectMenuCommand{
		SelectMenuCommandConfig: cfg,
		handler:                 handler,
	}
}

// funcModalCommand adapts a modal handler function and configuration
type FuncModalCommand struct {
	ModalCommandConfig
	handler ModalHandler
}

// run executes the adapted modal handler
func (f *FuncModalCommand) Run(ctx Context) error {
	if f.handler != nil {
		if mCtx, ok := AsModalContext(ctx); ok {
			return f.handler(mCtx)
		}
		return f.handler(&ModalCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     f,
			},
		})
	}
	return nil
}

// newFuncModalCommand creates a new functional modal command
func NewFuncModalCommand(cfg ModalCommandConfig, handler ModalHandler) *FuncModalCommand {
	return &FuncModalCommand{
		ModalCommandConfig: cfg,
		handler:            handler,
	}
}

// funcUserContextMenuCommand adapts a user context menu handler
type FuncUserContextMenuCommand struct {
	UserContextMenuCommandConfig
	handler UserContextMenuHandler
}

// run executes the adapted user context menu handler
func (f *FuncUserContextMenuCommand) Run(ctx Context) error {
	if f.handler != nil {
		if uCtx, ok := ctx.(*UserContextMenuCommandContext); ok {
			return f.handler(uCtx)
		}
		return f.handler(&UserContextMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     f,
			},
		})
	}
	return nil
}

// newFuncUserContextMenuCommand creates a new functional user context menu command
func NewFuncUserContextMenuCommand(cfg UserContextMenuCommandConfig, handler UserContextMenuHandler) *FuncUserContextMenuCommand {
	return &FuncUserContextMenuCommand{
		UserContextMenuCommandConfig: cfg,
		handler:                      handler,
	}
}

// funcMessageContextMenuCommand adapts a message context menu handler
type FuncMessageContextMenuCommand struct {
	MessageContextMenuCommandConfig
	handler MessageContextMenuHandler
}

// run executes the adapted message context menu handler
func (f *FuncMessageContextMenuCommand) Run(ctx Context) error {
	if f.handler != nil {
		if mCtx, ok := ctx.(*MessageContextMenuCommandContext); ok {
			return f.handler(mCtx)
		}
		return f.handler(&MessageContextMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     f,
			},
		})
	}
	return nil
}

// newFuncMessageContextMenuCommand creates a new functional message context menu command
func NewFuncMessageContextMenuCommand(cfg MessageContextMenuCommandConfig, handler MessageContextMenuHandler) *FuncMessageContextMenuCommand {
	return &FuncMessageContextMenuCommand{
		MessageContextMenuCommandConfig: cfg,
		handler:                         handler,
	}
}
