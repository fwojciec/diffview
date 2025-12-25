package chroma

import (
	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/fwojciec/diffview"
)

// StyleFromPalette returns a function that maps chroma token types to diffview styles
// based on the provided palette colors.
func StyleFromPalette(p diffview.Palette) StyleFunc {
	return func(tt chromalib.TokenType) diffview.Style {
		switch tt {
		// Type keywords (handled separately from other keywords)
		case chromalib.KeywordType:
			return diffview.Style{Foreground: string(p.Type), Bold: true}

		// Keywords
		case chromalib.Keyword, chromalib.KeywordConstant, chromalib.KeywordDeclaration,
			chromalib.KeywordNamespace, chromalib.KeywordPseudo, chromalib.KeywordReserved:
			return diffview.Style{Foreground: string(p.Keyword), Bold: true}

		// Comments
		case chromalib.Comment, chromalib.CommentHashbang, chromalib.CommentMultiline,
			chromalib.CommentPreproc, chromalib.CommentPreprocFile, chromalib.CommentSingle,
			chromalib.CommentSpecial:
			return diffview.Style{Foreground: string(p.Comment)}

		// Strings
		case chromalib.String, chromalib.StringAffix, chromalib.StringBacktick, chromalib.StringChar,
			chromalib.StringDelimiter, chromalib.StringDoc, chromalib.StringDouble,
			chromalib.StringEscape, chromalib.StringHeredoc, chromalib.StringInterpol,
			chromalib.StringOther, chromalib.StringRegex, chromalib.StringSingle,
			chromalib.StringSymbol:
			return diffview.Style{Foreground: string(p.String)}

		// Numbers
		case chromalib.Number, chromalib.NumberBin, chromalib.NumberFloat, chromalib.NumberHex,
			chromalib.NumberInteger, chromalib.NumberIntegerLong, chromalib.NumberOct:
			return diffview.Style{Foreground: string(p.Number)}

		// Operators
		case chromalib.Operator, chromalib.OperatorWord:
			return diffview.Style{Foreground: string(p.Operator)}

		// Function names
		case chromalib.NameFunction, chromalib.NameFunctionMagic:
			return diffview.Style{Foreground: string(p.Function)}

		// Constants
		case chromalib.NameConstant:
			return diffview.Style{Foreground: string(p.Constant)}

		// Punctuation
		case chromalib.Punctuation:
			return diffview.Style{Foreground: string(p.Punctuation)}

		default:
			return diffview.Style{}
		}
	}
}
