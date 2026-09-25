package shellenv

import (
	"errors"
	"strings"
	"testing"
)

const testBody = `[ -f '/h/.config/gu/env.sh' ] && . '/h/.config/gu/env.sh'`

func TestUpsertBlockAppendsToEmptyContent(t *testing.T) {
	got, err := UpsertBlock("", testBody)
	if err != nil {
		t.Fatal(err)
	}
	want := markerBegin + "\n" + testBody + "\n" + markerEnd + "\n"
	if got != want {
		t.Fatalf("UpsertBlock() = %q, want %q", got, want)
	}
}

func TestUpsertBlockPreservesExistingContent(t *testing.T) {
	got, err := UpsertBlock("# user config\nexport EDITOR=vim", testBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "# user config\nexport EDITOR=vim\n") {
		t.Fatalf("existing content was not preserved: %q", got)
	}
	if !HasBlock(got) {
		t.Fatalf("block missing after upsert: %q", got)
	}
}

func TestUpsertBlockIsIdempotent(t *testing.T) {
	once, err := UpsertBlock("# user config\n", testBody)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := UpsertBlock(once, testBody)
	if err != nil {
		t.Fatal(err)
	}
	if twice != once {
		t.Fatalf("second upsert changed content:\n once=%q\ntwice=%q", once, twice)
	}
	if got := strings.Count(twice, markerBegin); got != 1 {
		t.Fatalf("block count = %d, want 1", got)
	}
}

func TestUpsertBlockReplacesExistingBody(t *testing.T) {
	content := "before\n" + markerBegin + "\nold line\n" + markerEnd + "\nafter\n"
	got, err := UpsertBlock(content, testBody)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "old line") {
		t.Fatalf("old body survived: %q", got)
	}
	if !strings.Contains(got, testBody) || !strings.HasPrefix(got, "before\n") || !strings.HasSuffix(got, "after\n") {
		t.Fatalf("upsert result = %q", got)
	}
}

func TestUpsertBlockRejectsUnterminatedBegin(t *testing.T) {
	content := "# hi\n" + markerBegin + "\nexport PATH=x\n"
	_, err := UpsertBlock(content, testBody)
	if !errors.Is(err, ErrMissingBlockEnd) {
		t.Fatalf("err = %v, want ErrMissingBlockEnd", err)
	}
}

func TestUpsertBlockRejectsOrphanEnd(t *testing.T) {
	content := "# hi\n" + markerEnd + "\n"
	_, err := UpsertBlock(content, testBody)
	if !errors.Is(err, ErrCorruptedBlock) {
		t.Fatalf("err = %v, want ErrCorruptedBlock", err)
	}
}

func TestUpsertBlockRejectsDuplicateBlocks(t *testing.T) {
	content := markerBegin + "\na\n" + markerEnd + "\n" + markerBegin + "\nb\n" + markerEnd + "\n"
	_, err := UpsertBlock(content, testBody)
	if !errors.Is(err, ErrCorruptedBlock) {
		t.Fatalf("err = %v, want ErrCorruptedBlock", err)
	}
}

func TestRemoveBlockRestoresOriginal(t *testing.T) {
	original := "# user config\nexport EDITOR=vim\n"
	withBlock, err := UpsertBlock(original, testBody)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RemoveBlock(withBlock)
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("RemoveBlock() = %q, want %q", got, original)
	}
}

func TestRemoveBlockWithoutBlockIsNoop(t *testing.T) {
	content := "# user config\n"
	got, err := RemoveBlock(content)
	if err != nil {
		t.Fatal(err)
	}
	if got != content {
		t.Fatalf("RemoveBlock() = %q, want unchanged %q", got, content)
	}
}

func TestRemoveBlockRejectsCorruptedFile(t *testing.T) {
	_, err := RemoveBlock(markerBegin + "\nunterminated\n")
	if !errors.Is(err, ErrMissingBlockEnd) {
		t.Fatalf("err = %v, want ErrMissingBlockEnd", err)
	}
}

func TestHasBlockRejectsCorruptedFile(t *testing.T) {
	if HasBlock(markerBegin + "\nunterminated\n") {
		t.Fatal("HasBlock() = true for an unterminated block")
	}
}
