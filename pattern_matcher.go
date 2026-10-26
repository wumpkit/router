package router

import (
	"errors"
	"strings"
	"sync"
)

var (
	// ErrPatternAlreadyExists is returned when attempting to register a duplicate component pattern.
	ErrPatternAlreadyExists = errors.New("router: pattern already registered")
	// ErrInvalidPattern is returned when a component pattern has invalid syntax.
	ErrInvalidPattern = errors.New("router: invalid pattern syntax")
)

// MatchResult holds the result of a component pattern match, including extracted path parameters.
type MatchResult[T any] struct {
	Value   T
	Pattern string
	Params  map[string]string
}

// trieNode is an internal node in the Radix Trie for segment-based pattern matching.
type trieNode[T any] struct {
	literalChildren map[string]*trieNode[T]
	paramChild      *trieNode[T]
	isEnd           bool
	value           T
	pattern         string
	paramNames      []string
}

func newTrieNode[T any]() *trieNode[T] {
	return &trieNode[T]{
		literalChildren: make(map[string]*trieNode[T]),
	}
}

// PatternMatcher matches incoming component custom IDs against registered patterns with {param} placeholders
// using an O(K) segment-based Radix Trie with static precedence.
type PatternMatcher[T any] struct {
	root     *trieNode[T]
	exactMap map[string]T
	all      map[string]T
	mu       sync.RWMutex
}

// NewPatternMatcher creates a new empty PatternMatcher.
func NewPatternMatcher[T any]() *PatternMatcher[T] {
	return &PatternMatcher[T]{
		root:     newTrieNode[T](),
		exactMap: make(map[string]T),
		all:      make(map[string]T),
	}
}

// Insert registers a pattern and its associated value.
func (m *PatternMatcher[T]) Insert(pattern string, value T) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPattern := strings.TrimSpace(pattern)
	if cleanPattern == "" {
		return ErrEmptyCustomID
	}

	if _, exists := m.all[cleanPattern]; exists {
		return ErrPatternAlreadyExists
	}

	tokens := tokenizePath(cleanPattern)
	if len(tokens) == 0 {
		return ErrEmptyCustomID
	}

	segments, hasParam, err := parsePatternSegments(tokens)
	if err != nil {
		return err
	}

	var paramNames []string
	curr := m.root
	for _, seg := range segments {
		if seg.isParam {
			paramNames = append(paramNames, seg.paramName)
			if curr.paramChild == nil {
				curr.paramChild = newTrieNode[T]()
			}
			curr = curr.paramChild
		} else {
			key := strings.ToLower(seg.literal)
			if curr.literalChildren == nil {
				curr.literalChildren = make(map[string]*trieNode[T])
			}
			child, ok := curr.literalChildren[key]
			if !ok {
				child = newTrieNode[T]()
				curr.literalChildren[key] = child
			}
			curr = child
		}
	}

	if curr.isEnd {
		return ErrPatternAlreadyExists
	}

	curr.isEnd = true
	curr.value = value
	curr.pattern = cleanPattern
	curr.paramNames = paramNames

	if !hasParam {
		m.exactMap[strings.ToLower(cleanPattern)] = value
	}
	m.all[cleanPattern] = value

	return nil
}

// Match matches an incoming customID against registered patterns and returns the matched value and parameters.
func (m *PatternMatcher[T]) Match(customID string) (MatchResult[T], bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanID := strings.TrimSpace(customID)
	if cleanID == "" {
		return MatchResult[T]{}, false
	}

	// fast exact match check
	if val, ok := m.exactMap[strings.ToLower(cleanID)]; ok {
		return MatchResult[T]{
			Value:   val,
			Pattern: cleanID,
			Params:  nil,
		}, true
	}

	incomingTokens := tokenizePath(cleanID)
	if len(incomingTokens) == 0 {
		return MatchResult[T]{}, false
	}

	// trie traversal with static-literal priority
	return matchTrieNode(m.root, incomingTokens, 0, nil)
}

func matchTrieNode[T any](node *trieNode[T], tokens []string, idx int, paramValues []string) (MatchResult[T], bool) {
	if node == nil {
		return MatchResult[T]{}, false
	}

	if idx == len(tokens) {
		if node.isEnd {
			var params map[string]string
			if len(node.paramNames) > 0 {
				params = make(map[string]string, len(node.paramNames))
				for i, name := range node.paramNames {
					if i < len(paramValues) {
						params[name] = paramValues[i]
					}
				}
			}
			return MatchResult[T]{
				Value:   node.value,
				Pattern: node.pattern,
				Params:  params,
			}, true
		}
		return MatchResult[T]{}, false
	}

	token := tokens[idx]

	// try static literal match first (precedence)
	if node.literalChildren != nil {
		if child, ok := node.literalChildren[strings.ToLower(token)]; ok {
			if res, matched := matchTrieNode(child, tokens, idx+1, paramValues); matched {
				return res, true
			}
		}
	}

	// try wildcard parameter match
	if node.paramChild != nil {
		nextValues := append(paramValues, token)
		if res, matched := matchTrieNode(node.paramChild, tokens, idx+1, nextValues); matched {
			return res, true
		}
	}

	return MatchResult[T]{}, false
}

// All returns a copy of all registered patterns and values.
func (m *PatternMatcher[T]) All() map[string]T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]T, len(m.all))
	for k, v := range m.all {
		res[k] = v
	}
	return res
}

type patternSegment struct {
	isParam   bool
	paramName string
	literal   string
}

func parsePatternSegments(tokens []string) ([]patternSegment, bool, error) {
	var hasParam bool
	segments := make([]patternSegment, 0, len(tokens))

	for _, token := range tokens {
		if strings.HasPrefix(token, "{") && strings.HasSuffix(token, "}") {
			paramName := token[1 : len(token)-1]
			if paramName == "" {
				return nil, false, ErrInvalidPattern
			}
			segments = append(segments, patternSegment{
				isParam:   true,
				paramName: paramName,
			})
			hasParam = true
		} else {
			segments = append(segments, patternSegment{
				isParam: false,
				literal: token,
			})
		}
	}

	return segments, hasParam, nil
}

// tokenizePath splits path into slash and colon segments
func tokenizePath(s string) []string {
	n := len(s)
	if n == 0 {
		return nil
	}
	tokens := make([]string, 0, 4)
	start := 0
	for i := 0; i < n; i++ {
		c := s[i]
		if c == '/' || c == ':' {
			if i > start {
				tokens = append(tokens, s[start:i])
			}
			start = i + 1
		}
	}
	if start < n {
		tokens = append(tokens, s[start:])
	}
	return tokens
}
