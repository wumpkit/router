package router

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wumpkit/wumpkit"
)

func buildSlashCommand(cmd ChatCommand) (wumpkit.ApplicationCommand, bool) {
	if cmd == nil {
		return nil, false
	}
	cfg := cmd.Config()
	if wumpkit.BitFieldHas(cfg.DisabledIn, DisabledInSlash) {
		return nil, false
	}

	name := strings.ToLower(strings.TrimSpace(cfg.Name))
	desc := cfg.Description
	if desc == "" {
		desc = "No description provided"
	}

	builder := wumpkit.NewSlashCommandBuilder(name, desc).
		SetNSFW(cfg.NSFW)

	if wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeDM) &&
		!wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeGuild) {
		builder.SetContexts(wumpkit.InteractionContextTypeGuild)
	} else if wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeGuild) &&
		!wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeDM) {
		builder.SetContexts(wumpkit.InteractionContextTypeBotDM)
	}

	if cfg.UserPermissions != 0 {
		builder.SetDefaultMemberPermissions(cfg.UserPermissions)
	}

	if len(cfg.Subcommands) > 0 {
		for _, sub := range cfg.Subcommands {
			subCfg := sub.Config()
			if wumpkit.BitFieldHas(subCfg.DisabledIn, DisabledInSlash) {
				continue
			}
			if len(subCfg.Subcommands) > 0 {
				builder.AddOption(buildSubcommandGroupOption(sub))
			} else {
				sc := buildSubcommandOption(sub)
				builder.AddOption(&sc)
			}
		}
	} else {
		for _, arg := range cfg.Args {
			builder.AddOption(buildCommandOption(arg))
		}
	}

	return builder.Build(), true
}

func buildUserCommand(cmd UserContextMenuCommand) wumpkit.ApplicationCommand {
	cfg := cmd.Config()
	builder := wumpkit.NewUserCommandBuilder(cfg.Name)
	if cfg.UserPermissions != 0 {
		builder.SetDefaultMemberPermissions(cfg.UserPermissions)
	}
	return builder.Build()
}

func buildMessageCommand(cmd MessageContextMenuCommand) wumpkit.ApplicationCommand {
	cfg := cmd.Config()
	builder := wumpkit.NewMessageCommandBuilder(cfg.Name)
	if cfg.UserPermissions != 0 {
		builder.SetDefaultMemberPermissions(cfg.UserPermissions)
	}
	return builder.Build()
}

func buildSubcommandGroupOption(
	group ChatCommand,
) *wumpkit.ApplicationCommandOptionSubCommandGroup {
	cfg := group.Config()
	desc := cfg.Description
	if desc == "" {
		desc = "Subcommand group"
	}
	var subcommands []wumpkit.ApplicationCommandOptionSubCommand
	for _, child := range cfg.Subcommands {
		if wumpkit.BitFieldHas(child.Config().DisabledIn, DisabledInSlash) {
			continue
		}
		subcommands = append(subcommands, buildSubcommandOption(child))
	}
	return &wumpkit.ApplicationCommandOptionSubCommandGroup{
		OptionBase: wumpkit.OptionBase{
			Type:        wumpkit.ApplicationCommandOptionTypeSubCommandGroup,
			Name:        strings.ToLower(cfg.Name),
			Description: desc,
		},
		Options: subcommands,
	}
}

func buildSubcommandOption(subcmd ChatCommand) wumpkit.ApplicationCommandOptionSubCommand {
	cfg := subcmd.Config()
	desc := cfg.Description
	if desc == "" {
		desc = "Subcommand"
	}
	var options []wumpkit.ApplicationCommandOption
	for _, arg := range cfg.Args {
		options = append(options, buildCommandOption(arg))
	}
	return wumpkit.ApplicationCommandOptionSubCommand{
		OptionBase: wumpkit.OptionBase{
			Type:        wumpkit.ApplicationCommandOptionTypeSubCommand,
			Name:        strings.ToLower(cfg.Name),
			Description: desc,
		},
		Options: options,
	}
}

func buildCommandOption(arg ArgDef) wumpkit.ApplicationCommandOption {
	argDesc := arg.Description
	if argDesc == "" {
		argDesc = "Argument"
	}
	argName := strings.ToLower(arg.Name)

	switch arg.Type {
	case ArgTypeInt:
		return buildIntegerOption(argName, argDesc, arg)
	case ArgTypeFloat:
		return buildFloatOption(argName, argDesc, arg)
	case ArgTypeBool:
		return &wumpkit.ApplicationCommandOptionBool{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeBool,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	case ArgTypeUser, ArgTypeMember:
		return &wumpkit.ApplicationCommandOptionUser{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeUser,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	case ArgTypeChannel:
		return &wumpkit.ApplicationCommandOptionChannel{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeChannel,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	case ArgTypeRole:
		return &wumpkit.ApplicationCommandOptionRole{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeRole,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	case ArgTypeMentionable:
		return &wumpkit.ApplicationCommandOptionMentionable{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeMentionable,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	case ArgTypeAttachment:
		return &wumpkit.ApplicationCommandOptionAttachment{
			OptionBase: wumpkit.OptionBase{
				Type:        wumpkit.ApplicationCommandOptionTypeAttachment,
				Name:        argName,
				Description: argDesc,
			},
			RequiredBase: wumpkit.RequiredBase{
				Required: arg.Required,
			},
		}
	default:
		return buildStringOption(argName, argDesc, arg)
	}
}

func buildIntegerOption(name, desc string, arg ArgDef) *wumpkit.ApplicationCommandOptionInteger {
	var minVal, maxVal *int
	if arg.MinValue != nil {
		v := int(*arg.MinValue)
		minVal = &v
	}
	if arg.MaxValue != nil {
		v := int(*arg.MaxValue)
		maxVal = &v
	}

	return &wumpkit.ApplicationCommandOptionInteger{
		OptionBase: wumpkit.OptionBase{
			Type:        wumpkit.ApplicationCommandOptionTypeInteger,
			Name:        name,
			Description: desc,
		},
		RequiredBase: wumpkit.RequiredBase{
			Required: arg.Required,
		},
		ChoiceBase: wumpkit.ChoiceBase{
			Autocomplete: arg.Autocomplete,
		},
		ChoiceFieldsInteger: wumpkit.ChoiceFieldsInteger{
			Choices: buildIntegerChoices(arg.Choices),
		},
		IntegerConstraints: wumpkit.IntegerConstraints{
			MinValue: minVal,
			MaxValue: maxVal,
		},
	}
}

func buildFloatOption(name, desc string, arg ArgDef) *wumpkit.ApplicationCommandOptionFloat {
	return &wumpkit.ApplicationCommandOptionFloat{
		OptionBase: wumpkit.OptionBase{
			Type:        wumpkit.ApplicationCommandOptionTypeFloat,
			Name:        name,
			Description: desc,
		},
		RequiredBase: wumpkit.RequiredBase{
			Required: arg.Required,
		},
		ChoiceBase: wumpkit.ChoiceBase{
			Autocomplete: arg.Autocomplete,
		},
		ChoiceFieldsFloat: wumpkit.ChoiceFieldsFloat{
			Choices: buildFloatChoices(arg.Choices),
		},
		FloatConstraints: wumpkit.FloatConstraints{
			MinValue: arg.MinValue,
			MaxValue: arg.MaxValue,
		},
	}
}

func buildStringOption(name, desc string, arg ArgDef) *wumpkit.ApplicationCommandOptionString {
	return &wumpkit.ApplicationCommandOptionString{
		OptionBase: wumpkit.OptionBase{
			Type:        wumpkit.ApplicationCommandOptionTypeString,
			Name:        name,
			Description: desc,
		},
		RequiredBase: wumpkit.RequiredBase{
			Required: arg.Required,
		},
		ChoiceBase: wumpkit.ChoiceBase{
			Autocomplete: arg.Autocomplete,
		},
		ChoiceFieldsString: wumpkit.ChoiceFieldsString{
			Choices: buildStringChoices(arg.Choices),
		},
		StringConstraints: wumpkit.StringConstraints{
			MinLength: arg.MinLength,
			MaxLength: arg.MaxLength,
		},
	}
}

func buildIntegerChoices(choices []Choice) []wumpkit.ApplicationCommandOptionChoiceInteger {
	var res []wumpkit.ApplicationCommandOptionChoiceInteger
	for _, c := range choices {
		var intVal int
		switch v := c.Value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
		default:
			if val, err := strconv.Atoi(fmt.Sprint(c.Value)); err == nil {
				intVal = val
			}
		}
		res = append(res, wumpkit.ApplicationCommandOptionChoiceInteger{
			ChoiceOptionBase: wumpkit.ChoiceOptionBase{
				Name: c.Name,
			},
			Value: intVal,
		})
	}
	return res
}

func buildFloatChoices(choices []Choice) []wumpkit.ApplicationCommandOptionChoiceFloat {
	var res []wumpkit.ApplicationCommandOptionChoiceFloat
	for _, c := range choices {
		var floatVal float64
		switch v := c.Value.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		default:
			if val, err := strconv.ParseFloat(fmt.Sprint(c.Value), 64); err == nil {
				floatVal = val
			}
		}
		res = append(res, wumpkit.ApplicationCommandOptionChoiceFloat{
			ChoiceOptionBase: wumpkit.ChoiceOptionBase{
				Name: c.Name,
			},
			Value: floatVal,
		})
	}
	return res
}

func buildStringChoices(choices []Choice) []wumpkit.ApplicationCommandOptionChoiceString {
	var res []wumpkit.ApplicationCommandOptionChoiceString
	for _, c := range choices {
		res = append(res, wumpkit.ApplicationCommandOptionChoiceString{
			ChoiceOptionBase: wumpkit.ChoiceOptionBase{
				Name: c.Name,
			},
			Value: fmt.Sprint(c.Value),
		})
	}
	return res
}
