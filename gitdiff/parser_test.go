package gitdiff_test

import (
	"strings"
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/gitdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse_EmptyInput(t *testing.T) {
	t.Parallel()

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(""))

	require.NoError(t, err)
	assert.Empty(t, diff.Files)
}

func TestParser_Parse_ModifiedFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@ package main
 package main

 func main() {
-	println("hello")
+	println("hello world")
+	println("goodbye")
 }
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	// go-gitdiff strips a/ and b/ prefixes
	assert.Equal(t, "main.go", f.OldPath)
	assert.Equal(t, "main.go", f.NewPath)
	assert.Equal(t, diffview.FileModified, f.Operation)
	assert.False(t, f.IsBinary)

	require.Len(t, f.Hunks, 1)
	h := f.Hunks[0]
	assert.Equal(t, 1, h.OldStart)
	assert.Equal(t, 5, h.OldCount)
	assert.Equal(t, 1, h.NewStart)
	assert.Equal(t, 6, h.NewCount)
	assert.Equal(t, "package main", h.Section)

	// Verify line count: 4 context + 1 deleted + 2 added = 7 lines
	require.Len(t, h.Lines, 7)

	// Context line: "package main"
	assert.Equal(t, diffview.LineContext, h.Lines[0].Type)
	assert.Equal(t, "package main\n", h.Lines[0].Content)
	assert.Equal(t, 1, h.Lines[0].OldLineNum)
	assert.Equal(t, 1, h.Lines[0].NewLineNum)

	// Context line: "" (blank line)
	assert.Equal(t, diffview.LineContext, h.Lines[1].Type)
	assert.Equal(t, 2, h.Lines[1].OldLineNum)
	assert.Equal(t, 2, h.Lines[1].NewLineNum)

	// Context line: "func main() {"
	assert.Equal(t, diffview.LineContext, h.Lines[2].Type)
	assert.Equal(t, 3, h.Lines[2].OldLineNum)
	assert.Equal(t, 3, h.Lines[2].NewLineNum)

	// Deleted line: `	println("hello")`
	assert.Equal(t, diffview.LineDeleted, h.Lines[3].Type)
	assert.Equal(t, 4, h.Lines[3].OldLineNum)
	assert.Equal(t, 0, h.Lines[3].NewLineNum)

	// Added line: `	println("hello world")`
	assert.Equal(t, diffview.LineAdded, h.Lines[4].Type)
	assert.Equal(t, 0, h.Lines[4].OldLineNum)
	assert.Equal(t, 4, h.Lines[4].NewLineNum)

	// Added line: `	println("goodbye")`
	assert.Equal(t, diffview.LineAdded, h.Lines[5].Type)
	assert.Equal(t, 0, h.Lines[5].OldLineNum)
	assert.Equal(t, 5, h.Lines[5].NewLineNum)

	// Context line: "}"
	assert.Equal(t, diffview.LineContext, h.Lines[6].Type)
	assert.Equal(t, 5, h.Lines[6].OldLineNum)
	assert.Equal(t, 6, h.Lines[6].NewLineNum)
}

func TestParser_Parse_AddedFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Empty(t, f.OldPath)
	assert.Equal(t, "new.go", f.NewPath)
	assert.Equal(t, diffview.FileAdded, f.Operation)

	require.Len(t, f.Hunks, 1)
	h := f.Hunks[0]
	assert.Equal(t, 0, h.OldStart)
	assert.Equal(t, 0, h.OldCount)
	assert.Equal(t, 1, h.NewStart)
	assert.Equal(t, 3, h.NewCount)

	// All lines are added
	require.Len(t, h.Lines, 3)
	for i, line := range h.Lines {
		assert.Equal(t, diffview.LineAdded, line.Type)
		assert.Equal(t, 0, line.OldLineNum)
		assert.Equal(t, i+1, line.NewLineNum)
	}
}

func TestParser_Parse_DeletedFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/old.go b/old.go
deleted file mode 100644
index 1234567..0000000
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package main
-
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "old.go", f.OldPath)
	assert.Empty(t, f.NewPath)
	assert.Equal(t, diffview.FileDeleted, f.Operation)

	require.Len(t, f.Hunks, 1)
	h := f.Hunks[0]
	assert.Equal(t, 1, h.OldStart)
	assert.Equal(t, 2, h.OldCount)
	assert.Equal(t, 0, h.NewStart)
	assert.Equal(t, 0, h.NewCount)

	// All lines are deleted
	require.Len(t, h.Lines, 2)
	for i, line := range h.Lines {
		assert.Equal(t, diffview.LineDeleted, line.Type)
		assert.Equal(t, i+1, line.OldLineNum)
		assert.Equal(t, 0, line.NewLineNum)
	}
}

func TestParser_Parse_RenamedFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "old.go", f.OldPath)
	assert.Equal(t, "new.go", f.NewPath)
	assert.Equal(t, diffview.FileRenamed, f.Operation)
	assert.Empty(t, f.Hunks)
}

func TestParser_Parse_CopiedFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/original.go b/copy.go
similarity index 100%
copy from original.go
copy to copy.go
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "original.go", f.OldPath)
	assert.Equal(t, "copy.go", f.NewPath)
	assert.Equal(t, diffview.FileCopied, f.Operation)
	assert.Empty(t, f.Hunks)
}

func TestParser_Parse_BinaryFile(t *testing.T) {
	t.Parallel()

	input := `diff --git a/image.png b/image.png
new file mode 100644
index 0000000..1234567
Binary files /dev/null and b/image.png differ
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "image.png", f.NewPath)
	assert.Equal(t, diffview.FileAdded, f.Operation)
	assert.True(t, f.IsBinary)
	assert.Empty(t, f.Hunks)
}

func TestParser_Parse_MultipleFiles(t *testing.T) {
	t.Parallel()

	input := `diff --git a/a.go b/a.go
index 1234567..abcdefg 100644
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-old
+new
diff --git a/b.go b/b.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/b.go
@@ -0,0 +1 @@
+content
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 2)

	assert.Equal(t, "a.go", diff.Files[0].NewPath)
	assert.Equal(t, diffview.FileModified, diff.Files[0].Operation)

	assert.Equal(t, "b.go", diff.Files[1].NewPath)
	assert.Equal(t, diffview.FileAdded, diff.Files[1].Operation)
}

func TestParser_Parse_NoNewlineAtEOF(t *testing.T) {
	t.Parallel()

	input := `diff --git a/file.txt b/file.txt
index 1234567..abcdefg 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)
	require.Len(t, diff.Files[0].Hunks, 1)

	h := diff.Files[0].Hunks[0]
	require.Len(t, h.Lines, 2)

	assert.True(t, h.Lines[0].NoNewline)
	assert.True(t, h.Lines[1].NoNewline)
}

func TestParser_Parse_MalformedInput(t *testing.T) {
	t.Parallel()

	// go-gitdiff returns error for malformed git headers
	input := `diff --git a/file.go
@@ -1,1 +1,1 @@ incomplete header
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.Error(t, err)
	assert.Nil(t, diff)
}

func TestParser_Parse_ModeChange(t *testing.T) {
	t.Parallel()

	input := `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
`

	p := gitdiff.NewParser()

	diff, err := p.Parse(strings.NewReader(input))

	require.NoError(t, err)
	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "script.sh", f.OldPath)
	assert.Equal(t, "script.sh", f.NewPath)
	assert.Equal(t, diffview.FileModified, f.Operation)
	assert.Empty(t, f.Hunks)
}
