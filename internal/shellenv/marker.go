package shellenv

import (
	"errors"
	"strings"
)

const (
	markerBegin = "# >>> gu initialize >>>"
	markerEnd   = "# <<< gu initialize <<<"
)

// ErrCorruptedBlock reports a startup file whose gu markers are unbalanced or
// duplicated. gu refuses to edit such a file rather than guess at the intent.
var ErrCorruptedBlock = errors.New("gu 설정 블록이 손상되었습니다")

// ErrMissingBlockEnd reports a begin marker without a matching end marker.
var ErrMissingBlockEnd = errors.New("gu 설정 블록의 끝 마커가 없습니다")

// UpsertBlock inserts body between gu's markers, replacing the existing body
// when a well-formed block is already present. Re-running is therefore
// idempotent instead of appending duplicate blocks.
func UpsertBlock(content, body string) (string, error) {
	start, end, err := blockBounds(content)
	if err != nil {
		return "", err
	}
	block := markerBegin + "\n" + body + "\n" + markerEnd
	if start >= 0 {
		return content[:start] + block + content[end:], nil
	}
	if content == "" {
		return block + "\n", nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + block + "\n", nil
}

// RemoveBlock deletes gu's marked block, restoring the file to its pre-setup
// content without touching anything outside the markers.
func RemoveBlock(content string) (string, error) {
	start, end, err := blockBounds(content)
	if err != nil {
		return "", err
	}
	if start < 0 {
		return content, nil
	}
	before := content[:start]
	after := strings.TrimPrefix(content[end:], "\n")
	return before + after, nil
}

// HasBlock reports whether content holds a well-formed gu block.
func HasBlock(content string) bool {
	start, _, err := blockBounds(content)
	return err == nil && start >= 0
}

// blockBounds locates the byte range of gu's block. It returns (-1, -1, nil)
// when no begin marker exists and an error when the markers are unbalanced.
func blockBounds(content string) (int, int, error) {
	begin := strings.Index(content, markerBegin)
	if begin < 0 {
		if strings.Contains(content, markerEnd) {
			return -1, -1, ErrCorruptedBlock
		}
		return -1, -1, nil
	}
	if begin > 0 && content[begin-1] != '\n' {
		return -1, -1, ErrCorruptedBlock
	}
	if before := content[:begin]; strings.Contains(before, markerEnd) {
		return -1, -1, ErrCorruptedBlock
	}

	rest := content[begin+len(markerBegin):]
	relEnd := strings.Index(rest, markerEnd)
	if relEnd < 0 {
		return -1, -1, ErrMissingBlockEnd
	}
	endStart := begin + len(markerBegin) + relEnd
	if endStart > 0 && content[endStart-1] != '\n' {
		return -1, -1, ErrCorruptedBlock
	}
	end := endStart + len(markerEnd)

	if strings.Contains(content[end:], markerBegin) || strings.Contains(content[end:], markerEnd) {
		return -1, -1, ErrCorruptedBlock
	}
	return begin, end, nil
}
