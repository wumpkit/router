package router

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/wumpkit/wumpkit"
)

// ArgType represents the data type of a positional argument.
type ArgType int

const (
	// ArgTypeString matches any string token.
	ArgTypeString ArgType = iota
	// ArgTypeInt matches standard signed integers.
	ArgTypeInt
	// ArgTypeFloat matches floating point numbers.
	ArgTypeFloat
	// ArgTypeBool matches true/false, yes/no, 1/0, on/off.
	ArgTypeBool
	// ArgTypeUser matches user mentions (<@123>, <@!123>) or raw user snowflake IDs.
	ArgTypeUser
	// ArgTypeMember matches guild member mentions, snowflake IDs, or usernames/nicknames.
	ArgTypeMember
	// ArgTypeRole matches role mentions (<@&123>) or raw role snowflake IDs.
	ArgTypeRole
	// ArgTypeChannel matches channel mentions (<#123>) or raw channel snowflake IDs.
	ArgTypeChannel
	// ArgTypeMentionable matches any user or role mention/ID.
	ArgTypeMentionable
	// ArgTypeAttachment matches uploaded file attachments.
	ArgTypeAttachment
)

// String returns the human-readable name of the argument type.
func (t ArgType) String() string {
	switch t {
	case ArgTypeString:
		return "string"
	case ArgTypeInt:
		return "integer"
	case ArgTypeFloat:
		return "number"
	case ArgTypeBool:
		return "boolean"
	case ArgTypeUser:
		return "user"
	case ArgTypeMember:
		return "member"
	case ArgTypeRole:
		return "role"
	case ArgTypeChannel:
		return "channel"
	case ArgTypeMentionable:
		return "mentionable"
	case ArgTypeAttachment:
		return "attachment"
	default:
		return fmt.Sprintf("ArgType(%d)", int(t))
	}
}

// FlagType represents the data type of a named flag.
type FlagType int

const (
	// FlagTypeString matches any string token value.
	FlagTypeString FlagType = iota
	// FlagTypeInt matches standard signed integers.
	FlagTypeInt
	// FlagTypeFloat matches floating point numbers.
	FlagTypeFloat
	// FlagTypeBool matches boolean flags (presence indicates true or parsed boolean).
	FlagTypeBool
	// FlagTypeUser matches user mentions (<@123>, <@!123>) or raw user snowflake IDs.
	FlagTypeUser
	// FlagTypeMember matches guild member mentions, snowflake IDs, or usernames/nicknames.
	FlagTypeMember
	// FlagTypeRole matches role mentions (<@&123>) or raw role snowflake IDs.
	FlagTypeRole
	// FlagTypeChannel matches channel mentions (<#123>) or raw channel snowflake IDs.
	FlagTypeChannel
	// FlagTypeMentionable matches any user or role mention/ID.
	FlagTypeMentionable
)

// String returns the human-readable name of the flag type.
func (t FlagType) String() string {
	switch t {
	case FlagTypeString:
		return "string"
	case FlagTypeInt:
		return "integer"
	case FlagTypeFloat:
		return "number"
	case FlagTypeBool:
		return "boolean"
	case FlagTypeUser:
		return "user"
	case FlagTypeMember:
		return "member"
	case FlagTypeRole:
		return "role"
	case FlagTypeChannel:
		return "channel"
	case FlagTypeMentionable:
		return "mentionable"
	default:
		return fmt.Sprintf("FlagType(%d)", int(t))
	}
}

// Choice defines a pre-determined valid choice for an argument or flag.
type Choice struct {
	Name  string
	Value any
}

// ArgDef defines a positional command argument definition.
type ArgDef struct {
	Name         string
	Description  string
	Type         ArgType
	Required     bool
	Default      any
	Endless      bool
	Autocomplete bool

	// Constraints
	MinLength *int
	MaxLength *int
	MinValue  *float64
	MaxValue  *float64
	Choices   []Choice
	Parser    func(raw string) (any, error)
}

// FlagDef defines a named command flag definition (e.g. --silent or -s).
type FlagDef struct {
	Name        string
	Short       string
	Description string
	Type        FlagType
	Default     any
	Hidden      bool

	// Constraints
	MinLength *int
	MaxLength *int
	MinValue  *float64
	MaxValue  *float64
	Choices   []Choice
}

// Args provides type-safe retrieval of parsed positional arguments for chat commands.
// Arguments missing from the invocation safely return their corresponding zero value.
type Args struct {
	values map[string]any
}

// NewArgs returns an Args instance backed by the provided values map.
func NewArgs(values map[string]any) Args {
	return Args{values: values}
}

// Has reports whether the specified argument was provided in the command invocation.
func (a Args) Has(name string) bool {
	if a.values == nil {
		return false
	}
	_, ok := a.values[name]
	return ok
}

// Get retrieves the raw untyped value for an argument.
func (a Args) Get(name string) (any, bool) {
	if a.values == nil {
		return nil, false
	}
	v, ok := a.values[name]
	return v, ok
}

// StringE retrieves a string argument value by name or returns an error if conversion fails.
func (a Args) StringE(name string) (string, error) {
	if a.values == nil {
		return "", nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return "", nil
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case json.RawMessage:
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			return s, nil
		}
		return string(val), nil
	case []byte:
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			return s, nil
		}
		return string(val), nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return "", fmt.Errorf("router: argument %q is %T, cannot convert to string", name, v)
	}
}

// String retrieves a string argument value by name, returning "" if missing or on error.
func (a Args) String(name string) string {
	val, _ := a.StringE(name)
	return val
}

// IntE retrieves an integer argument value by name or returns an error if conversion fails.
func (a Args) IntE(name string) (int, error) {
	if a.values == nil {
		return 0, nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case int32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float64:
		return int(val), nil
	case float32:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as int: %w", name, err)
		}
		return i, nil
	case json.RawMessage:
		var i int
		if err := json.Unmarshal(val, &i); err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as int: %w", name, err)
		}
		return i, nil
	case []byte:
		var i int
		if err := json.Unmarshal(val, &i); err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as int: %w", name, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("router: argument %q is %T, cannot convert to int", name, v)
	}
}

// Int retrieves an integer argument value by name, returning 0 if missing or on error.
func (a Args) Int(name string) int {
	val, _ := a.IntE(name)
	return val
}

// FloatE retrieves a float64 argument value by name or returns an error if conversion fails.
func (a Args) FloatE(name string) (float64, error) {
	if a.values == nil {
		return 0, nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as float64: %w", name, err)
		}
		return f, nil
	case json.RawMessage:
		var f float64
		if err := json.Unmarshal(val, &f); err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as float64: %w", name, err)
		}
		return f, nil
	case []byte:
		var f float64
		if err := json.Unmarshal(val, &f); err != nil {
			return 0, fmt.Errorf("router: argument %q cannot be parsed as float64: %w", name, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("router: argument %q is %T, cannot convert to float64", name, v)
	}
}

// Float retrieves a 64-bit floating point argument value by name, returning 0.0 if missing or on error.
func (a Args) Float(name string) float64 {
	val, _ := a.FloatE(name)
	return val
}

// BoolE retrieves a boolean argument value by name or returns an error if conversion fails.
func (a Args) BoolE(name string) (bool, error) {
	if a.values == nil {
		return false, nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return false, nil
	}
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		b, err := parseBoolValue(val)
		if err != nil {
			return false, fmt.Errorf("router: argument %q cannot be parsed as bool: %w", name, err)
		}
		return b, nil
	case json.RawMessage:
		var b bool
		if err := json.Unmarshal(val, &b); err != nil {
			return false, fmt.Errorf("router: argument %q cannot be parsed as bool: %w", name, err)
		}
		return b, nil
	case []byte:
		var b bool
		if err := json.Unmarshal(val, &b); err != nil {
			return false, fmt.Errorf("router: argument %q cannot be parsed as bool: %w", name, err)
		}
		return b, nil
	default:
		return false, fmt.Errorf("router: argument %q is %T, cannot convert to bool", name, v)
	}
}

// Bool retrieves a boolean argument value by name, returning false if missing or on error.
func (a Args) Bool(name string) bool {
	val, _ := a.BoolE(name)
	return val
}

// SnowflakeE retrieves a Discord Snowflake ID argument value by name or returns an error if parsing fails.
func (a Args) SnowflakeE(name string) (wumpkit.Snowflake, error) {
	if a.values == nil {
		return 0, nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case wumpkit.Snowflake:
		return val, nil
	case wumpkit.Attachment:
		return val.ID, nil
	case *wumpkit.Attachment:
		if val != nil {
			return val.ID, nil
		}
		return 0, nil
	case string:
		id, err := parseMentionOrSnowflake(val)
		if err != nil {
			return 0, fmt.Errorf("router: argument %q value %q is not a valid Snowflake: %w", name, val, err)
		}
		return id, nil
	case json.RawMessage:
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			id, err := parseMentionOrSnowflake(s)
			if err != nil {
				return 0, fmt.Errorf("router: argument %q value %q is not a valid Snowflake: %w", name, s, err)
			}
			return id, nil
		}
		var sf wumpkit.Snowflake
		if err := json.Unmarshal(val, &sf); err == nil && sf > 0 {
			return sf, nil
		}
		return 0, fmt.Errorf("router: argument %q value %q is not a valid Snowflake", name, string(val))
	case []byte:
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			id, err := parseMentionOrSnowflake(s)
			if err != nil {
				return 0, fmt.Errorf("router: argument %q value %q is not a valid Snowflake: %w", name, s, err)
			}
			return id, nil
		}
		return 0, fmt.Errorf("router: argument %q value %q is not a valid Snowflake", name, string(val))
	default:
		return 0, fmt.Errorf("router: argument %q is %T, cannot convert to Snowflake", name, v)
	}
}

// Snowflake retrieves a Discord Snowflake ID argument value by name, returning 0 if missing or on error.
func (a Args) Snowflake(name string) wumpkit.Snowflake {
	val, _ := a.SnowflakeE(name)
	return val
}

// AttachmentE retrieves a Discord Attachment argument value by name or returns an error if conversion fails.
func (a Args) AttachmentE(name string) (wumpkit.Attachment, error) {
	if a.values == nil {
		return wumpkit.Attachment{}, nil
	}
	v, ok := a.values[name]
	if !ok || v == nil {
		return wumpkit.Attachment{}, nil
	}
	switch val := v.(type) {
	case wumpkit.Attachment:
		return val, nil
	case *wumpkit.Attachment:
		if val != nil {
			return *val, nil
		}
		return wumpkit.Attachment{}, nil
	default:
		return wumpkit.Attachment{}, fmt.Errorf("router: argument %q is %T, cannot convert to Attachment", name, v)
	}
}

// Attachment retrieves a Discord Attachment argument value by name, returning an empty Attachment if missing or on error.
func (a Args) Attachment(name string) wumpkit.Attachment {
	val, _ := a.AttachmentE(name)
	return val
}

// Flags provides type-safe retrieval of parsed named flags (e.g. --reason="spam", -s) for chat commands.
// Flags missing from the invocation safely return their corresponding zero value.
type Flags struct {
	values map[string]string
}

// NewFlags returns a Flags instance backed by the provided values map.
func NewFlags(values map[string]string) Flags {
	return Flags{values: values}
}

// Has reports whether the specified flag was provided in the command invocation.
// Flag names are case-insensitive.
func (f Flags) Has(name string) bool {
	if f.values == nil {
		return false
	}
	_, ok := f.values[strings.ToLower(name)]
	return ok
}

// Get retrieves the raw string value of a flag along with an existence boolean.
func (f Flags) Get(name string) (string, bool) {
	if f.values == nil {
		return "", false
	}
	val, ok := f.values[strings.ToLower(name)]
	return val, ok
}

// StringE retrieves a string flag value by name.
func (f Flags) StringE(name string) (string, error) {
	if f.values == nil {
		return "", nil
	}
	return f.values[strings.ToLower(name)], nil
}

// String retrieves a string flag value by name, returning "" if not present.
func (f Flags) String(name string) string {
	val, _ := f.StringE(name)
	return val
}

// IntE retrieves an integer flag value by name or returns an error if parsing fails.
func (f Flags) IntE(name string) (int, error) {
	if f.values == nil {
		return 0, nil
	}
	val, ok := f.values[strings.ToLower(name)]
	if !ok {
		return 0, nil
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("router: flag %q value %q cannot be parsed as int: %w", name, val, err)
	}
	return i, nil
}

// Int retrieves an integer flag value by name, returning 0 if missing or on error.
func (f Flags) Int(name string) int {
	val, _ := f.IntE(name)
	return val
}

// FloatE retrieves a floating point flag value by name or returns an error if parsing fails.
func (f Flags) FloatE(name string) (float64, error) {
	if f.values == nil {
		return 0, nil
	}
	val, ok := f.values[strings.ToLower(name)]
	if !ok {
		return 0, nil
	}
	flt, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("router: flag %q value %q cannot be parsed as float64: %w", name, val, err)
	}
	return flt, nil
}

// Float retrieves a floating point flag value by name, returning 0.0 if missing or on error.
func (f Flags) Float(name string) float64 {
	val, _ := f.FloatE(name)
	return val
}

// BoolE retrieves a boolean flag value by name or returns an error if parsing fails.
func (f Flags) BoolE(name string) (bool, error) {
	if f.values == nil {
		return false, nil
	}
	val, ok := f.values[strings.ToLower(name)]
	if !ok {
		return false, nil
	}
	b, err := parseBoolValue(val)
	if err != nil {
		return false, fmt.Errorf("router: flag %q value %q cannot be parsed as bool: %w", name, val, err)
	}
	return b, nil
}

// Bool retrieves a boolean flag value by name, returning false if missing or on error.
func (f Flags) Bool(name string) bool {
	val, _ := f.BoolE(name)
	return val
}

// SnowflakeE retrieves a Discord Snowflake ID flag value by name or returns an error if parsing fails.
func (f Flags) SnowflakeE(name string) (wumpkit.Snowflake, error) {
	if f.values == nil {
		return 0, nil
	}
	val, ok := f.values[strings.ToLower(name)]
	if !ok {
		return 0, nil
	}
	id, err := parseMentionOrSnowflake(val)
	if err != nil {
		return 0, fmt.Errorf("router: flag %q value %q is not a valid Snowflake: %w", name, val, err)
	}
	return id, nil
}

// Snowflake retrieves a Discord Snowflake ID flag value by name, returning 0 if missing or on error.
func (f Flags) Snowflake(name string) wumpkit.Snowflake {
	val, _ := f.SnowflakeE(name)
	return val
}

func parseArgType(val string, t ArgType) (any, error) {
	switch t {
	case ArgTypeString:
		return val, nil
	case ArgTypeInt:
		return strconv.Atoi(val)
	case ArgTypeFloat:
		return strconv.ParseFloat(val, 64)
	case ArgTypeBool:
		return parseBoolValue(val)
	case ArgTypeUser, ArgTypeMember, ArgTypeRole, ArgTypeChannel, ArgTypeMentionable, ArgTypeAttachment:
		return parseMentionOrSnowflake(val)
	default:
		return val, nil
	}
}

func parseFlagType(val string, t FlagType) (any, error) {
	switch t {
	case FlagTypeString:
		return val, nil
	case FlagTypeInt:
		return strconv.Atoi(val)
	case FlagTypeFloat:
		return strconv.ParseFloat(val, 64)
	case FlagTypeBool:
		return parseBoolValue(val)
	case FlagTypeUser, FlagTypeMember, FlagTypeRole, FlagTypeChannel, FlagTypeMentionable:
		return parseMentionOrSnowflake(val)
	default:
		return val, nil
	}
}

func parseBoolValue(val string) (bool, error) {
	switch strings.ToLower(val) {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean value: %q", val)
}

func parseMentionOrSnowflake(val string) (wumpkit.Snowflake, error) {
	var prefixes = [4]string{"@!", "@&", "@", "#"}
	s := strings.Trim(val, "<>")
	for _, p := range prefixes {
		s = strings.TrimPrefix(s, p)
	}
	id, err := wumpkit.ParseSnowflake(s)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid snowflake ID or mention: %q", val)
	}
	return id, nil
}
