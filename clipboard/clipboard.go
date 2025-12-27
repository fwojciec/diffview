// Package clipboard provides clipboard operations via platform-specific commands.
package clipboard

import (
	"os/exec"
	"strings"

	"github.com/fwojciec/diffview"
)

// Ensure PBCopy implements the Clipboard interface.
var _ diffview.Clipboard = (*PBCopy)(nil)

// PBCopy implements Clipboard using macOS pbcopy command.
type PBCopy struct{}

// NewPBCopy returns a new PBCopy clipboard.
func NewPBCopy() *PBCopy {
	return &PBCopy{}
}

// Copy writes content to the system clipboard using pbcopy.
func (p *PBCopy) Copy(content string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
