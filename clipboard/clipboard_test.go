package clipboard_test

import (
	"os/exec"
	"testing"

	"github.com/fwojciec/diffview/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPBCopy_Copy(t *testing.T) {
	t.Parallel()

	// Skip if pbcopy is not available (non-macOS systems)
	if _, err := exec.LookPath("pbcopy"); err != nil {
		t.Skip("pbcopy not available, skipping clipboard test")
	}

	cb := clipboard.NewPBCopy()
	testContent := "test clipboard content from diffview"

	err := cb.Copy(testContent)
	require.NoError(t, err)

	// Verify by reading back with pbpaste
	if _, err := exec.LookPath("pbpaste"); err != nil {
		t.Skip("pbpaste not available, cannot verify clipboard content")
	}

	out, err := exec.Command("pbpaste").Output()
	require.NoError(t, err)
	assert.Equal(t, testContent, string(out))
}
