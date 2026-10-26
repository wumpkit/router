package router

import (
	"github.com/wumpkit-go/wumpkit"
)

// commandOption configures a ChatCommandConfig
type CommandOption func(*ChatCommandConfig)

// withDescription sets the command description
func WithDescription(desc string) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Description = desc
	}
}

// withAliases sets command aliases
func WithAliases(aliases ...string) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Aliases = append(c.Aliases, aliases...)
	}
}

// withUserPermissions sets required user permissions
func WithUserPermissions(perms wumpkit.Permissions) CommandOption {
	return func(c *ChatCommandConfig) {
		c.UserPermissions = perms
	}
}

// withAppPermissions sets required bot permissions
func WithAppPermissions(perms wumpkit.Permissions) CommandOption {
	return func(c *ChatCommandConfig) {
		c.AppPermissions = perms
	}
}

// withGuildID sets a guild restriction
func WithGuildID(guildID wumpkit.Snowflake) CommandOption {
	return func(c *ChatCommandConfig) {
		c.GuildID = &guildID
	}
}

// withNSFW sets whether the command is age-restricted
func WithNSFW(nsfw bool) CommandOption {
	return func(c *ChatCommandConfig) {
		c.NSFW = nsfw
	}
}

// withDisabledIn sets execution disabled surfaces
func WithDisabledIn(disabled DisabledIn) CommandOption {
	return func(c *ChatCommandConfig) {
		c.DisabledIn = disabled
	}
}

// withExcludedIn sets excluded channel types
func WithExcludedIn(excluded ExcludedIn) CommandOption {
	return func(c *ChatCommandConfig) {
		c.ExcludedIn = excluded
	}
}

// withCooldown adds a cooldown rule
func WithCooldown(rule CooldownRule) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Cooldowns = append(c.Cooldowns, rule)
	}
}

// withCategory assigns a category
func WithCategory(cat *Category) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Category = cat
	}
}

// withArgs sets command arguments
func WithArgs(args ...ArgDef) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Args = append(c.Args, args...)
	}
}

// withFlags sets command flags
func WithFlags(flags ...FlagDef) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Flags = append(c.Flags, flags...)
	}
}

// withSubcommands sets subcommands
func WithSubcommands(subcommands ...ChatCommand) CommandOption {
	return func(c *ChatCommandConfig) {
		c.Subcommands = append(c.Subcommands, subcommands...)
	}
}

// buttonOption configures a ButtonCommandConfig
type ButtonOption func(*ButtonCommandConfig)

// withButtonGuildID restricts the button to a guild
func WithButtonGuildID(guildID wumpkit.Snowflake) ButtonOption {
	return func(c *ButtonCommandConfig) {
		c.GuildID = &guildID
	}
}

// withButtonUserPermissions sets required user permissions for the button
func WithButtonUserPermissions(perms wumpkit.Permissions) ButtonOption {
	return func(c *ButtonCommandConfig) {
		c.UserPermissions = perms
	}
}

// withButtonAppPermissions sets required bot permissions for the button
func WithButtonAppPermissions(perms wumpkit.Permissions) ButtonOption {
	return func(c *ButtonCommandConfig) {
		c.AppPermissions = perms
	}
}

// withButtonCooldown adds a cooldown rule to the button
func WithButtonCooldown(rule CooldownRule) ButtonOption {
	return func(c *ButtonCommandConfig) {
		c.Cooldowns = append(c.Cooldowns, rule)
	}
}

// withButtonExcludedIn sets excluded channel types for the button
func WithButtonExcludedIn(excluded ExcludedIn) ButtonOption {
	return func(c *ButtonCommandConfig) {
		c.ExcludedIn = excluded
	}
}

// selectMenuOption configures a SelectMenuCommandConfig
type SelectMenuOption func(*SelectMenuCommandConfig)

// withSelectMenuGuildID restricts select menu to a guild
func WithSelectMenuGuildID(guildID wumpkit.Snowflake) SelectMenuOption {
	return func(c *SelectMenuCommandConfig) {
		c.GuildID = &guildID
	}
}

// withSelectMenuUserPermissions sets required user permissions for select menu
func WithSelectMenuUserPermissions(perms wumpkit.Permissions) SelectMenuOption {
	return func(c *SelectMenuCommandConfig) {
		c.UserPermissions = perms
	}
}

// withSelectMenuAppPermissions sets required bot permissions for select menu
func WithSelectMenuAppPermissions(perms wumpkit.Permissions) SelectMenuOption {
	return func(c *SelectMenuCommandConfig) {
		c.AppPermissions = perms
	}
}

// withSelectMenuCooldown adds a cooldown rule to select menu
func WithSelectMenuCooldown(rule CooldownRule) SelectMenuOption {
	return func(c *SelectMenuCommandConfig) {
		c.Cooldowns = append(c.Cooldowns, rule)
	}
}

// withSelectMenuExcludedIn sets excluded channel types for select menu
func WithSelectMenuExcludedIn(excluded ExcludedIn) SelectMenuOption {
	return func(c *SelectMenuCommandConfig) {
		c.ExcludedIn = excluded
	}
}

// modalOption configures a ModalCommandConfig
type ModalOption func(*ModalCommandConfig)

// withModalGuildID restricts modal to a guild
func WithModalGuildID(guildID wumpkit.Snowflake) ModalOption {
	return func(c *ModalCommandConfig) {
		c.GuildID = &guildID
	}
}

// withModalUserPermissions sets required user permissions for modal
func WithModalUserPermissions(perms wumpkit.Permissions) ModalOption {
	return func(c *ModalCommandConfig) {
		c.UserPermissions = perms
	}
}

// withModalAppPermissions sets required bot permissions for modal
func WithModalAppPermissions(perms wumpkit.Permissions) ModalOption {
	return func(c *ModalCommandConfig) {
		c.AppPermissions = perms
	}
}

// withModalCooldown adds a cooldown rule to modal
func WithModalCooldown(rule CooldownRule) ModalOption {
	return func(c *ModalCommandConfig) {
		c.Cooldowns = append(c.Cooldowns, rule)
	}
}

// withModalExcludedIn sets excluded channel types for modal
func WithModalExcludedIn(excluded ExcludedIn) ModalOption {
	return func(c *ModalCommandConfig) {
		c.ExcludedIn = excluded
	}
}
