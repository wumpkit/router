package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wumpkit-go/wumpkit"
)

// DefaultPrefix is the fallback command prefix string.
const DefaultPrefix = "!"

// PrefixFunc dynamically determines the command prefix for a message.
type PrefixFunc func(msg *wumpkit.Message) string

// DynamicPrefixFunc dynamically determines the prefix with context
type DynamicPrefixFunc func(ctx context.Context, msg *wumpkit.Message) (string, error)

// HookFunc runs before command execution
type HookFunc func(ctx Context) error

// AfterHookFunc runs after command execution
type AfterHookFunc func(ctx Context, err error)

// PermissionResolver dynamically determines user and bot permissions for a command context.
type PermissionResolver func(ctx Context) (userPerms wumpkit.Permissions, appPerms wumpkit.Permissions, err error)

// Option configures a Router instance.
type Option func(*Router)

// WithDynamicPrefixFunc sets a context-aware prefix function
func WithDynamicPrefixFunc(fn DynamicPrefixFunc) Option {
	return func(r *Router) {
		r.dynamicPrefixFunc = fn
	}
}

// WithClient sets the wumpkit client for the router.
func WithClient(client *wumpkit.Client) Option {
	return func(r *Router) {
		r.client = client
	}
}

// WithCommandErrorHandler sets a custom error handler for commands.
func WithCommandErrorHandler(handler CommandErrorHandler) Option {
	return func(r *Router) {
		if handler != nil {
			r.errorHandler = handler
		}
	}
}

// WithPrefix sets a static prefix string (e.g. "!").
func WithPrefix(prefix string) Option {
	return func(r *Router) {
		r.prefixFunc = func(msg *wumpkit.Message) string {
			return prefix
		}
	}
}

// WithPrefixFunc sets a dynamic prefix function (e.g. for guild custom prefixes).
func WithPrefixFunc(fn PrefixFunc) Option {
	return func(r *Router) {
		r.prefixFunc = fn
	}
}

// WithLogger sets the structured logger for router operations.
func WithLogger(logger wumpkit.Logger) Option {
	return func(r *Router) {
		r.logger = logger
	}
}

// WithPermissionResolver sets a custom resolver for guild permissions (useful for prefix commands or custom auth).
func WithPermissionResolver(resolver PermissionResolver) Option {
	return func(r *Router) {
		r.permissionResolver = resolver
	}
}

// Router is the central command, interaction, and component routing engine.
// It embeds *Registry so all command registration and lookup methods are directly available on Router.
type Router struct {
	*Registry

	client             *wumpkit.Client
	logger             wumpkit.Logger
	errorHandler       CommandErrorHandler
	prefixFunc         PrefixFunc
	dynamicPrefixFunc  DynamicPrefixFunc
	permissionResolver PermissionResolver
	middlewares        []Middleware
	beforeHooks        []HookFunc
	afterHooks         []AfterHookFunc
}

// New creates a new Router instance with default error handling and an initialized Registry.
func New(options ...Option) *Router {
	r := &Router{
		Registry:     NewRegistry(),
		errorHandler: NewDefaultCommandErrorHandler(),
		prefixFunc: func(msg *wumpkit.Message) string {
			return DefaultPrefix
		},
	}

	for _, opt := range options {
		opt(r)
	}

	if r.errorHandler == nil {
		r.errorHandler = NewDefaultCommandErrorHandler()
	}

	return r
}

// NewWithClient creates a new Router initialized with a wumpkit client
func NewWithClient(client *wumpkit.Client, options ...Option) *Router {
	opts := append([]Option{WithClient(client)}, options...)
	return New(opts...)
}

// Client returns the underlying wumpkit client.
func (r *Router) Client() *wumpkit.Client {
	return r.client
}

// Logger returns the configured structured logger.
func (r *Router) Logger() wumpkit.Logger {
	return r.logger
}

// CommandErrorHandler returns the configured command error handler interface.
func (r *Router) CommandErrorHandler() CommandErrorHandler {
	return r.errorHandler
}

// Use appends middlewares to the execution pipeline.
func (r *Router) Use(middlewares ...Middleware) *Router {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// MustRegisterCommand registers a chat command and panics if registration fails.
func (r *Router) MustRegisterCommand(cmd ChatCommand) *Router {
	if err := r.RegisterCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterUserContextMenuCommand registers a user context menu command and panics if registration fails.
func (r *Router) MustRegisterUserContextMenuCommand(cmd UserContextMenuCommand) *Router {
	if err := r.RegisterUserContextMenuCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterMessageContextMenuCommand registers a message context menu command and panics if registration fails.
func (r *Router) MustRegisterMessageContextMenuCommand(cmd MessageContextMenuCommand) *Router {
	if err := r.RegisterMessageContextMenuCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterButtonCommand registers a button command and panics if registration fails.
func (r *Router) MustRegisterButtonCommand(cmd ButtonCommand) *Router {
	if err := r.RegisterButtonCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterSelectMenuCommand registers a select menu command and panics if registration fails.
func (r *Router) MustRegisterSelectMenuCommand(cmd SelectMenuCommand) *Router {
	if err := r.RegisterSelectMenuCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterModalCommand registers a modal command and panics if registration fails.
func (r *Router) MustRegisterModalCommand(cmd ModalCommand) *Router {
	if err := r.RegisterModalCommand(cmd); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterAlias associates an alias with a registered command and panics on error.
func (r *Router) MustRegisterAlias(alias string, target string) *Router {
	if err := r.RegisterAlias(alias, target); err != nil {
		panic(err)
	}
	return r
}

// MustRegisterCategory registers a category and panics if registration fails.
func (r *Router) MustRegisterCategory(cat *Category) *Router {
	if err := r.RegisterCategory(cat); err != nil {
		panic(err)
	}
	return r
}

// AutoRegister attaches event handlers to the wumpkit Client.
func (r *Router) AutoRegister() {
	if r.client == nil {
		return
	}
	r.client.OnMessageCreate(func(ctx context.Context, e wumpkit.MessageCreateEvent) {
		r.HandleMessage(ctx, e)
	})

	r.client.OnInteractionCreate(func(ctx context.Context, e wumpkit.InteractionCreateEvent) {
		r.HandleInteraction(ctx, e)
	})
}

// attach attaches router listeners to the client
func (r *Router) Attach(client *wumpkit.Client) *Router {
	if client != nil {
		r.client = client
	}
	r.AutoRegister()
	return r
}

// autoSync automatically synchronizes application commands on ready
func (r *Router) AutoSync(appID wumpkit.Snowflake, devGuildIDs ...wumpkit.Snowflake) *Router {
	if r.client == nil {
		return r
	}
	r.client.OnReady(func(ctx context.Context, _ wumpkit.ReadyEvent) {
		if err := r.SyncApplicationCommands(ctx, appID, devGuildIDs...); err != nil && r.logger != nil {
			r.logger.Error("failed syncing application commands", "error", err)
		}
	})
	return r
}

// before registers hooks running before command execution
func (r *Router) Before(hooks ...HookFunc) *Router {
	r.beforeHooks = append(r.beforeHooks, hooks...)
	return r
}

// after registers hooks running after command execution
func (r *Router) After(hooks ...AfterHookFunc) *Router {
	r.afterHooks = append(r.afterHooks, hooks...)
	return r
}

// on registers a functional chat command
func (r *Router) On(name string, handler ChatCommandHandler, opts ...CommandOption) *Router {
	cfg := ChatCommandConfig{Name: name}
	for _, opt := range opts {
		opt(&cfg)
	}
	r.MustRegisterCommand(NewFuncChatCommand(cfg, handler))
	return r
}

// command registers a functional chat command
func (r *Router) Command(name string, handler ChatCommandHandler, opts ...CommandOption) *Router {
	return r.On(name, handler, opts...)
}

// onButton registers a functional button component handler
func (r *Router) OnButton(pattern string, handler ButtonHandler, opts ...ButtonOption) *Router {
	cfg := ButtonCommandConfig{CustomID: pattern}
	for _, opt := range opts {
		opt(&cfg)
	}
	r.MustRegisterButtonCommand(NewFuncButtonCommand(cfg, handler))
	return r
}

// onSelectMenu registers a functional select menu component handler
func (r *Router) OnSelectMenu(pattern string, handler SelectMenuHandler, opts ...SelectMenuOption) *Router {
	cfg := SelectMenuCommandConfig{CustomID: pattern}
	for _, opt := range opts {
		opt(&cfg)
	}
	r.MustRegisterSelectMenuCommand(NewFuncSelectMenuCommand(cfg, handler))
	return r
}

// onModal registers a functional modal submission handler
func (r *Router) OnModal(pattern string, handler ModalHandler, opts ...ModalOption) *Router {
	cfg := ModalCommandConfig{CustomID: pattern}
	for _, opt := range opts {
		opt(&cfg)
	}
	r.MustRegisterModalCommand(NewFuncModalCommand(cfg, handler))
	return r
}

// onUserContextMenu registers a functional user context menu handler
func (r *Router) OnUserContextMenu(name string, handler UserContextMenuHandler, guildID ...wumpkit.Snowflake) *Router {
	cfg := UserContextMenuCommandConfig{Name: name}
	if len(guildID) > 0 {
		cfg.GuildID = &guildID[0]
	}
	r.MustRegisterUserContextMenuCommand(NewFuncUserContextMenuCommand(cfg, handler))
	return r
}

// onMessageContextMenu registers a functional message context menu handler
func (r *Router) OnMessageContextMenu(name string, handler MessageContextMenuHandler, guildID ...wumpkit.Snowflake) *Router {
	cfg := MessageContextMenuCommandConfig{Name: name}
	if len(guildID) > 0 {
		cfg.GuildID = &guildID[0]
	}
	r.MustRegisterMessageContextMenuCommand(NewFuncMessageContextMenuCommand(cfg, handler))
	return r
}

// HandleMessage handles an incoming chat message event.
func (r *Router) HandleMessage(ctx context.Context, e wumpkit.MessageCreateEvent) {
	msg := &e.Message
	if msg.Author.Bot || msg.WebhookID != 0 {
		return
	}

	var prefix string
	if r.dynamicPrefixFunc != nil {
		var err error
		prefix, err = r.dynamicPrefixFunc(ctx, msg)
		if err != nil {
			if r.logger != nil {
				r.logger.Error("dynamic prefix resolution failed", "error", err)
			}
			return
		}
	} else if r.prefixFunc != nil {
		prefix = r.prefixFunc(msg)
	}
	if !strings.HasPrefix(msg.Content, prefix) {
		return
	}

	tokens := Tokenize(msg.Content[len(prefix):])
	if len(tokens) == 0 {
		return
	}

	cmd, rawArgs := r.resolvePrefixCommand(tokens)
	if cmd == nil {
		return
	}

	msgCtx := &MessageCommandContext{
		Context: ctx,
		cmd:     cmd,
		client:  r.client,
		event:   e,
	}

	if msgCtx.InGuild() {
		if r.permissionResolver != nil {
			uPerms, aPerms, err := r.permissionResolver(msgCtx)
			if err == nil {
				msgCtx.SetUserPermissions(uPerms)
				msgCtx.SetAppPermissions(aPerms)
			}
		} else if r.client != nil {
			r.resolveMessageGuildPermissions(msgCtx)
		}
	}

	if excluded, ok := r.checkContextExclusion(cmd, msgCtx); ok {
		r.errorHandler.OnContextExcluded(msgCtx, excluded)
		return
	}

	parsedInput := ParseTokens(rawArgs, cmd.Config().Flags)
	var flags Flags
	if len(parsedInput.Flags) > 0 {
		flags = Flags{values: parsedInput.Flags}
	}

	var args Args
	argValues, bindErr := r.bindMessageArgs(cmd, parsedInput, msg)
	if bindErr.errType != bindErrNone {
		if bindErr.errType == bindErrInvalid {
			r.errorHandler.OnInvalidArgument(msgCtx, bindErr.name, bindErr.argType, bindErr.val)
		} else {
			r.errorHandler.OnMissingArgument(msgCtx, bindErr.name, bindErr.argType)
		}
		return
	}

	if len(argValues) > 0 {
		args.values = argValues
	}

	msgCtx.args = args
	msgCtx.flags = flags

	r.executePipeline(msgCtx, cmd, func(ctx Context) error {
		return cmd.Run(msgCtx)
	})
}

// HandleInteraction processes incoming Discord interaction events.
func (r *Router) HandleInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent) {
	if e.Interaction == nil {
		return
	}

	switch it := e.Interaction.(type) {
	case *wumpkit.ChatInputCommandInteraction:
		r.handleChatInputInteraction(ctx, e, it)
	case *wumpkit.UserContextMenuCommandInteraction:
		r.handleUserContextMenuCommandInteraction(ctx, e, it)
	case *wumpkit.MessageContextMenuCommandInteraction:
		r.handleMessageContextMenuCommandInteraction(ctx, e, it)
	case *wumpkit.AutoCompleteInteraction:
		r.handleAutoCompleteInteraction(ctx, e, it)
	case *wumpkit.ComponentInteraction:
		r.handleComponentInteraction(ctx, e, it)
	case *wumpkit.ModalSubmitInteraction:
		r.handleModalSubmitInteraction(ctx, e, it)
	}
}

// SyncApplicationCommands registers all commands with Discord's Application Command REST API.
// It automatically diffs against existing registered commands to prevent burning Discord rate limits.
func (r *Router) SyncApplicationCommands(ctx context.Context, appID wumpkit.Snowflake, guildIDs ...wumpkit.Snowflake) error {
	return r.syncApplicationCommandsInternal(ctx, appID, false, guildIDs...)
}

// ForceSyncApplicationCommands unconditionally registers all commands with Discord's REST API, bypassing diffing.
func (r *Router) ForceSyncApplicationCommands(ctx context.Context, appID wumpkit.Snowflake, guildIDs ...wumpkit.Snowflake) error {
	return r.syncApplicationCommandsInternal(ctx, appID, true, guildIDs...)
}

func (r *Router) syncApplicationCommandsInternal(ctx context.Context, appID wumpkit.Snowflake, force bool, guildIDs ...wumpkit.Snowflake) error {
	if r.client == nil {
		return fmt.Errorf("router: client is nil")
	}

	appCommands := r.buildAppCommands()

	if len(guildIDs) > 0 {
		return r.syncGuildCommands(ctx, appID, guildIDs, appCommands, force)
	}
	return r.syncGlobalCommands(ctx, appID, appCommands, force)
}

func (r *Router) buildAppCommands() []wumpkit.ApplicationCommand {
	var appCommands []wumpkit.ApplicationCommand

	seenChatCmds := make(map[string]bool)
	for _, cmd := range r.Commands() {
		if cmd == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(cmd.Config().Name))
		if seenChatCmds[name] {
			continue
		}
		seenChatCmds[name] = true

		if slashCmd, ok := buildSlashCommand(cmd); ok {
			appCommands = append(appCommands, slashCmd)
		}
	}

	for _, cmd := range r.UserContextMenuCommands() {
		if cmd != nil {
			appCommands = append(appCommands, buildUserCommand(cmd))
		}
	}

	for _, cmd := range r.MessageContextMenuCommands() {
		if cmd != nil {
			appCommands = append(appCommands, buildMessageCommand(cmd))
		}
	}

	return appCommands
}

func (r *Router) resolvePrefixCommand(tokens []string) (ChatCommand, []string) {
	var cmd ChatCommand
	var cmdTokenCount int

	maxTokens := min(len(tokens), 3)
	for i := maxTokens; i > 0; i-- {
		attempt := strings.Join(tokens[:i], " ")
		if c, ok := r.GetCommand(attempt); ok {
			if wumpkit.BitFieldHas(c.Config().DisabledIn, DisabledInMessage) {
				continue
			}
			cmd = c
			cmdTokenCount = i
			break
		}
	}

	if cmd == nil {
		return nil, nil
	}

	tokens = tokens[cmdTokenCount:]
	posIdx := 0

	for posIdx < len(tokens) {
		nextToken := tokens[posIdx]
		matched := false
		for _, sub := range cmd.Config().Subcommands {
			subCfg := sub.Config()
			if wumpkit.BitFieldHas(subCfg.DisabledIn, DisabledInMessage) {
				continue
			}
			if strings.EqualFold(subCfg.Name, nextToken) || slices.ContainsFunc(subCfg.Aliases, func(a string) bool { return strings.EqualFold(a, nextToken) }) {
				cmd = sub
				posIdx++
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}

	return cmd, tokens[posIdx:]
}

func (r *Router) bindMessageArgs(cmd ChatCommand, parsedInput *ParsedInput, msg *wumpkit.Message) (map[string]any, bindError) {
	cfg := cmd.Config()
	if len(cfg.Args) == 0 && len(cfg.Flags) == 0 {
		return nil, bindError{}
	}

	argValues := make(map[string]any, len(cfg.Args)+len(cfg.Flags))
	argPosIdx := 0
	attachmentIdx := 0

	for _, argDef := range cfg.Args {
		if argDef.Type == ArgTypeAttachment {
			if err := bindAttachmentArg(argDef, msg, &attachmentIdx, argValues); err != nil {
				return nil, *err
			}
			continue
		}

		vc := valueConstraints{
			MinLength: argDef.MinLength,
			MaxLength: argDef.MaxLength,
			MinValue:  argDef.MinValue,
			MaxValue:  argDef.MaxValue,
			Choices:   argDef.Choices,
		}

		if argDef.Endless {
			if err := bindEndlessArg(argDef, parsedInput.Positional, &argPosIdx, vc, argValues); err != nil {
				return nil, *err
			}
		} else {
			if err := bindPositionalArg(argDef, parsedInput.Positional, &argPosIdx, vc, argValues); err != nil {
				return nil, *err
			}
		}
	}

	for _, flagDef := range cfg.Flags {
		if err := bindFlag(flagDef, parsedInput.Flags, argValues); err != nil {
			return nil, *err
		}
	}

	return argValues, bindError{}
}

func (r *Router) handleChatInputInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.ChatInputCommandInteraction) {
	cmd, ok := r.GetCommand(it.Data.Name)
	if !ok || wumpkit.BitFieldHas(cmd.Config().DisabledIn, DisabledInSlash) {
		return
	}

	targetCmd, options := r.traverseSlashSubcommands(cmd, it.Data.Options)
	if targetCmd == nil {
		return
	}

	var args Args
	if len(options) > 0 {
		argValues := make(map[string]any, len(options))
		for _, opt := range options {
			argValues[opt.Name] = decodeOptionValue(opt)
		}
		args = Args{values: argValues}
	}

	slashCtx := &SlashCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     targetCmd,
			client:  r.client,
			event:   e,
		},
		args: args,
	}

	if excluded, ok := r.checkContextExclusion(targetCmd, slashCtx); ok {
		r.errorHandler.OnContextExcluded(slashCtx, excluded)
		return
	}

	r.executePipeline(slashCtx, targetCmd, func(ctx Context) error {
		return targetCmd.Run(slashCtx)
	})
}

func (r *Router) handleUserContextMenuCommandInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.UserContextMenuCommandInteraction) {
	cmd, ok := r.GetUserContextMenuCommand(it.Data.Name)
	if !ok {
		return
	}

	uCtx := &UserContextMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     cmd,
			client:  r.client,
			event:   e,
		},
	}

	if excluded, ok := r.checkContextExclusion(cmd, uCtx); ok {
		r.errorHandler.OnContextExcluded(uCtx, excluded)
		return
	}

	r.executePipeline(uCtx, cmd, func(ctx Context) error {
		return cmd.Run(uCtx)
	})
}

func (r *Router) handleMessageContextMenuCommandInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.MessageContextMenuCommandInteraction) {
	cmd, ok := r.GetMessageContextMenuCommand(it.Data.Name)
	if !ok {
		return
	}

	mCtx := &MessageContextMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     cmd,
			client:  r.client,
			event:   e,
		},
	}

	if excluded, ok := r.checkContextExclusion(cmd, mCtx); ok {
		r.errorHandler.OnContextExcluded(mCtx, excluded)
		return
	}

	r.executePipeline(mCtx, cmd, func(ctx Context) error {
		return cmd.Run(mCtx)
	})
}

func (r *Router) handleAutoCompleteInteraction(ctx context.Context, _ wumpkit.InteractionCreateEvent, it *wumpkit.AutoCompleteInteraction) {
	cmdName := it.Data.Name

	focused := findFocusedOption(it.Data.Options)
	var optName string
	if focused != nil {
		optName = focused.Name
	}

	if handler, ok := r.GetAutocompleteHandler(cmdName, optName); ok {
		if err := handler(ctx, it); err != nil && r.logger != nil {
			r.logger.Error("autocomplete handler failed", "command", cmdName, "option", optName, "error", err.Error())
		}
		return
	}

	if cmd, ok := r.GetCommand(cmdName); ok {
		leafCmd, _ := r.traverseSlashSubcommands(cmd, it.Data.Options)
		if leafCmd == nil {
			return
		}

		if completer, ok := leafCmd.(Autocompleter); ok {
			if err := completer.Autocomplete(ctx, it); err != nil && r.logger != nil {
				r.logger.Error("command autocomplete failed", "command", cmdName, "option", optName, "error", err.Error())
			}
		}
	}
}

func (r *Router) handleComponentInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.ComponentInteraction) {
	if it.Data.ComponentType == wumpkit.ComponentTypeButton {
		r.dispatchButtonInteraction(ctx, e, it)
	} else {
		r.dispatchSelectMenuInteraction(ctx, e, it)
	}
}

func (r *Router) dispatchButtonInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.ComponentInteraction) {
	m, ok := r.MatchButtonCommand(it.Data.CustomID)
	if !ok || m.Value == nil {
		if r.logger != nil {
			r.logger.Debug("button interaction received with no registered command", "custom_id", it.Data.CustomID)
		}
		return
	}

	bCmd := m.Value
	bCtx := &ButtonCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     bCmd,
			client:  r.client,
			event:   e,
		},
		patternContext: patternContext{
			matchedPattern: m.Pattern,
			params:         m.Params,
		},
	}

	if excluded, ok := r.checkContextExclusion(bCmd, bCtx); ok {
		r.errorHandler.OnContextExcluded(bCtx, excluded)
		return
	}

	r.executePipeline(bCtx, bCmd, func(ctx Context) error {
		return bCmd.Run(bCtx)
	})
}

func (r *Router) dispatchSelectMenuInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.ComponentInteraction) {
	m, ok := r.MatchSelectMenuCommand(it.Data.CustomID)
	if !ok || m.Value == nil {
		if r.logger != nil {
			r.logger.Debug("select menu interaction received with no registered command", "custom_id", it.Data.CustomID)
		}
		return
	}

	sCmd := m.Value
	sCtx := &SelectMenuCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     sCmd,
			client:  r.client,
			event:   e,
		},
		patternContext: patternContext{
			matchedPattern: m.Pattern,
			params:         m.Params,
		},
	}

	if excluded, ok := r.checkContextExclusion(sCmd, sCtx); ok {
		r.errorHandler.OnContextExcluded(sCtx, excluded)
		return
	}

	r.executePipeline(sCtx, sCmd, func(ctx Context) error {
		return sCmd.Run(sCtx)
	})
}

func (r *Router) handleModalSubmitInteraction(ctx context.Context, e wumpkit.InteractionCreateEvent, it *wumpkit.ModalSubmitInteraction) {
	m, found := r.MatchModalCommand(it.Data.CustomID)
	if !found || m.Value == nil {
		if r.logger != nil {
			r.logger.Debug("modal interaction received with no registered command", "custom_id", it.Data.CustomID)
		}
		return
	}
	cmd := m.Value

	modalCtx := &ModalCommandContext{
		interactionBaseContext: interactionBaseContext{
			Context: ctx,
			cmd:     cmd,
			client:  r.client,
			event:   e,
		},
		patternContext: patternContext{
			matchedPattern: m.Pattern,
			params:         m.Params,
		},
	}

	if excluded, ok := r.checkContextExclusion(cmd, modalCtx); ok {
		r.errorHandler.OnContextExcluded(modalCtx, excluded)
		return
	}

	r.executePipeline(modalCtx, cmd, func(ctx Context) error {
		return cmd.Run(modalCtx)
	})
}

func (r *Router) traverseSlashSubcommands(cmd ChatCommand, options []wumpkit.ChatInputInteractionCommandOption) (ChatCommand, []wumpkit.ChatInputInteractionCommandOption) {
	targetCmd := cmd
	opts := options

	for len(opts) > 0 && (opts[0].Type == wumpkit.ApplicationCommandOptionTypeSubCommandGroup || opts[0].Type == wumpkit.ApplicationCommandOptionTypeSubCommand) {
		subOpt := opts[0]
		var matched bool
		for _, sub := range targetCmd.Config().Subcommands {
			if strings.EqualFold(sub.Config().Name, subOpt.Name) {
				if wumpkit.BitFieldHas(sub.Config().DisabledIn, DisabledInSlash) {
					return nil, nil
				}
				targetCmd = sub
				opts = subOpt.Options
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}

	return targetCmd, opts
}

func (r *Router) executePipeline(ctx Context, cmd any, runner func(ctx Context) error) {
	cfg := ExtractCommonConfig(cmd)

	if cfg.UserPermissions != 0 && ctx.InGuild() {
		if missing := ctx.UserPermissions().Missing(cfg.UserPermissions); missing != 0 {
			r.errorHandler.OnUserPermissionMissing(ctx, missing)
			return
		}
	}

	if cfg.AppPermissions != 0 && ctx.InGuild() {
		if missing := ctx.AppPermissions().Missing(cfg.AppPermissions); missing != 0 {
			r.errorHandler.OnAppPermissionMissing(ctx, missing)
			return
		}
	}

	handler := CommandHandler(runner)

	for _, mw := range slices.Backward(r.middlewares) {
		handler = mw(handler)
	}

	var err error
	func() {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("router: panic recovered during command execution: %v", p)
				if r.logger != nil {
					r.logger.Error("panic recovered during command execution",
						"command", cfg.Name,
						"panic", fmt.Sprint(p),
					)
				}
			}
		}()
		for _, hook := range r.beforeHooks {
			if err = hook(ctx); err != nil {
				return
			}
		}
		err = handler(ctx)
		for _, hook := range r.afterHooks {
			hook(ctx, err)
		}
	}()

	if err != nil {
		var cdErr *CooldownError
		if errors.As(err, &cdErr) {
			r.errorHandler.OnCooldown(ctx, cdErr.Remaining, cdErr.Scope)
			return
		}

		r.errorHandler.OnCommandError(ctx, cmd, err)
	}
}

func (r *Router) checkContextExclusion(cmd any, ctx Context) (ExcludedIn, bool) {
	cfg := ExtractCommonConfig(cmd)
	if cfg.ExcludedIn == 0 {
		return 0, false
	}

	guildID := ctx.GuildID()
	if guildID == nil {
		if wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeDM) {
			return ExcludeDM, true
		}
	} else {
		if wumpkit.BitFieldHas(cfg.ExcludedIn, ExcludeGuild) {
			return ExcludeGuild, true
		}
	}

	if f := getInteractionFields(ctx.Interaction()); f != nil && (f.ChannelID != 0 || f.Channel.ID != 0) {
		if excluded, ok := matchChannelTypeExclusion(cfg.ExcludedIn, f.Channel.Type); ok {
			return excluded, true
		}
	}

	return 0, false
}

func (r *Router) syncGuildCommands(ctx context.Context, appID wumpkit.Snowflake, guildIDs []wumpkit.Snowflake, appCommands []wumpkit.ApplicationCommand, force bool) error {
	for _, gID := range guildIDs {
		if !force {
			existing, err := r.client.FetchGuildApplicationCommands(ctx, appID, gID)
			if err == nil && applicationCommandsEqual(existing, appCommands) {
				if r.logger != nil {
					r.logger.Info("guild application commands up-to-date, skipping sync",
						"app_id", appID.String(),
						"guild_id", gID.String(),
						"command_count", len(appCommands),
					)
				}
				continue
			}
		}

		_, err := r.client.BulkOverwriteGuildApplicationCommands(ctx, appID, gID, appCommands)
		if err != nil {
			if r.logger != nil {
				r.logger.Error("failed syncing guild application commands",
					"app_id", appID.String(),
					"guild_id", gID.String(),
					"command_count", len(appCommands),
					"error", err.Error(),
				)
			}
			return fmt.Errorf("failed syncing commands to guild %s: %w", gID.String(), err)
		}
		if r.logger != nil {
			r.logger.Info("synchronized guild application commands",
				"app_id", appID.String(),
				"guild_id", gID.String(),
				"command_count", len(appCommands),
			)
		}
	}
	return nil
}

func (r *Router) syncGlobalCommands(ctx context.Context, appID wumpkit.Snowflake, appCommands []wumpkit.ApplicationCommand, force bool) error {
	if !force {
		existing, err := r.client.FetchGlobalApplicationCommands(ctx, appID)
		if err == nil && applicationCommandsEqual(existing, appCommands) {
			if r.logger != nil {
				r.logger.Info("global application commands up-to-date, skipping sync",
					"app_id", appID.String(),
					"command_count", len(appCommands),
				)
			}
			return nil
		}
	}

	_, err := r.client.BulkOverwriteGlobalApplicationCommands(ctx, appID, appCommands)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed syncing global application commands",
				"app_id", appID.String(),
				"command_count", len(appCommands),
				"error", err.Error(),
			)
		}
		return fmt.Errorf("failed syncing global application commands: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("synchronized global application commands",
			"app_id", appID.String(),
			"command_count", len(appCommands),
		)
	}
	return nil
}

func (r *Router) resolveMessageGuildPermissions(msgCtx *MessageCommandContext) {
	guildID := msgCtx.GuildID()
	if guildID == nil || r.client == nil {
		return
	}

	guild, err := r.client.Guild(msgCtx.Context, *guildID)
	if err != nil {
		return
	}

	roles, err := r.client.GuildRoles(msgCtx.Context, *guildID)
	if err != nil {
		return
	}

	authorID := msgCtx.Author().ID
	member, err := r.client.Member(msgCtx.Context, *guildID, authorID)
	if err == nil {
		msgCtx.SetAuthorMember(&member)
		userPerms := computeGuildPermissions(authorID, guild.OwnerID, guild.ID, roles, member.RoleIDs)
		msgCtx.SetUserPermissions(userPerms)
	}

	botMember, err := r.client.FetchCurrentUserGuildMember(msgCtx.Context, *guildID)
	if err == nil {
		appPerms := computeGuildPermissions(botMember.ID, guild.OwnerID, guild.ID, roles, botMember.RoleIDs)
		msgCtx.SetAppPermissions(appPerms)
	}
}

func computeGuildPermissions(userID, ownerID, guildID wumpkit.Snowflake, roles map[wumpkit.Snowflake]wumpkit.Role, memberRoleIDs []wumpkit.Snowflake) wumpkit.Permissions {
	if userID != 0 && userID == ownerID {
		return ^wumpkit.Permissions(0)
	}

	var perms wumpkit.Permissions
	if everyone, ok := roles[guildID]; ok {
		perms |= everyone.Permissions
	}

	for _, roleID := range memberRoleIDs {
		if role, ok := roles[roleID]; ok {
			perms |= role.Permissions
		}
	}

	if perms.Has(wumpkit.PermissionAdministrator) {
		return ^wumpkit.Permissions(0)
	}

	return perms
}

func applicationCommandsEqual(existing, desired []wumpkit.ApplicationCommand) bool {
	if len(existing) != len(desired) {
		return false
	}

	type cmdKey struct {
		name    string
		cmdType wumpkit.ApplicationCommandType
	}

	existingMap := make(map[cmdKey]map[string]any, len(existing))
	for _, cmd := range existing {
		cleaned, err := cleanCommandMap(cmd)
		if err != nil {
			return false
		}
		existingMap[cmdKey{name: strings.ToLower(cmd.GetName()), cmdType: cmd.GetType()}] = cleaned
	}

	for _, cmd := range desired {
		cleaned, err := cleanCommandMap(cmd)
		if err != nil {
			return false
		}
		key := cmdKey{name: strings.ToLower(cmd.GetName()), cmdType: cmd.GetType()}
		existCleaned, found := existingMap[key]
		if !found {
			return false
		}
		if !reflect.DeepEqual(existCleaned, cleaned) {
			return false
		}
	}

	return true
}

func cleanCommandMap(cmd wumpkit.ApplicationCommand) (map[string]any, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	delete(m, "id")
	delete(m, "application_id")
	delete(m, "version")
	delete(m, "guild_id")
	delete(m, "default_permission")
	return m, nil
}

type bindErrorType int

const (
	bindErrNone bindErrorType = iota
	bindErrInvalid
	bindErrMissing
)

type bindError struct {
	errType bindErrorType
	name    string
	argType string
	val     string
}

type valueConstraints struct {
	MinLength *int
	MaxLength *int
	MinValue  *float64
	MaxValue  *float64
	Choices   []Choice
}

func (vc valueConstraints) validate(val any, rawVal string, isNumeric bool) bool {
	if vc.MinLength != nil || vc.MaxLength != nil {
		var length int
		if isNumeric {
			cleanDigits := strings.TrimPrefix(rawVal, "-")
			cleanDigits = strings.ReplaceAll(cleanDigits, ".", "")
			length = utf8.RuneCountInString(cleanDigits)
		} else {
			strVal, ok := val.(string)
			if !ok {
				strVal = fmt.Sprint(val)
			}
			length = utf8.RuneCountInString(strVal)
		}

		if vc.MinLength != nil && length < *vc.MinLength {
			return false
		}
		if vc.MaxLength != nil && length > *vc.MaxLength {
			return false
		}
	}

	if vc.MinValue != nil || vc.MaxValue != nil {
		var num float64
		switch v := val.(type) {
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case float64:
			num = v
		case float32:
			num = float64(v)
		default:
			if f, err := strconv.ParseFloat(rawVal, 64); err == nil {
				num = f
			} else {
				return false
			}
		}

		if vc.MinValue != nil && num < *vc.MinValue {
			return false
		}
		if vc.MaxValue != nil && num > *vc.MaxValue {
			return false
		}
	}

	if len(vc.Choices) > 0 {
		matched := false
		valStr := fmt.Sprint(val)
		for _, c := range vc.Choices {
			if fmt.Sprint(c.Value) == valStr || strings.EqualFold(c.Name, rawVal) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func bindAttachmentArg(argDef ArgDef, msg *wumpkit.Message, attachmentIdx *int, out map[string]any) *bindError {
	if msg != nil && *attachmentIdx < len(msg.Attachments) {
		att := msg.Attachments[*attachmentIdx]
		*attachmentIdx++
		out[argDef.Name] = att
		return nil
	}
	if argDef.Required {
		return &bindError{
			errType: bindErrMissing,
			name:    argDef.Name,
			argType: argDef.Type.String(),
		}
	}
	if argDef.Default != nil {
		out[argDef.Name] = argDef.Default
	}
	return nil
}

func bindEndlessArg(argDef ArgDef, positional []string, posIdx *int, vc valueConstraints, out map[string]any) *bindError {
	if *posIdx < len(positional) {
		endlessStr := strings.Join(positional[*posIdx:], " ")
		if !vc.validate(endlessStr, endlessStr, false) {
			return &bindError{
				errType: bindErrInvalid,
				name:    argDef.Name,
				argType: argDef.Type.String(),
				val:     endlessStr,
			}
		}
		if argDef.Parser != nil {
			parsed, pErr := argDef.Parser(endlessStr)
			if pErr != nil {
				return &bindError{
					errType: bindErrInvalid,
					name:    argDef.Name,
					argType: argDef.Type.String(),
					val:     endlessStr,
				}
			}
			out[argDef.Name] = parsed
		} else {
			out[argDef.Name] = endlessStr
		}
		*posIdx = len(positional)
		return nil
	}
	if argDef.Required {
		return &bindError{
			errType: bindErrMissing,
			name:    argDef.Name,
			argType: argDef.Type.String(),
		}
	}
	if argDef.Default != nil {
		out[argDef.Name] = argDef.Default
	}
	return nil
}

func bindPositionalArg(argDef ArgDef, positional []string, posIdx *int, vc valueConstraints, out map[string]any) *bindError {
	if *posIdx < len(positional) {
		valStr := positional[*posIdx]
		*posIdx++

		isNumeric := argDef.Type == ArgTypeInt || argDef.Type == ArgTypeFloat
		var typedVal any
		var err error
		if argDef.Parser != nil {
			typedVal, err = argDef.Parser(valStr)
		} else {
			typedVal, err = parseArgType(valStr, argDef.Type)
		}
		if err != nil || !vc.validate(typedVal, valStr, isNumeric) {
			return &bindError{
				errType: bindErrInvalid,
				name:    argDef.Name,
				argType: argDef.Type.String(),
				val:     valStr,
			}
		}
		out[argDef.Name] = typedVal
		return nil
	}
	if argDef.Required {
		return &bindError{
			errType: bindErrMissing,
			name:    argDef.Name,
			argType: argDef.Type.String(),
		}
	}
	if argDef.Default != nil {
		out[argDef.Name] = argDef.Default
	}
	return nil
}

func bindFlag(flagDef FlagDef, parsedFlags map[string]string, out map[string]any) *bindError {
	valStr, exists := parsedFlags[flagDef.Name]
	if !exists && flagDef.Short != "" {
		valStr, exists = parsedFlags[flagDef.Short]
	}

	if exists {
		vc := valueConstraints{
			MinLength: flagDef.MinLength,
			MaxLength: flagDef.MaxLength,
			MinValue:  flagDef.MinValue,
			MaxValue:  flagDef.MaxValue,
			Choices:   flagDef.Choices,
		}
		isNumeric := flagDef.Type == FlagTypeInt || flagDef.Type == FlagTypeFloat
		typedVal, err := parseFlagType(valStr, flagDef.Type)
		if err != nil || !vc.validate(typedVal, valStr, isNumeric) {
			return &bindError{
				errType: bindErrInvalid,
				name:    flagDef.Name,
				argType: flagDef.Type.String(),
				val:     valStr,
			}
		}
		out[flagDef.Name] = typedVal
	} else if flagDef.Default != nil {
		out[flagDef.Name] = flagDef.Default
	}
	return nil
}

func findFocusedOption(options []wumpkit.ChatInputInteractionCommandOption) *wumpkit.ChatInputInteractionCommandOption {
	for i := range options {
		if options[i].Focused {
			return &options[i]
		}
		for j := range options[i].Options {
			if options[i].Options[j].Focused {
				return &options[i].Options[j]
			}
		}
	}
	return nil
}

func decodeOptionValue(opt wumpkit.ChatInputInteractionCommandOption) any {
	switch opt.Type {
	case wumpkit.ApplicationCommandOptionTypeString:
		raw := opt.Value
		if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' && !bytes.ContainsRune(raw, '\\') {
			return string(raw[1 : len(raw)-1])
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
		return string(raw)
	case wumpkit.ApplicationCommandOptionTypeInteger:
		if i, err := strconv.Atoi(string(opt.Value)); err == nil {
			return i
		}
	case wumpkit.ApplicationCommandOptionTypeFloat:
		if f, err := strconv.ParseFloat(string(opt.Value), 64); err == nil {
			return f
		}
	case wumpkit.ApplicationCommandOptionTypeBool:
		if bytes.Equal(opt.Value, []byte("true")) {
			return true
		}
		if bytes.Equal(opt.Value, []byte("false")) {
			return false
		}
		var b bool
		if err := json.Unmarshal(opt.Value, &b); err == nil {
			return b
		}
	case wumpkit.ApplicationCommandOptionTypeUser,
		wumpkit.ApplicationCommandOptionTypeChannel,
		wumpkit.ApplicationCommandOptionTypeRole,
		wumpkit.ApplicationCommandOptionTypeMentionable,
		wumpkit.ApplicationCommandOptionTypeAttachment:
		trimmed := bytes.Trim(opt.Value, "\"")
		if id, err := wumpkit.ParseSnowflake(string(trimmed)); err == nil {
			return id
		}
		var id uint64
		if err := json.Unmarshal(opt.Value, &id); err == nil {
			return wumpkit.Snowflake(id)
		}
	}
	return opt.Value
}

func matchChannelTypeExclusion(excludedIn ExcludedIn, channelType wumpkit.ChannelType) (ExcludedIn, bool) {
	switch {
	case wumpkit.BitFieldHas(excludedIn, ExcludeGroupDM) && channelType == wumpkit.ChannelTypeGroupDM:
		return ExcludeGroupDM, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeTextChannel) && channelType == wumpkit.ChannelTypeGuildText:
		return ExcludeTextChannel, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeVoiceChannel) && channelType == wumpkit.ChannelTypeGuildVoice:
		return ExcludeVoiceChannel, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeStageChannel) && channelType == wumpkit.ChannelTypeGuildStageVoice:
		return ExcludeStageChannel, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeThread) && (channelType == wumpkit.ChannelTypePublicThread || channelType == wumpkit.ChannelTypePrivateThread || channelType == wumpkit.ChannelTypeAnnouncementThread):
		return ExcludeThread, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeForum) && channelType == wumpkit.ChannelTypeGuildForum:
		return ExcludeForum, true
	case wumpkit.BitFieldHas(excludedIn, ExcludeAnnouncement) && channelType == wumpkit.ChannelTypeGuildAnnouncement:
		return ExcludeAnnouncement, true
	default:
		return 0, false
	}
}
