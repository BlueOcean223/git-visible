package cmd

import (
	"bytes"
	"testing"

	"git-visible/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAliasAdd_AddNewAlias(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	out, err := executeSetCommand(t, "alias", "add", "Alice", "alice@company.com", "alice@gmail.com")
	require.NoError(t, err)
	assert.Contains(t, out, `alias "Alice" saved`)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Aliases, 1)
	assert.Equal(t, "Alice", cfg.Aliases[0].Name)
	assert.Equal(t, []string{"alice@company.com", "alice@gmail.com"}, cfg.Aliases[0].Emails)
}

func TestSetAliasAdd_UpdateExistingAlias(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "",
		Months: config.DefaultMonths,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"old@company.com"}},
			{Name: "Bob", Emails: []string{"bob@company.com"}},
		},
	})

	_, err := executeSetCommand(t, "alias", "add", "Alice", "alice@new.com", "alice@gmail.com")
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Aliases, 2)
	assert.Equal(t, "Alice", cfg.Aliases[0].Name)
	assert.Equal(t, []string{"alice@new.com", "alice@gmail.com"}, cfg.Aliases[0].Emails)
	assert.Equal(t, "Bob", cfg.Aliases[1].Name)
}

func TestSetAliasAdd_InsufficientArgs_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "alias", "add", "Alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage: git-visible set alias add <name> <email1> [email2...]")
}

func TestSetAliasAdd_CrossGroupDuplicateEmail_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "",
		Months: config.DefaultMonths,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"alice@company.com"}},
		},
	})

	_, err := executeSetCommand(t, "alias", "add", "Bob", "ALICE@company.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `email "ALICE@company.com" already belongs to alias "Alice"`)
}

func TestSetAliasRemove_RemoveExistingAlias(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "",
		Months: config.DefaultMonths,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"alice@company.com"}},
			{Name: "Bob", Emails: []string{"bob@company.com"}},
		},
	})

	out, err := executeSetCommand(t, "alias", "remove", "Alice")
	require.NoError(t, err)
	assert.Contains(t, out, `alias "Alice" removed`)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Aliases, 1)
	assert.Equal(t, "Bob", cfg.Aliases[0].Name)
}

func TestSetAliasRemove_AliasNotFound_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "alias", "remove", "Unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `alias "Unknown" not found`)
}

func TestSetAliasList_WithAliases(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "",
		Months: config.DefaultMonths,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"alice@company.com", "alice@gmail.com"}},
			{Name: "Bob", Emails: []string{"bob@work.com", "bob@personal.com"}},
		},
	})

	out, err := executeSetCommand(t, "alias", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "aliases:\n")
	assert.Contains(t, out, "  Alice: alice@company.com, alice@gmail.com\n")
	assert.Contains(t, out, "  Bob: bob@work.com, bob@personal.com\n")
}

func TestSetAliasList_NoAliases(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	out, err := executeSetCommand(t, "alias", "list")
	require.NoError(t, err)
	assert.Equal(t, "No aliases configured\n", out)
}

func TestSet_NoArgs_ShowsConfigWithAliases(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "your@email.com",
		Months: 6,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"alice@company.com", "alice@gmail.com"}},
			{Name: "Bob", Emails: []string{"bob@work.com", "bob@personal.com"}},
		},
	})

	out, err := executeSetCommand(t)
	require.NoError(t, err)
	assert.Contains(t, out, "email: your@email.com\n")
	assert.Contains(t, out, "months: 6\n")
	assert.Contains(t, out, "aliases:\n")
	assert.Contains(t, out, "  Alice: alice@company.com, alice@gmail.com\n")
	assert.Contains(t, out, "  Bob: bob@work.com, bob@personal.com\n")
}

func TestSet_NoArgs_ShowsNoneWhenNoAliases(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "your@email.com", Months: 6})

	out, err := executeSetCommand(t)
	require.NoError(t, err)
	assert.Contains(t, out, "email: your@email.com\n")
	assert.Contains(t, out, "months: 6\n")
	assert.Contains(t, out, "aliases: (none)\n")
}

func executeSetCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := newSetCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return out.String(), err
}

func TestSetAliasAdd_DuplicateEmails_Deduped(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	out, err := executeSetCommand(t, "alias", "add", "Alice", "alice@company.com", "ALICE@company.com", "alice@gmail.com")
	require.NoError(t, err)
	assert.Contains(t, out, `alias "Alice" saved`)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Aliases, 1)
	// ALICE@company.com 应被去重（大小写不敏感）
	assert.Equal(t, []string{"alice@company.com", "alice@gmail.com"}, cfg.Aliases[0].Emails)
}

func TestSetAliasAdd_InvalidEmailFormat_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "alias", "add", "Alice", "not-an-email")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")
}

func TestSetAliasAdd_EmptyName_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "alias", "add", " ", "alice@company.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias name cannot be empty")
}

func TestSet_SetEmail_HappyPath(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "email", "alice@company.com")
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "alice@company.com", cfg.Email)
}

func TestSet_SetMonths_HappyPath(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "months", "12")
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 12, cfg.Months)
}

func TestSet_SetMonths_InvalidValue_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "months", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "months must be > 0")
}

func TestSet_UnsupportedKey_ReturnsError(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	_, err := executeSetCommand(t, "unknown", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key")
}

func TestSetAliasAdd_CaseInsensitiveUpdate(t *testing.T) {
	withTempHome(t)
	setTestConfig(t, config.Config{
		Email:  "",
		Months: config.DefaultMonths,
		Aliases: []config.Alias{
			{Name: "Alice", Emails: []string{"old@company.com"}},
		},
	})

	_, err := executeSetCommand(t, "alias", "add", "alice", "new@company.com")
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Aliases, 1)
	// name 应更新为新输入的 "alice"
	assert.Equal(t, "alice", cfg.Aliases[0].Name)
	assert.Equal(t, []string{"new@company.com"}, cfg.Aliases[0].Emails)
}
