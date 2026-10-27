package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "ban 12345 spam",
			expected: []string{"ban", "12345", "spam"},
		},
		{
			name:     "double quotes",
			input:    `ban @user "reason with spaces" --silent`,
			expected: []string{"ban", "@user", "reason with spaces", "--silent"},
		},
		{
			name:     "single quotes and backticks",
			input:    "echo 'hello world' `code snippet`",
			expected: []string{"echo", "hello world", "code snippet"},
		},
		{
			name:     "escaped quotes",
			input:    `say "he said \"hello\""`,
			expected: []string{"say", `he said "hello"`},
		},
		{
			name:     "flags with equals",
			input:    `kick @target --reason="bad behavior" --delete-days=7`,
			expected: []string{"kick", "@target", "--reason=bad behavior", "--delete-days=7"},
		},
		{
			name:     "unicode and emojis",
			input:    "say 🚀 'hello 世界' --flag=🎉",
			expected: []string{"say", "🚀", "hello 世界", "--flag=🎉"},
		},
		{
			name:     "empty input",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "unclosed quote",
			input:    `say "unclosed string`,
			expected: []string{"say", "unclosed string"},
		},
		{
			name:     "multibyte unicode whitespace",
			input:    "cmd\u00A0arg1\u3000arg2\u2003arg3",
			expected: []string{"cmd", "arg1", "arg2", "arg3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Tokenize(tt.input))
		})
	}
}

func TestParseTokens(t *testing.T) {
	flags := []FlagDef{
		{Name: "reason", Short: "r", Type: FlagTypeString},
		{Name: "silent", Short: "s", Type: FlagTypeBool},
		{Name: "days", Short: "d", Type: FlagTypeInt},
	}

	tests := []struct {
		name               string
		tokens             []string
		expectedPositional []string
		expectedFlags      map[string]string
	}{
		{
			name: "mixed positional and flags",
			tokens: []string{
				"@user",
				"--reason=spamming",
				"-s",
				"-d",
				"7",
				"extra_arg",
			},
			expectedPositional: []string{"@user", "extra_arg"},
			expectedFlags:      map[string]string{"reason": "spamming", "s": "true", "d": "7"},
		},
		{
			name:               "only positional",
			tokens:             []string{"hello", "world"},
			expectedPositional: []string{"hello", "world"},
			expectedFlags:      map[string]string{},
		},
		{
			name:               "only flags",
			tokens:             []string{"--reason=spam", "-s"},
			expectedPositional: []string{},
			expectedFlags:      map[string]string{"reason": "spam", "s": "true"},
		},
		{
			name:               "empty tokens",
			tokens:             []string{},
			expectedPositional: nil,
			expectedFlags:      map[string]string{},
		},
		{
			name:               "bool flag presence means true",
			tokens:             []string{"-s"},
			expectedPositional: []string{},
			expectedFlags:      map[string]string{"s": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseTokens(tt.tokens, flags)
			assert.Equal(t, tt.expectedPositional, parsed.Positional)
			for k, v := range tt.expectedFlags {
				assert.Equal(t, v, parsed.Flags[k], "flag %q", k)
			}
		})
	}
}

func BenchmarkTokenize(b *testing.B) {
	input := `kick @target --reason="bad behavior in general" --delete-days=7 -s extra arguments here`
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = Tokenize(input)
	}
}

func BenchmarkParseTokens(b *testing.B) {
	flags := []FlagDef{
		{Name: "reason", Short: "r", Type: FlagTypeString},
		{Name: "silent", Short: "s", Type: FlagTypeBool},
		{Name: "days", Short: "d", Type: FlagTypeInt},
	}
	tokens := []string{"@user", "--reason=spamming", "-s", "-d", "7", "extra_arg"}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = ParseTokens(tokens, flags)
	}
}
