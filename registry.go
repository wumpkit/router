package router

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// Sentinel errors for command registration and validation.
	ErrNilCommand           = errors.New("router: command cannot be nil")
	ErrEmptyCommandName     = errors.New("router: command name cannot be empty")
	ErrEmptyCustomID        = errors.New("router: custom ID cannot be empty")
	ErrCommandAlreadyExists = errors.New(
		"router: command name already registered",
	)
	ErrAliasAlreadyExists = errors.New(
		"router: alias already registered to another command",
	)
	ErrTargetCommandNotFound               = errors.New("router: target command not found")
	ErrUserContextMenuCommandAlreadyExists = errors.New(
		"router: user context menu command already registered",
	)
	ErrMessageContextMenuCommandAlreadyExists = errors.New(
		"router: message context menu command already registered",
	)
	ErrButtonCommandAlreadyExists = errors.New(
		"router: button command already registered",
	)
	ErrSelectMenuCommandAlreadyExists = errors.New(
		"router: select menu command already registered",
	)
	ErrModalCommandAlreadyExists = errors.New(
		"router: modal command already registered",
	)
	ErrNilCategory           = errors.New("router: category cannot be nil")
	ErrEmptyCategoryName     = errors.New("router: category name cannot be empty")
	ErrCategoryAlreadyExists = errors.New("router: category already registered")
	ErrCategoryCycle         = errors.New("router: category parent cycle detected")
	ErrCommandNameTooLong    = errors.New(
		"router: command name cannot exceed 3 words",
	)
	ErrAliasTooLong           = errors.New("router: alias cannot exceed 3 words")
	ErrRequiredArgWithDefault = errors.New(
		"router: required argument cannot have a default value",
	)
	ErrOptionalArgBeforeRequired = errors.New(
		"router: optional argument cannot precede a required argument",
	)
	ErrEndlessArgNotLast = errors.New(
		"router: endless argument must be the last argument",
	)
	ErrEmptyArgName      = errors.New("router: argument name cannot be empty")
	ErrDuplicateArgName  = errors.New("router: duplicate argument name")
	ErrEmptyFlagName     = errors.New("router: flag name cannot be empty")
	ErrDuplicateFlagName = errors.New(
		"router: duplicate flag name or short alias",
	)
	ErrInvalidLengthRange = errors.New(
		"router: invalid length range (MinLength cannot exceed MaxLength or be negative)",
	)
	ErrInvalidValueRange = errors.New(
		"router: invalid value range (MinValue cannot exceed MaxValue)",
	)
)

// CommandRegistrationError represents a detailed error during command, argument, flag, or alias registration.
type CommandRegistrationError struct {
	Err      error
	Command  string
	Alias    string
	Argument string
	Flag     string
}

func (e *CommandRegistrationError) Error() string {
	if e.Alias != "" {
		return fmt.Sprintf("%v for alias %q", e.Err, e.Alias)
	}
	if e.Argument != "" {
		return fmt.Sprintf("%v %q in command %q", e.Err, e.Argument, e.Command)
	}
	if e.Flag != "" {
		return fmt.Sprintf("%v %q in command %q", e.Err, e.Flag, e.Command)
	}
	if e.Command != "" {
		return fmt.Sprintf("%v %q", e.Err, e.Command)
	}
	return e.Err.Error()
}

func (e *CommandRegistrationError) Unwrap() error {
	return e.Err
}

// Registry stores registered commands, context menu actions, component commands, modals, and categories.
type Registry struct {
	chatCommands               map[string]ChatCommand
	userContextMenuCommands    map[string]UserContextMenuCommand
	messageContextMenuCommands map[string]MessageContextMenuCommand
	buttonMatcher              *PatternMatcher[ButtonCommand]
	selectMenuMatcher          *PatternMatcher[SelectMenuCommand]
	modalMatcher               *PatternMatcher[ModalCommand]
	autocompleteHandlers       map[string]AutocompleteHandler
	categories                 map[string]*Category
	commandCategories          map[string]*Category
	mu                         sync.RWMutex
}

// NewRegistry creates a new empty Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		chatCommands:               make(map[string]ChatCommand),
		userContextMenuCommands:    make(map[string]UserContextMenuCommand),
		messageContextMenuCommands: make(map[string]MessageContextMenuCommand),
		buttonMatcher:              NewPatternMatcher[ButtonCommand](),
		selectMenuMatcher:          NewPatternMatcher[SelectMenuCommand](),
		modalMatcher:               NewPatternMatcher[ModalCommand](),
		autocompleteHandlers:       make(map[string]AutocompleteHandler),
		categories:                 make(map[string]*Category),
		commandCategories:          make(map[string]*Category),
	}
}

// RegisterCommand registers a chat / slash command and its aliases.
func (r *Registry) RegisterCommand(cmd ChatCommand) error {
	if err := validateChatCommand(cmd); err != nil {
		return err
	}

	cfg := cmd.Config()
	name := strings.ToLower(strings.TrimSpace(cfg.Name))

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.chatCommands[name]; ok {
		return &CommandRegistrationError{
			Err:     ErrCommandAlreadyExists,
			Command: name,
		}
	}

	seenAliases := make(map[string]bool)
	var cleanAliases []string

	for _, rawAlias := range cfg.Aliases {
		alias := strings.ToLower(strings.TrimSpace(rawAlias))
		if alias == "" || alias == name || seenAliases[alias] {
			continue
		}

		if len(strings.Fields(alias)) > 3 {
			return &CommandRegistrationError{
				Err:     ErrAliasTooLong,
				Command: name,
				Alias:   alias,
			}
		}

		seenAliases[alias] = true

		if existing, ok := r.chatCommands[alias]; ok && existing != cmd {
			return &CommandRegistrationError{
				Err:     ErrAliasAlreadyExists,
				Command: name,
				Alias:   alias,
			}
		}
		cleanAliases = append(cleanAliases, alias)
	}

	r.chatCommands[name] = cmd

	if cfg.Category != nil {
		r.commandCategories[name] = cfg.Category
		cfg.Category.AddChatCommands(cmd)
	}

	for _, alias := range cleanAliases {
		r.chatCommands[alias] = cmd
	}

	return nil
}

// RegisterAlias associates a new alias with an existing registered command.
func (r *Registry) RegisterAlias(alias string, target string) error {
	cleanAlias := strings.ToLower(strings.TrimSpace(alias))
	cleanTarget := strings.ToLower(strings.TrimSpace(target))

	if cleanAlias == "" || cleanTarget == "" {
		return ErrEmptyCommandName
	}

	if len(strings.Fields(cleanAlias)) > 3 {
		return &CommandRegistrationError{
			Err:     ErrAliasTooLong,
			Command: target,
			Alias:   alias,
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	targetCmd, ok := r.chatCommands[cleanTarget]
	if !ok {
		return &CommandRegistrationError{
			Err:     ErrTargetCommandNotFound,
			Command: target,
		}
	}

	if existing, ok := r.chatCommands[cleanAlias]; ok && existing != targetCmd {
		return &CommandRegistrationError{
			Err:     ErrAliasAlreadyExists,
			Command: target,
			Alias:   alias,
		}
	}

	r.chatCommands[cleanAlias] = targetCmd
	return nil
}

// GetCommand retrieves a command by name or alias.
func (r *Registry) GetCommand(name string) (ChatCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.chatCommands[strings.ToLower(name)]
	return cmd, ok
}

// Commands returns a copy of the registered chat commands map.
func (r *Registry) Commands() map[string]ChatCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make(map[string]ChatCommand, len(r.chatCommands))
	for k, v := range r.chatCommands {
		res[k] = v
	}
	return res
}

// RegisterUserContextMenuCommand registers a user context menu command.
func (r *Registry) RegisterUserContextMenuCommand(cmd UserContextMenuCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}

	name := strings.ToLower(strings.TrimSpace(cmd.Config().Name))
	if name == "" {
		return ErrEmptyCommandName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.userContextMenuCommands[name]; ok {
		return &CommandRegistrationError{
			Err:     ErrUserContextMenuCommandAlreadyExists,
			Command: name,
		}
	}

	r.userContextMenuCommands[name] = cmd
	return nil
}

// GetUserContextMenuCommand retrieves a user context menu command by name.
func (r *Registry) GetUserContextMenuCommand(name string) (UserContextMenuCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.userContextMenuCommands[strings.ToLower(name)]
	return cmd, ok
}

// UserContextMenuCommands returns all registered user context menu commands.
func (r *Registry) UserContextMenuCommands() []UserContextMenuCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]UserContextMenuCommand, 0, len(r.userContextMenuCommands))
	for _, cmd := range r.userContextMenuCommands {
		res = append(res, cmd)
	}
	return res
}

// RegisterMessageContextMenuCommand registers a message context menu command.
func (r *Registry) RegisterMessageContextMenuCommand(cmd MessageContextMenuCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}

	name := strings.ToLower(strings.TrimSpace(cmd.Config().Name))
	if name == "" {
		return ErrEmptyCommandName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.messageContextMenuCommands[name]; ok {
		return &CommandRegistrationError{
			Err:     ErrMessageContextMenuCommandAlreadyExists,
			Command: name,
		}
	}

	r.messageContextMenuCommands[name] = cmd
	return nil
}

// GetMessageContextMenuCommand retrieves a message context menu command by name.
func (r *Registry) GetMessageContextMenuCommand(name string) (MessageContextMenuCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.messageContextMenuCommands[strings.ToLower(name)]
	return cmd, ok
}

// MessageContextMenuCommands returns all registered message context menu commands.
func (r *Registry) MessageContextMenuCommands() []MessageContextMenuCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]MessageContextMenuCommand, 0, len(r.messageContextMenuCommands))
	for _, cmd := range r.messageContextMenuCommands {
		res = append(res, cmd)
	}
	return res
}

// RegisterButtonCommand registers a button command supporting HTTP-style path parameters (e.g. 'Hello/{world}').
func (r *Registry) RegisterButtonCommand(cmd ButtonCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}
	customID := strings.TrimSpace(cmd.Config().CustomID)
	if customID == "" {
		return ErrEmptyCustomID
	}

	err := r.buttonMatcher.Insert(customID, cmd)
	if errors.Is(err, ErrPatternAlreadyExists) {
		return &CommandRegistrationError{
			Err:     ErrButtonCommandAlreadyExists,
			Command: customID,
		}
	}
	return err
}

// MatchButtonCommand matches a button command and returns the MatchResult with extracted path parameters.
func (r *Registry) MatchButtonCommand(customID string) (MatchResult[ButtonCommand], bool) {
	return r.buttonMatcher.Match(customID)
}

// GetButtonCommand finds a registered button command for the given customID.
func (r *Registry) GetButtonCommand(customID string) (ButtonCommand, bool) {
	res, ok := r.buttonMatcher.Match(customID)
	if !ok {
		return nil, false
	}
	return res.Value, true
}

// ButtonCommands returns all registered button command patterns and their handlers.
func (r *Registry) ButtonCommands() map[string]ButtonCommand {
	return r.buttonMatcher.All()
}

// RegisterSelectMenuCommand registers a select menu command supporting HTTP-style path parameters (e.g. 'menus/{menu_id}').
func (r *Registry) RegisterSelectMenuCommand(cmd SelectMenuCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}
	customID := strings.TrimSpace(cmd.Config().CustomID)
	if customID == "" {
		return ErrEmptyCustomID
	}

	err := r.selectMenuMatcher.Insert(customID, cmd)
	if errors.Is(err, ErrPatternAlreadyExists) {
		return &CommandRegistrationError{
			Err:     ErrSelectMenuCommandAlreadyExists,
			Command: customID,
		}
	}
	return err
}

// MatchSelectMenuCommand matches a select menu command and returns the MatchResult.
func (r *Registry) MatchSelectMenuCommand(customID string) (MatchResult[SelectMenuCommand], bool) {
	return r.selectMenuMatcher.Match(customID)
}

// GetSelectMenuCommand finds a registered select menu command for the given customID.
func (r *Registry) GetSelectMenuCommand(customID string) (SelectMenuCommand, bool) {
	res, ok := r.selectMenuMatcher.Match(customID)
	if !ok {
		return nil, false
	}
	return res.Value, true
}

// SelectMenuCommands returns all registered select menu command patterns and their handlers.
func (r *Registry) SelectMenuCommands() map[string]SelectMenuCommand {
	return r.selectMenuMatcher.All()
}

// RegisterModalCommand registers a modal submission command supporting HTTP-style path parameters (e.g. 'feedback/{user_id}').
func (r *Registry) RegisterModalCommand(cmd ModalCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}
	customID := strings.TrimSpace(cmd.Config().CustomID)
	if customID == "" {
		return ErrEmptyCustomID
	}

	err := r.modalMatcher.Insert(customID, cmd)
	if errors.Is(err, ErrPatternAlreadyExists) {
		return &CommandRegistrationError{
			Err:     ErrModalCommandAlreadyExists,
			Command: customID,
		}
	}
	return err
}

// MatchModalCommand matches a modal command and returns the MatchResult.
func (r *Registry) MatchModalCommand(customID string) (MatchResult[ModalCommand], bool) {
	return r.modalMatcher.Match(customID)
}

// GetModalCommand finds a registered modal command for the given customID.
func (r *Registry) GetModalCommand(customID string) (ModalCommand, bool) {
	res, ok := r.modalMatcher.Match(customID)
	if !ok {
		return nil, false
	}
	return res.Value, true
}

// ModalCommands returns all registered modal command patterns and their handlers.
func (r *Registry) ModalCommands() map[string]ModalCommand {
	return r.modalMatcher.All()
}

// OnAutocomplete registers a handler for autocomplete option queries.
func (r *Registry) OnAutocomplete(commandName, optionName string, handler AutocompleteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(commandName) + ":" + strings.TrimSpace(optionName))
	r.autocompleteHandlers[key] = handler
}

// GetAutocompleteHandler retrieves an autocomplete handler for a specific command and option.
func (r *Registry) GetAutocompleteHandler(
	commandName, optionName string,
) (AutocompleteHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(commandName) + ":" + strings.TrimSpace(optionName))
	h, ok := r.autocompleteHandlers[key]
	return h, ok
}

// RegisterCategory registers a category, its aliases, all owned chat commands, and nested subcategories.
func (r *Registry) RegisterCategory(cat *Category) error {
	if err := validateCategory(cat); err != nil {
		return err
	}

	name := strings.ToLower(strings.TrimSpace(cat.Name))

	r.mu.Lock()
	if _, exists := r.categories[name]; exists {
		r.mu.Unlock()
		return &CommandRegistrationError{
			Err:     ErrCategoryAlreadyExists,
			Command: cat.Name,
		}
	}

	r.categories[name] = cat
	for _, alias := range cat.Aliases {
		cleanAlias := strings.ToLower(strings.TrimSpace(alias))
		if cleanAlias != "" && cleanAlias != name {
			if _, exists := r.categories[cleanAlias]; !exists {
				r.categories[cleanAlias] = cat
			}
		}
	}
	r.mu.Unlock()

	for _, cmd := range cat.ChatCommands() {
		if err := r.RegisterCommand(cmd); err != nil && !errors.Is(err, ErrCommandAlreadyExists) {
			return err
		}
		r.mu.Lock()
		r.commandCategories[strings.ToLower(strings.TrimSpace(cmd.Config().Name))] = cat
		r.mu.Unlock()
	}

	for _, sub := range cat.Subcategories {
		if sub != nil {
			sub.Parent = cat
			if err := r.RegisterCategory(sub); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCategory retrieves a registered category by name or alias.
func (r *Registry) GetCategory(name string) (*Category, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cat, ok := r.categories[strings.ToLower(strings.TrimSpace(name))]
	return cat, ok
}

// Categories returns all registered top-level categories.
func (r *Registry) Categories() []*Category {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[*Category]bool)
	var top []*Category
	for _, cat := range r.categories {
		if !seen[cat] {
			seen[cat] = true
			if cat.IsTopLevel() {
				top = append(top, cat)
			}
		}
	}
	return top
}

// AllCategories returns all registered categories.
func (r *Registry) AllCategories() []*Category {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[*Category]bool)
	var res []*Category
	for _, cat := range r.categories {
		if !seen[cat] {
			seen[cat] = true
			res = append(res, cat)
		}
	}
	return res
}

// CategoryOf returns the category that owns the given command name, if any.
func (r *Registry) CategoryOf(cmdName string) (*Category, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cat, ok := r.commandCategories[strings.ToLower(strings.TrimSpace(cmdName))]
	return cat, ok
}

func validateChatCommand(cmd ChatCommand) error {
	if cmd == nil {
		return ErrNilCommand
	}

	cfg := cmd.Config()
	name := strings.ToLower(strings.TrimSpace(cfg.Name))
	if name == "" {
		return ErrEmptyCommandName
	}

	if len(strings.Fields(name)) > 3 {
		return &CommandRegistrationError{
			Err:     ErrCommandNameTooLong,
			Command: name,
		}
	}

	if err := validateArgDefinitions(name, cfg.Args); err != nil {
		return err
	}

	if err := validateFlagDefinitions(name, cfg.Flags); err != nil {
		return err
	}

	for _, sub := range cfg.Subcommands {
		if err := validateChatCommand(sub); err != nil {
			return err
		}
	}

	return nil
}

func validateArgDefinitions(cmdName string, args []ArgDef) error {
	seenArgs := make(map[string]bool, len(args))
	seenOptional := false

	for i, arg := range args {
		argName := strings.ToLower(strings.TrimSpace(arg.Name))
		if argName == "" {
			return &CommandRegistrationError{
				Err:      ErrEmptyArgName,
				Command:  cmdName,
				Argument: arg.Name,
			}
		}

		if seenArgs[argName] {
			return &CommandRegistrationError{
				Err:      ErrDuplicateArgName,
				Command:  cmdName,
				Argument: argName,
			}
		}
		seenArgs[argName] = true

		if arg.Required && arg.Default != nil {
			return &CommandRegistrationError{
				Err:      ErrRequiredArgWithDefault,
				Command:  cmdName,
				Argument: argName,
			}
		}

		if arg.Required {
			if seenOptional {
				return &CommandRegistrationError{
					Err:      ErrOptionalArgBeforeRequired,
					Command:  cmdName,
					Argument: argName,
				}
			}
		} else {
			seenOptional = true
		}

		if arg.Endless && i != len(args)-1 {
			return &CommandRegistrationError{
				Err:      ErrEndlessArgNotLast,
				Command:  cmdName,
				Argument: argName,
			}
		}

		if err := validateValueConstraints(
			cmdName,
			argName,
			arg.MinLength,
			arg.MaxLength,
			arg.MinValue,
			arg.MaxValue,
			false,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateFlagDefinitions(cmdName string, flags []FlagDef) error {
	seenFlags := make(map[string]bool, len(flags))
	seenShortFlags := make(map[string]bool, len(flags))

	for _, flag := range flags {
		flagName := strings.ToLower(strings.TrimSpace(flag.Name))
		if flagName == "" {
			return &CommandRegistrationError{
				Err:     ErrEmptyFlagName,
				Command: cmdName,
			}
		}

		if seenFlags[flagName] {
			return &CommandRegistrationError{
				Err:     ErrDuplicateFlagName,
				Command: cmdName,
				Flag:    flagName,
			}
		}
		seenFlags[flagName] = true

		short := strings.ToLower(strings.TrimSpace(flag.Short))
		if short != "" {
			if seenShortFlags[short] {
				return &CommandRegistrationError{
					Err:     ErrDuplicateFlagName,
					Command: cmdName,
					Flag:    short,
				}
			}
			seenShortFlags[short] = true
		}

		if err := validateValueConstraints(
			cmdName,
			flagName,
			flag.MinLength,
			flag.MaxLength,
			flag.MinValue,
			flag.MaxValue,
			true,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateValueConstraints(
	cmdName, itemName string,
	minLen, maxLen *int,
	minVal, maxVal *float64,
	isFlag bool,
) error {
	if minLen != nil && *minLen < 0 {
		return makeConstraintError(ErrInvalidLengthRange, cmdName, itemName, isFlag)
	}
	if minLen != nil && maxLen != nil && *minLen > *maxLen {
		return makeConstraintError(ErrInvalidLengthRange, cmdName, itemName, isFlag)
	}
	if minVal != nil && maxVal != nil && *minVal > *maxVal {
		return makeConstraintError(ErrInvalidValueRange, cmdName, itemName, isFlag)
	}
	return nil
}

func validateCategory(cat *Category) error {
	if cat == nil {
		return ErrNilCategory
	}
	name := strings.ToLower(strings.TrimSpace(cat.Name))
	if name == "" {
		return ErrEmptyCategoryName
	}
	visited := make(map[*Category]bool)
	curr := cat
	for curr != nil {
		if visited[curr] {
			return ErrCategoryCycle
		}
		visited[curr] = true
		curr = curr.Parent
	}
	return nil
}

func makeConstraintError(err error, cmdName, itemName string, isFlag bool) error {
	if isFlag {
		return &CommandRegistrationError{
			Err:     err,
			Command: cmdName,
			Flag:    itemName,
		}
	}
	return &CommandRegistrationError{
		Err:      err,
		Command:  cmdName,
		Argument: itemName,
	}
}
