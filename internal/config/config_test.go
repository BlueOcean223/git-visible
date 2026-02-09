package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeEmail_AliasEmailMapsToPrimary(t *testing.T) {
	cfg := &Config{
		Aliases: []Alias{
			{Name: "Alice", Emails: []string{"alice@company.com", "alice@gmail.com"}},
		},
	}

	assert.Equal(t, "alice@company.com", cfg.NormalizeEmail("alice@gmail.com"))
}

func TestNormalizeEmail_UnmatchedReturnsOriginal(t *testing.T) {
	cfg := &Config{
		Aliases: []Alias{
			{Name: "Alice", Emails: []string{"alice@company.com", "alice@gmail.com"}},
		},
	}

	assert.Equal(t, "unknown@x.com", cfg.NormalizeEmail("unknown@x.com"))
}

func TestNormalizeEmail_CaseInsensitiveAndTrimSpace(t *testing.T) {
	cfg := &Config{
		Aliases: []Alias{
			{Name: "Alice", Emails: []string{"alice@company.com", "alice@gmail.com"}},
		},
	}

	assert.Equal(t, "alice@company.com", cfg.NormalizeEmail("  ALICE@GMAIL.COM  "))
}

func TestNormalizeEmail_NoAliasesReturnsOriginal(t *testing.T) {
	cfg := &Config{}

	assert.Equal(t, "nobody@example.com", cfg.NormalizeEmail("nobody@example.com"))
}
