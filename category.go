package router

import (
	"fmt"
	"sync"
)

// Category groups related chat commands with optional hierarchy (parent and subcategories).
type Category struct {
	Name          string
	Description   string
	Aliases       []string
	Parent        *Category
	Subcategories []*Category
	commands      []ChatCommand
	mu            sync.RWMutex
}

// NewCategory creates a new Category with the given name and description.
func NewCategory(name, description string) *Category {
	return &Category{
		Name:        name,
		Description: description,
	}
}

// IsTopLevel reports whether this category has no parent.
func (c *Category) IsTopLevel() bool {
	return c.Parent == nil
}

// AddChatCommands appends one or more chat commands to the category.
func (c *Category) AddChatCommands(cmds ...ChatCommand) *Category {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cmd := range cmds {
		if cmd != nil {
			c.commands = append(c.commands, cmd)
		}
	}
	return c
}

// ChatCommands returns a copy of all chat commands directly owned by this category.
func (c *Category) ChatCommands() []ChatCommand {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]ChatCommand, len(c.commands))
	copy(res, c.commands)
	return res
}

// AllChatCommands returns all chat commands owned by this category and its subcategories recursively.
func (c *Category) AllChatCommands() []ChatCommand {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]ChatCommand, len(c.commands))
	copy(res, c.commands)
	for _, sub := range c.Subcategories {
		if sub != nil {
			res = append(res, sub.AllChatCommands()...)
		}
	}
	return res
}

// AddSubcategories appends child categories to this category and sets their Parent to c.
func (c *Category) AddSubcategories(subs ...*Category) *Category {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, sub := range subs {
		if sub != nil {
			sub.Parent = c
			c.Subcategories = append(c.Subcategories, sub)
		}
	}
	return c
}

// Path returns the breadcrumb hierarchy path of the category (e.g. "Admin > Roles").
func (c *Category) Path() string {
	if c.Parent == nil {
		return c.Name
	}
	return fmt.Sprintf("%s > %s", c.Parent.Path(), c.Name)
}
