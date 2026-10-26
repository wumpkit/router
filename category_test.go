package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategory_HierarchyAndCommands(t *testing.T) {
	admin := NewCategory("Admin", "Administration commands")
	admin.Aliases = []string{"administration"}

	assert.True(t, admin.IsTopLevel())
	assert.Equal(t, "Admin", admin.Path())

	cmd1 := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "kick"}}
	cmd2 := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "ban"}}
	admin.AddChatCommands(cmd1, cmd2)

	assert.Len(t, admin.ChatCommands(), 2)

	roles := NewCategory("Roles", "Role management")
	cmd3 := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "role-add"}}
	roles.AddChatCommands(cmd3)

	admin.AddSubcategories(roles)

	assert.False(t, roles.IsTopLevel())
	assert.Equal(t, admin, roles.Parent)
	assert.Equal(t, "Admin > Roles", roles.Path())

	assert.Len(t, admin.ChatCommands(), 2)
	assert.Len(t, admin.AllChatCommands(), 3)
	assert.Len(t, roles.ChatCommands(), 1)
}

func TestCategory_ValidationAndErrors(t *testing.T) {
	r := New()

	assert.ErrorIs(t, r.RegisterCategory(nil), ErrNilCategory)
	assert.ErrorIs(t, r.RegisterCategory(&Category{Name: ""}), ErrEmptyCategoryName)

	catA := NewCategory("CatA", "Category A")
	catB := NewCategory("CatB", "Category B")
	catA.Parent = catB
	catB.Parent = catA

	assert.ErrorIs(t, r.RegisterCategory(catA), ErrCategoryCycle)

	validCat := NewCategory("Mod", "Moderation")
	validCat.Aliases = []string{"moderation"}
	require.NoError(t, r.RegisterCategory(validCat))

	assert.ErrorIs(t, r.RegisterCategory(validCat), ErrCategoryAlreadyExists)
}

func TestCategory_RegistrationAndLookup(t *testing.T) {
	r := New()

	admin := NewCategory("Admin", "Administration commands")
	admin.Aliases = []string{"adm"}

	banCmd := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "ban"}}
	kickCmd := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "kick"}}
	admin.AddChatCommands(banCmd, kickCmd)

	roles := NewCategory("Roles", "Role management")
	roleAddCmd := &mockCommand{ChatCommandConfig: ChatCommandConfig{Name: "role-add"}}
	roles.AddChatCommands(roleAddCmd)

	admin.AddSubcategories(roles)

	require.NoError(t, r.RegisterCategory(admin))

	// Commands registered automatically
	_, foundBan := r.GetCommand("ban")
	assert.True(t, foundBan)
	_, foundRoleAdd := r.GetCommand("role-add")
	assert.True(t, foundRoleAdd)

	// Lookup category by name and alias
	foundAdmin, ok := r.GetCategory("admin")
	assert.True(t, ok)
	assert.Equal(t, "Admin", foundAdmin.Name)

	foundByAlias, ok := r.GetCategory("adm")
	assert.True(t, ok)
	assert.Equal(t, "Admin", foundByAlias.Name)

	foundRoles, ok := r.GetCategory("roles")
	assert.True(t, ok)
	assert.Equal(t, "Roles", foundRoles.Name)

	// Top level vs all categories
	topCats := r.Categories()
	assert.Len(t, topCats, 1)
	assert.Equal(t, "Admin", topCats[0].Name)

	allCats := r.AllCategories()
	assert.Len(t, allCats, 2)

	// CategoryOf lookup
	owningCat, ok := r.CategoryOf("ban")
	assert.True(t, ok)
	assert.Equal(t, "Admin", owningCat.Name)

	roleCat, ok := r.CategoryOf("role-add")
	assert.True(t, ok)
	assert.Equal(t, "Roles", roleCat.Name)
}

func TestCategory_CommandDeclaresCategory(t *testing.T) {
	r := New()

	utilCat := NewCategory("Utility", "Utility commands")
	pingCmd := &mockCommand{
		ChatCommandConfig: ChatCommandConfig{
			Name:     "ping",
			Category: utilCat,
		},
	}

	require.NoError(t, r.RegisterCommand(pingCmd))

	owningCat, ok := r.CategoryOf("ping")
	assert.True(t, ok)
	assert.Equal(t, "Utility", owningCat.Name)
	assert.Len(t, utilCat.ChatCommands(), 1)
}

func TestCategory_MustRegisterPanics(t *testing.T) {
	r := New()
	r.MustRegisterCategory(NewCategory("Fun", "Fun commands"))

	assert.Panics(t, func() {
		r.MustRegisterCategory(NewCategory("Fun", "Duplicate"))
	})
}
