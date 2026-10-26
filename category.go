package router

import (
	"fmt"
	"sync"
)

// category groups related chat commands with optional hierarchy
type Category struct {
	Name          string
	Description   string
	Aliases       []string
	Parent        *Category
	Subcategories []*Category
	commands      []ChatCommand
	mu            sync.RWMutex
}

// newcategory creates a new category with name and description
func NewCategory(name, description string) *Category {
	return &Category{
		Name:        name,
		Description: description,
	}
}

// istoplevel reports whether this category has no parent
func (c *Category) IsTopLevel() bool {
	return c.Parent == nil
}

// addChatCommands appends one or more chat commands to the category
func (c *Category) AddChatCommands(cmds ...ChatCommand) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cmd := range cmds {
		if cmd != nil {
			c.commands = append(c.commands, cmd)
		}
	}
}

// chatCommands returns a copy of all chat commands directly owned by this category
func (c *Category) ChatCommands() []ChatCommand {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]ChatCommand, len(c.commands))
	copy(res, c.commands)
	return res
}

// allChatCommands returns all chat commands owned by this category and its subcategories recursively
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

// addSubcategories appends child categories to this category and sets their parent to c
func (c *Category) AddSubcategories(subs ...*Category) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, sub := range subs {
		if sub != nil {
			sub.Parent = c
			c.Subcategories = append(c.Subcategories, sub)
		}
	}
}

// path returns breadcrumb hierarchy path of the category
func (c *Category) Path() string {
	if c.Parent == nil {
		return c.Name
	}
	return fmt.Sprintf("%s > %s", c.Parent.Path(), c.Name)
}
