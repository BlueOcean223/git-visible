package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShow_BranchFlagsMutuallyExclusive(t *testing.T) {
	resetShowFlags()

	c := &cobra.Command{
		Use:   "show",
		Short: "Show contribution heatmap",
		Args:  cobra.NoArgs,
		RunE:  runShow,
	}
	addShowFlags(c)

	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)

	c.SetArgs([]string{"--branch", "main", "--all-branches"})
	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch")
	assert.Contains(t, err.Error(), "all-branches")
}

func resetShowFlags() {
	showEmails = nil
	showMonths = 0
	showSince = ""
	showUntil = ""
	showBranch = ""
	showAllBranch = false
	showFormat = "table"
	showNoLegend = false
	showLegend = false
	showNoSummary = false
	showSummary = false
}
