package router

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParsedInput holds the separated positional argument tokens and named flags.
type ParsedInput struct {
	// Raw is the original content string without the command prefix.
	Raw string
	// Positional is the ordered list of positional string tokens.
	Positional []string
	// Flags holds the parsed key-value flags.
	Flags map[string]string
}

func (input *ParsedInput) setFlag(key, val string) {
	if input.Flags == nil {
		input.Flags = make(map[string]string, 4)
	}
	input.Flags[key] = val
}

// Tokenize splits a raw message string into tokens respecting single/double quotes and backticks.
// It uses zero-allocation substring slicing whenever tokens contain no escape characters.
func Tokenize(s string) []string {
	if len(s) == 0 {
		return nil
	}

	var tokens []string
	n := len(s)
	i := 0

	for i < n {
		for i < n {
			r, size := utf8.DecodeRuneInString(s[i:])
			if !unicode.IsSpace(r) {
				break
			}
			i += size
		}
		if i >= n {
			break
		}

		if tokens == nil {
			tokens = make([]string, 0, 8)
		}

		var (
			hasEscapes bool
			hasQuotes  bool
			quoteChar  byte
			start      = i
			end        = i
		)

		if s[i] == '"' || s[i] == '\'' || s[i] == '`' {
			quoteChar = s[i]
			i++
			start = i
			for i < n {
				if s[i] == '\\' && i+1 < n {
					hasEscapes = true
					i += 2
					continue
				}
				if s[i] == quoteChar {
					end = i
					i++
					break
				}
				i++
				end = i
			}
		} else {
			var internalQuote byte
			for i < n {
				r, size := utf8.DecodeRuneInString(s[i:])
				if internalQuote != 0 {
					if s[i] == '\\' && i+1 < n {
						hasEscapes = true
						i += 2
						continue
					}
					if s[i] == internalQuote {
						internalQuote = 0
						hasQuotes = true
						i++
						end = i
						continue
					}
					i += size
					end = i
					continue
				}

				if unicode.IsSpace(r) {
					break
				}

				if s[i] == '\\' && i+1 < n {
					hasEscapes = true
					i += 2
					continue
				}

				if (s[i] == '"' || s[i] == '\'' || s[i] == '`') && i > start && s[i-1] == '=' {
					internalQuote = s[i]
					hasQuotes = true
					i++
					end = i
					continue
				}

				i += size
				end = i
			}
		}

		if !hasEscapes && !hasQuotes {
			tokens = append(tokens, s[start:end])
		} else {
			tokens = append(tokens, unescapeToken(s[start:end], quoteChar))
		}
	}

	return tokens
}

// ParseTokens splits a list of tokens into positional arguments and flags
func ParseTokens(tokens []string, flagDefs []FlagDef) *ParsedInput {
	if len(tokens) == 0 {
		return &ParsedInput{}
	}

	var hasFlags bool
	for _, tok := range tokens {
		if len(tok) > 1 && tok[0] == '-' {
			hasFlags = true
			break
		}
	}

	if !hasFlags {
		return &ParsedInput{
			Positional: tokens,
		}
	}

	input := &ParsedInput{
		Positional: make([]string, 0, len(tokens)),
	}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		var flagContent string
		if strings.HasPrefix(tok, "--") && len(tok) > 2 {
			flagContent = tok[2:]
		} else if strings.HasPrefix(tok, "-") && len(tok) > 1 {
			flagContent = tok[1:]
		}

		if flagContent != "" {
			if key, val, found := strings.Cut(flagContent, "="); found {
				input.setFlag(strings.ToLower(key), val)
			} else {
				key := strings.ToLower(flagContent)
				def, exists := findFlag(key, flagDefs)
				if exists && def.Type != FlagTypeBool && i+1 < len(tokens) &&
					!strings.HasPrefix(tokens[i+1], "-") {
					i++
					input.setFlag(key, tokens[i])
				} else {
					input.setFlag(key, "true")
				}
			}
			continue
		}

		input.Positional = append(input.Positional, tok)
	}

	return input
}

func unescapeToken(s string, quoteChar byte) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			i++
			sb.WriteByte(s[i])
			continue
		}
		if quoteChar == 0 && (c == '"' || c == '\'' || c == '`') {
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

func findFlag(key string, flagDefs []FlagDef) (FlagDef, bool) {
	for _, f := range flagDefs {
		if strings.EqualFold(f.Name, key) || (f.Short != "" && strings.EqualFold(f.Short, key)) {
			return f, true
		}
	}
	return FlagDef{}, false
}
