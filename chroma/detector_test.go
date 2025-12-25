package chroma_test

import (
	"testing"

	"github.com/fwojciec/diffview/chroma"
	"github.com/stretchr/testify/assert"
)

func TestDetector_DetectFromPath(t *testing.T) {
	t.Parallel()

	t.Run("detects Go from .go files", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()
		lang := detector.DetectFromPath("src/main.go")

		assert.Equal(t, "Go", lang)
	})

	t.Run("detects common languages", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()

		cases := []struct {
			path string
			want string
		}{
			{"app.py", "Python"},
			{"component.tsx", "TypeScript"},
			{"lib.rs", "Rust"},
			{"main.js", "JavaScript"},
			{"style.css", "CSS"},
		}

		for _, tc := range cases {
			lang := detector.DetectFromPath(tc.path)
			assert.Equal(t, tc.want, lang, "path: %s", tc.path)
		}
	})

	t.Run("strips b/ prefix from diff paths", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()
		lang := detector.DetectFromPath("b/src/foo.go")

		assert.Equal(t, "Go", lang)
	})

	t.Run("strips a/ prefix from diff paths", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()
		lang := detector.DetectFromPath("a/src/foo.go")

		assert.Equal(t, "Go", lang)
	})

	t.Run("returns empty string for unknown extensions", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()
		lang := detector.DetectFromPath("file.unknownext")

		assert.Empty(t, lang)
	})

	t.Run("handles paths without directories", func(t *testing.T) {
		t.Parallel()

		detector := chroma.NewDetector()
		lang := detector.DetectFromPath("main.go")

		assert.Equal(t, "Go", lang)
	})
}
