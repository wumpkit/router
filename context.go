package router

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/wumpkit/wumpkit"
)

// context represents unified execution context for all commands
type Context interface {
	context.Context

	// core environment and metadata
	Client() *wumpkit.Client
	Author() wumpkit.User
	AuthorMember() *wumpkit.Member
	GuildID() *wumpkit.Snowflake
	ChannelID() wumpkit.Snowflake
	InGuild() bool
	AppPermissions() wumpkit.Permissions
	UserPermissions() wumpkit.Permissions
	ShardID() wumpkit.ShardID

	// event and command
	Event() any
	Command() any
	Message() *wumpkit.Message
	Interaction() wumpkit.Interaction

	// middleware store
	Set(key string, val any)
	Get(key string) (any, bool)

	// primary responses
	Reply(content string, opts ...ReplyOptions) (*wumpkit.Message, error)
	ReplyEmbed(embed wumpkit.Embed, ephemeral ...bool) (*wumpkit.Message, error)
	ReplyEphemeral(content string) (*wumpkit.Message, error)
	Followup(content string, opts ...ReplyOptions) (*wumpkit.Message, error)
	Edit(content string, opts ...ReplyOptions) (*wumpkit.Message, error)
	Delete() error
	Defer(ephemeral bool) error

	// arguments and flags
	Args() Args
	Flags() Flags
}

// replyoptions configures response parameters for commands and interactions
type ReplyOptions struct {
	// content is the text content of the message
	Content string
	// flags specifies message bitfield flags
	Flags wumpkit.MessageFlags
	// tts specifies whether this is a text-to-speech message
	TTS bool
	// embeds contains discord embed objects
	Embeds []wumpkit.Embed
	// components contains message interactive components
	Components []wumpkit.Component
	// files contains file attachments
	Files []wumpkit.File
	// allowedmentions controls mention suppression
	AllowedMentions *wumpkit.AllowedMentions
}

// contextconfig bundles configuration parameters for building a generic context
type ContextConfig struct {
	Command          any
	Client           *wumpkit.Client
	MessageEvent     *wumpkit.MessageCreateEvent
	InteractionEvent *wumpkit.InteractionCreateEvent
	Message          *wumpkit.Message
	Interaction      wumpkit.Interaction
	ShardID          wumpkit.ShardID
}

// newcontext creates a new context instance for generic use or testing
func NewContext(ctx context.Context, cfg ContextConfig) Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.MessageEvent != nil {
		var chatCmd ChatCommand
		if c, ok := cfg.Command.(ChatCommand); ok {
			chatCmd = c
		}
		return &MessageCommandContext{
			Context: ctx,
			cmd:     chatCmd,
			client:  cfg.Client,
			event:   *cfg.MessageEvent,
		}
	}
	if cfg.Message != nil {
		var chatCmd ChatCommand
		if c, ok := cfg.Command.(ChatCommand); ok {
			chatCmd = c
		}
		return &MessageCommandContext{
			Context: ctx,
			cmd:     chatCmd,
			client:  cfg.Client,
			event:   wumpkit.MessageCreateEvent{ShardID: cfg.ShardID, Message: *cfg.Message},
		}
	}
	if cfg.InteractionEvent != nil {
		return newInteractionContext(ctx, cfg.Command, cfg.Client, *cfg.InteractionEvent)
	}
	if cfg.Interaction != nil {
		return newInteractionContext(
			ctx,
			cfg.Command,
			cfg.Client,
			wumpkit.InteractionCreateEvent{ShardID: cfg.ShardID, Interaction: cfg.Interaction},
		)
	}
	return &interactionBaseContext{
		Context: ctx,
		cmd:     cfg.Command,
		client:  cfg.Client,
	}
}

// MessageCommandContext represents the execution context for prefix/message commands.
type MessageCommandContext struct {
	context.Context
	contextStore

	cmd             ChatCommand
	client          *wumpkit.Client
	event           wumpkit.MessageCreateEvent
	args            Args
	flags           Flags
	userPermissions wumpkit.Permissions
	appPermissions  wumpkit.Permissions
	authorMember    *wumpkit.Member
	initialReply    *wumpkit.Message
	mu              sync.Mutex
}

func (c *MessageCommandContext) Command() any                        { return c.cmd }
func (c *MessageCommandContext) Client() *wumpkit.Client             { return c.client }
func (c *MessageCommandContext) Message() *wumpkit.Message           { return &c.event.Message }
func (c *MessageCommandContext) Interaction() wumpkit.Interaction    { return nil }
func (c *MessageCommandContext) Event() any                          { return c.event }
func (c *MessageCommandContext) ShardID() wumpkit.ShardID            { return c.event.ShardID }
func (c *MessageCommandContext) AuthorMember() *wumpkit.Member       { return c.authorMember }
func (c *MessageCommandContext) SetAuthorMember(m *wumpkit.Member)   { c.authorMember = m }
func (c *MessageCommandContext) AppPermissions() wumpkit.Permissions { return c.appPermissions }
func (c *MessageCommandContext) SetAppPermissions(perms wumpkit.Permissions) {
	c.appPermissions = perms
}
func (c *MessageCommandContext) UserPermissions() wumpkit.Permissions { return c.userPermissions }
func (c *MessageCommandContext) SetUserPermissions(perms wumpkit.Permissions) {
	c.userPermissions = perms
}
func (c *MessageCommandContext) Args() Args   { return c.args }
func (c *MessageCommandContext) Flags() Flags { return c.flags }

func (c *MessageCommandContext) Author() wumpkit.User {
	return c.event.Message.Author
}

func (c *MessageCommandContext) GuildID() *wumpkit.Snowflake {
	if c.event.Message.GuildID != 0 {
		return &c.event.Message.GuildID
	}
	return nil
}

func (c *MessageCommandContext) ChannelID() wumpkit.Snowflake {
	return c.event.Message.ChannelID
}

func (c *MessageCommandContext) InGuild() bool {
	return c.GuildID() != nil
}

func (c *MessageCommandContext) Reply(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	if c.client == nil {
		return nil, nil
	}
	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	msgContent := content
	if msgContent == "" && opt.Content != "" {
		msgContent = opt.Content
	}

	msg, err := c.client.CreateMessage(
		c.Context,
		c.event.Message.ChannelID,
		wumpkit.CreateMessageOptions{
			Content:         msgContent,
			TTS:             opt.TTS,
			Flags:           opt.Flags,
			Embeds:          opt.Embeds,
			AllowedMentions: opt.AllowedMentions,
			Files:           opt.Files,
		},
	)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.initialReply = &msg
	c.mu.Unlock()
	return &msg, nil
}

func (c *MessageCommandContext) Defer(ephemeral bool) error {
	if c.client != nil && c.event.Message.ChannelID != 0 {
		return c.client.TriggerTypingIndicator(c.Context, c.event.Message.ChannelID)
	}
	return nil
}

// replyembed sends an embed response
func (c *MessageCommandContext) ReplyEmbed(
	embed wumpkit.Embed,
	ephemeral ...bool,
) (*wumpkit.Message, error) {
	return c.Reply("", ReplyOptions{Embeds: []wumpkit.Embed{embed}})
}

// replyephemeral sends an ephemeral response
func (c *MessageCommandContext) ReplyEphemeral(content string) (*wumpkit.Message, error) {
	return c.Reply(content)
}

// followup sends a followup message
func (c *MessageCommandContext) Followup(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	return c.Reply(content, opts...)
}

// toLayoutComponents converts slice of components to layout components
func toLayoutComponents(comps []wumpkit.Component) *[]wumpkit.LayoutComponent {
	if comps == nil {
		return nil
	}
	res := make([]wumpkit.LayoutComponent, 0, len(comps))
	for _, c := range comps {
		if lc, ok := c.(wumpkit.LayoutComponent); ok {
			res = append(res, lc)
		}
	}
	return &res
}

// edit modifies the initial reply
func (c *MessageCommandContext) Edit(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initialReply == nil || c.client == nil {
		return nil, fmt.Errorf("router: no message to edit")
	}
	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	msgContent := content
	if msgContent == "" && opt.Content != "" {
		msgContent = opt.Content
	}
	var cnt *string
	if msgContent != "" {
		cnt = &msgContent
	}
	msg, err := c.client.EditMessage(
		c.Context,
		c.initialReply.ChannelID,
		c.initialReply.ID,
		wumpkit.EditMessageOptions{
			Content:         cnt,
			Embeds:          &opt.Embeds,
			AllowedMentions: opt.AllowedMentions,
			Components:      toLayoutComponents(opt.Components),
			Files:           opt.Files,
		},
	)
	if err != nil {
		return nil, err
	}
	c.initialReply = &msg
	return &msg, nil
}

// delete removes the initial reply
func (c *MessageCommandContext) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initialReply == nil || c.client == nil {
		return fmt.Errorf("router: no message to delete")
	}
	return c.client.DeleteMessage(c.Context, c.initialReply.ChannelID, c.initialReply.ID)
}

// interactionBaseContext provides common interaction lifecycle and response methods
type interactionBaseContext struct {
	context.Context
	contextStore

	cmd                  any
	client               *wumpkit.Client
	event                wumpkit.InteractionCreateEvent
	mu                   sync.RWMutex
	deferred             bool
	replied              bool
	initialReply         *wumpkit.Message
	overrideAuthorMember *wumpkit.Member
	overrideAppPerms     *wumpkit.Permissions
	overrideUserPerms    *wumpkit.Permissions
}

func (c *interactionBaseContext) Command() any                     { return c.cmd }
func (c *interactionBaseContext) Client() *wumpkit.Client          { return c.client }
func (c *interactionBaseContext) Interaction() wumpkit.Interaction { return c.event.Interaction }
func (c *interactionBaseContext) Event() any                       { return c.event }
func (c *interactionBaseContext) ShardID() wumpkit.ShardID         { return c.event.ShardID }
func (c *interactionBaseContext) Args() Args                       { return Args{} }
func (c *interactionBaseContext) Flags() Flags                     { return Flags{} }

func (c *interactionBaseContext) Author() wumpkit.User {
	if f := getInteractionFields(c.event.Interaction); f != nil {
		if u := f.Author(); u != nil {
			return *u
		}
		if f.User != nil {
			return *f.User
		}
		if f.Member != nil {
			return f.Member.User
		}
	}
	return wumpkit.User{}
}

func (c *interactionBaseContext) AuthorMember() *wumpkit.Member {
	if c.overrideAuthorMember != nil {
		return c.overrideAuthorMember
	}
	if f := getInteractionFields(c.event.Interaction); f != nil && f.Member != nil {
		return &f.Member.Member
	}
	return nil
}

func (c *interactionBaseContext) SetAuthorMember(m *wumpkit.Member) {
	c.overrideAuthorMember = m
}

func (c *interactionBaseContext) GuildID() *wumpkit.Snowflake {
	if f := getInteractionFields(c.event.Interaction); f != nil {
		if f.Guild != nil && f.Guild.ID != 0 {
			return &f.Guild.ID
		}
		if f.GuildID != 0 {
			return &f.GuildID
		}
	}
	switch it := c.event.Interaction.(type) {
	case *wumpkit.ChatInputCommandInteraction:
		if it.Data.GuildID != 0 {
			return &it.Data.GuildID
		}
	case *wumpkit.UserContextMenuCommandInteraction:
		if it.Data.GuildID != 0 {
			return &it.Data.GuildID
		}
	case *wumpkit.MessageContextMenuCommandInteraction:
		if it.Data.GuildID != 0 {
			return &it.Data.GuildID
		}
	case *wumpkit.ComponentInteraction:
		if it.GuildID != 0 {
			return &it.GuildID
		}
	case *wumpkit.ModalSubmitInteraction:
		if it.GuildID != 0 {
			return &it.GuildID
		}
	}
	return nil
}

func (c *interactionBaseContext) ChannelID() wumpkit.Snowflake {
	if f := getInteractionFields(c.event.Interaction); f != nil {
		if f.ChannelID != 0 {
			return f.ChannelID
		}
		if f.Channel.ID != 0 {
			return f.Channel.ID
		}
	}
	return 0
}

func (c *interactionBaseContext) Message() *wumpkit.Message {
	if in, ok := c.event.Interaction.(*wumpkit.ComponentInteraction); ok {
		return &in.Message
	}
	if in, ok := c.event.Interaction.(*wumpkit.ModalSubmitInteraction); ok {
		return in.Message
	}
	return nil
}

func (c *interactionBaseContext) AppPermissions() wumpkit.Permissions {
	if c.overrideAppPerms != nil {
		return *c.overrideAppPerms
	}
	if f := getInteractionFields(c.event.Interaction); f != nil {
		return f.AppPermissions
	}
	return 0
}

func (c *interactionBaseContext) SetAppPermissions(perms wumpkit.Permissions) {
	c.overrideAppPerms = &perms
}

func (c *interactionBaseContext) UserPermissions() wumpkit.Permissions {
	if c.overrideUserPerms != nil {
		return *c.overrideUserPerms
	}
	if f := getInteractionFields(c.event.Interaction); f != nil && f.Member != nil {
		return f.Member.Permissions
	}
	return 0
}

func (c *interactionBaseContext) SetUserPermissions(perms wumpkit.Permissions) {
	c.overrideUserPerms = &perms
}

func (c *interactionBaseContext) InGuild() bool {
	if c.GuildID() != nil {
		return true
	}
	if f := getInteractionFields(c.event.Interaction); f != nil && f.Member != nil {
		return true
	}
	return false
}

func (c *interactionBaseContext) Reply(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	msgContent := content
	if msgContent == "" && opt.Content != "" {
		msgContent = opt.Content
	}

	if c.client == nil || c.event.Interaction == nil {
		return nil, nil
	}

	if !c.deferred && !c.replied {
		err := c.client.RespondMessage(
			c.Context,
			c.event.Interaction,
			msgContent,
			wumpkit.RespondMessageOptions{
				Content:         msgContent,
				Flags:           opt.Flags,
				TTS:             opt.TTS,
				Embeds:          opt.Embeds,
				Components:      opt.Components,
				AllowedMentions: opt.AllowedMentions,
				Files:           opt.Files,
			},
		)
		if err != nil {
			return nil, err
		}
		c.replied = true
		return nil, nil
	}

	if c.deferred && !c.replied {
		c.replied = true
		return c.editReplyInternal(msgContent, opt)
	}

	msg, err := c.client.CreateFollowupMessage(
		c.Context,
		c.event.Interaction.GetApplicationID(),
		c.event.Interaction.GetToken(),
		wumpkit.CreateFollowupMessageOptions{
			Content:         msgContent,
			TTS:             opt.TTS,
			Flags:           opt.Flags,
			Embeds:          opt.Embeds,
			Components:      opt.Components,
			Files:           opt.Files,
			AllowedMentions: opt.AllowedMentions,
		},
	)
	if err != nil {
		return nil, err
	}
	c.initialReply = &msg
	return &msg, nil
}

func (c *interactionBaseContext) Defer(ephemeral bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.event.Interaction == nil {
		return nil
	}
	if c.deferred || c.replied {
		return nil
	}
	err := c.client.DeferResponse(c.Context, c.event.Interaction, ephemeral)
	if err == nil {
		c.deferred = true
	}
	return err
}

func (c *interactionBaseContext) editReplyInternal(
	msgContent string,
	opt ReplyOptions,
) (*wumpkit.Message, error) {
	if c.event.Interaction != nil && c.client != nil {
		var cnt *string
		if msgContent != "" {
			cnt = &msgContent
		}
		var flags *wumpkit.MessageFlags
		if opt.Flags != 0 {
			flags = &opt.Flags
		}
		msg, err := c.client.EditOriginalInteractionResponse(
			c.Context,
			c.event.Interaction.GetApplicationID(),
			c.event.Interaction.GetToken(),
			wumpkit.EditFollowupMessageOptions{
				Content:         cnt,
				Embeds:          &opt.Embeds,
				Components:      &opt.Components,
				Files:           opt.Files,
				AllowedMentions: opt.AllowedMentions,
				Flags:           flags,
			},
		)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	}

	return nil, fmt.Errorf("router: no interaction reply to edit")
}

// replyembed sends an embed response
func (c *interactionBaseContext) ReplyEmbed(
	embed wumpkit.Embed,
	ephemeral ...bool,
) (*wumpkit.Message, error) {
	var flags wumpkit.MessageFlags
	if len(ephemeral) > 0 && ephemeral[0] {
		flags = wumpkit.MessageFlagEphemeral
	}
	return c.Reply("", ReplyOptions{Embeds: []wumpkit.Embed{embed}, Flags: flags})
}

// replyephemeral sends an ephemeral text response
func (c *interactionBaseContext) ReplyEphemeral(content string) (*wumpkit.Message, error) {
	return c.Reply(content, ReplyOptions{Flags: wumpkit.MessageFlagEphemeral})
}

// followup sends an interaction followup message
func (c *interactionBaseContext) Followup(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.event.Interaction == nil {
		return nil, nil
	}

	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	msgContent := content
	if msgContent == "" && opt.Content != "" {
		msgContent = opt.Content
	}

	msg, err := c.client.CreateFollowupMessage(
		c.Context,
		c.event.Interaction.GetApplicationID(),
		c.event.Interaction.GetToken(),
		wumpkit.CreateFollowupMessageOptions{
			Content:         msgContent,
			TTS:             opt.TTS,
			Flags:           opt.Flags,
			Embeds:          opt.Embeds,
			Components:      opt.Components,
			Files:           opt.Files,
			AllowedMentions: opt.AllowedMentions,
		},
	)
	if err != nil {
		return nil, err
	}
	c.initialReply = &msg
	return &msg, nil
}

// edit modifies the initial interaction response
func (c *interactionBaseContext) Edit(
	content string,
	opts ...ReplyOptions,
) (*wumpkit.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return c.editReplyInternal(content, opt)
}

// delete deletes the original interaction response
func (c *interactionBaseContext) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.event.Interaction == nil {
		return nil
	}
	return c.client.DeleteOriginalInteractionResponse(
		c.Context,
		c.event.Interaction.GetApplicationID(),
		c.event.Interaction.GetToken(),
	)
}

func (c *interactionBaseContext) updateComponentMessage(opts ...ReplyOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.event.Interaction == nil {
		return nil
	}

	in, ok := c.event.Interaction.(*wumpkit.ComponentInteraction)
	if !ok {
		return fmt.Errorf("router: interaction is not a component interaction")
	}

	var opt ReplyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	err := c.client.UpdateComponentMessage(c.Context, in, wumpkit.RespondMessageOptions{
		Content:         opt.Content,
		TTS:             opt.TTS,
		Embeds:          opt.Embeds,
		Components:      opt.Components,
		AllowedMentions: opt.AllowedMentions,
		Files:           opt.Files,
	})
	if err == nil {
		c.replied = true
	}
	return err
}

func (c *interactionBaseContext) deferComponentUpdate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.event.Interaction == nil {
		return nil
	}

	if c.deferred || c.replied {
		return nil
	}

	err := c.client.DeferComponentUpdate(c.Context, c.event.Interaction)
	if err == nil {
		c.deferred = true
	}
	return err
}

// SlashCommandContext represents the execution context for slash (ChatInput) commands.
type SlashCommandContext struct {
	interactionBaseContext
	args  Args
	flags Flags
}

func (c *SlashCommandContext) Args() Args   { return c.args }
func (c *SlashCommandContext) Flags() Flags { return c.flags }

// UserContextMenuCommandContext represents the execution context for user context menu commands.
type UserContextMenuCommandContext struct {
	interactionBaseContext
}

func (c *UserContextMenuCommandContext) TargetUser() *wumpkit.User {
	if in, ok := c.event.Interaction.(*wumpkit.UserContextMenuCommandInteraction); ok {
		if u, exists := in.Data.Resolved.Users[in.Data.TargetID]; exists {
			return &u
		}
	}
	return nil
}

func (c *UserContextMenuCommandContext) TargetMember() *wumpkit.PartialMember {
	if in, ok := c.event.Interaction.(*wumpkit.UserContextMenuCommandInteraction); ok {
		if m, exists := in.Data.Resolved.Members[in.Data.TargetID]; exists {
			return &m
		}
	}
	return nil
}

// MessageContextMenuCommandContext represents the execution context for message context menu commands.
type MessageContextMenuCommandContext struct {
	interactionBaseContext
}

func (c *MessageContextMenuCommandContext) TargetMessage() *wumpkit.Message {
	if in, ok := c.event.Interaction.(*wumpkit.MessageContextMenuCommandInteraction); ok {
		if m, exists := in.Data.Resolved.Messages[in.Data.TargetID]; exists {
			return &m
		}
	}
	return nil
}

// ButtonCommandContext represents the execution context for button component commands.
type ButtonCommandContext struct {
	interactionBaseContext
	patternContext
}

func (c *ButtonCommandContext) CustomID() string {
	if in, ok := c.event.Interaction.(*wumpkit.ComponentInteraction); ok {
		return in.Data.CustomID
	}
	return ""
}

func (c *ButtonCommandContext) UpdateComponentMessage(opts ...ReplyOptions) error {
	return c.updateComponentMessage(opts...)
}

func (c *ButtonCommandContext) DeferUpdate() error {
	return c.deferComponentUpdate()
}

// SelectMenuCommandContext represents the execution context for select menu component commands.
type SelectMenuCommandContext struct {
	interactionBaseContext
	patternContext
}

func (c *SelectMenuCommandContext) Values() []string {
	if in, ok := c.event.Interaction.(*wumpkit.ComponentInteraction); ok {
		res := make([]string, len(in.Data.Values))
		copy(res, in.Data.Values)
		return res
	}
	return nil
}

func (c *SelectMenuCommandContext) CustomID() string {
	if in, ok := c.event.Interaction.(*wumpkit.ComponentInteraction); ok {
		return in.Data.CustomID
	}
	return ""
}

func (c *SelectMenuCommandContext) UpdateComponentMessage(opts ...ReplyOptions) error {
	return c.updateComponentMessage(opts...)
}

func (c *SelectMenuCommandContext) DeferUpdate() error {
	return c.deferComponentUpdate()
}

// ModalCommandContext represents the execution context for modal submission commands.
type ModalCommandContext struct {
	interactionBaseContext
	patternContext
}

func (c *ModalCommandContext) CustomID() string {
	if in, ok := c.event.Interaction.(*wumpkit.ModalSubmitInteraction); ok {
		return in.Data.CustomID
	}
	return ""
}

func (c *ModalCommandContext) InputValue(customID string) string {
	return c.ModalValues()[customID]
}

func (c *ModalCommandContext) ModalValues() map[string]string {
	if in, ok := c.event.Interaction.(*wumpkit.ModalSubmitInteraction); ok {
		res := make(map[string]string)
		for _, comp := range in.Data.Components {
			extractComponentValues(comp, res)
		}
		return res
	}
	return nil
}

// patternContext holds matched pattern metadata and path parameters extracted from component routing.
type patternContext struct {
	matchedPattern string
	params         map[string]string
}

func (p *patternContext) MatchedPattern() string {
	return p.matchedPattern
}

func (p *patternContext) Param(name string) string {
	if p.params == nil {
		return ""
	}
	if val, ok := p.params[name]; ok {
		return val
	}
	for k, v := range p.params {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

func (p *patternContext) Params() map[string]string {
	if p.params == nil {
		return nil
	}
	res := make(map[string]string, len(p.params))
	for k, v := range p.params {
		res[k] = v
	}
	return res
}

// contextStore provides thread-safe key-value store functionality for Context.
type contextStore struct {
	mu    sync.RWMutex
	store map[string]any
}

func (s *contextStore) Set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		s.store = make(map[string]any)
	}
	s.store[key] = val
}

func (s *contextStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.store == nil {
		return nil, false
	}
	v, ok := s.store[key]
	return v, ok
}

func newInteractionContext(
	ctx context.Context,
	cmd any,
	client *wumpkit.Client,
	event wumpkit.InteractionCreateEvent,
) Context {
	switch it := event.Interaction.(type) {
	case *wumpkit.ChatInputCommandInteraction:
		var chatCmd ChatCommand
		if c, ok := cmd.(ChatCommand); ok {
			chatCmd = c
		}
		return &SlashCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     chatCmd,
				client:  client,
				event:   event,
			},
		}
	case *wumpkit.UserContextMenuCommandInteraction:
		return &UserContextMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     cmd,
				client:  client,
				event:   event,
			},
		}
	case *wumpkit.MessageContextMenuCommandInteraction:
		return &MessageContextMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     cmd,
				client:  client,
				event:   event,
			},
		}
	case *wumpkit.ComponentInteraction:
		if it.Data.ComponentType == wumpkit.ComponentTypeButton {
			return &ButtonCommandContext{
				interactionBaseContext: interactionBaseContext{
					Context: ctx,
					cmd:     cmd,
					client:  client,
					event:   event,
				},
			}
		}
		return &SelectMenuCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     cmd,
				client:  client,
				event:   event,
			},
		}
	case *wumpkit.ModalSubmitInteraction:
		return &ModalCommandContext{
			interactionBaseContext: interactionBaseContext{
				Context: ctx,
				cmd:     cmd,
				client:  client,
				event:   event,
			},
		}
	default:
		return &interactionBaseContext{
			Context: ctx,
			cmd:     cmd,
			client:  client,
			event:   event,
		}
	}
}

func getInteractionFields(in wumpkit.Interaction) *wumpkit.ApplicationCommandInteractionFields {
	if in == nil {
		return nil
	}
	switch it := in.(type) {
	case *wumpkit.ChatInputCommandInteraction:
		return &it.ApplicationCommandInteractionFields
	case *wumpkit.UserContextMenuCommandInteraction:
		return &it.ApplicationCommandInteractionFields
	case *wumpkit.MessageContextMenuCommandInteraction:
		return &it.ApplicationCommandInteractionFields
	case *wumpkit.AutoCompleteInteraction:
		return &it.ApplicationCommandInteractionFields
	case *wumpkit.ComponentInteraction:
		return &it.ApplicationCommandInteractionFields
	case *wumpkit.ModalSubmitInteraction:
		return &it.ApplicationCommandInteractionFields
	default:
		return nil
	}
}

func extractComponentValues(comp wumpkit.Component, res map[string]string) {
	switch c := comp.(type) {
	case *wumpkit.ActionRowComponent:
		for _, child := range c.Components {
			extractComponentValues(child, res)
		}
	case *wumpkit.TextInputComponent:
		if c.CustomID != "" {
			res[c.CustomID] = c.Value
		}
	}
}

// compile-time assertions
var (
	_ Context = (*MessageCommandContext)(nil)
	_ Context = (*SlashCommandContext)(nil)
	_ Context = (*UserContextMenuCommandContext)(nil)
	_ Context = (*MessageContextMenuCommandContext)(nil)
	_ Context = (*ButtonCommandContext)(nil)
	_ Context = (*SelectMenuCommandContext)(nil)
	_ Context = (*ModalCommandContext)(nil)
)
