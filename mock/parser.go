// Package mock provides test doubles for diffview interfaces.
package mock

import (
	"io"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.Parser = (*Parser)(nil)

// Parser is a mock implementation of diffview.Parser.
type Parser struct {
	ParseFn func(r io.Reader) (*diffview.Diff, error)
}

func (p *Parser) Parse(r io.Reader) (*diffview.Diff, error) {
	return p.ParseFn(r)
}
