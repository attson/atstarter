package codexadapter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"rootyard/internal/projectenv"
)

func applyTextEdits(original []byte, edits []projectenv.TextEdit) ([]byte, error) {
	ordered := append([]projectenv.TextEdit(nil), edits...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Start == ordered[j].Start {
			return ordered[i].End < ordered[j].End
		}
		return ordered[i].Start < ordered[j].Start
	})
	previousEnd := 0
	for index, edit := range ordered {
		if edit.Start < 0 || edit.End < edit.Start || edit.End > len(original) {
			return nil, fmt.Errorf("invalid text edit range %d:%d", edit.Start, edit.End)
		}
		if index > 0 && edit.Start < previousEnd {
			return nil, fmt.Errorf("overlapping text edits at byte %d", edit.Start)
		}
		previousEnd = edit.End
	}

	result := append([]byte(nil), original...)
	for index := len(ordered) - 1; index >= 0; index-- {
		edit := ordered[index]
		next := make([]byte, 0, len(result)-(edit.End-edit.Start)+len(edit.Replacement))
		next = append(next, result[:edit.Start]...)
		next = append(next, edit.Replacement...)
		next = append(next, result[edit.End:]...)
		result = next
	}
	return result, nil
}

func tomlBasicString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for len(value) > 0 {
		r, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\b':
			builder.WriteString(`\b`)
		case '\t':
			builder.WriteString(`\t`)
		case '\n':
			builder.WriteString(`\n`)
		case '\f':
			builder.WriteString(`\f`)
		case '\r':
			builder.WriteString(`\r`)
		default:
			if r < 0x20 || r == 0x7f {
				_, _ = fmt.Fprintf(&builder, `\u%04X`, r)
			} else {
				builder.WriteRune(r)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func insertionEdit(doc document, lines []string) projectenv.TextEdit {
	joined := strings.Join(lines, doc.Newline)
	offset := doc.FirstTable
	if offset < len(doc.Data) {
		return projectenv.TextEdit{Start: offset, End: offset, Replacement: joined + doc.Newline}
	}
	if len(doc.Data) == 0 {
		return projectenv.TextEdit{Start: 0, End: 0, Replacement: joined + doc.Newline}
	}
	if bytes.HasSuffix(doc.Data, []byte("\n")) {
		return projectenv.TextEdit{Start: offset, End: offset, Replacement: joined + doc.Newline}
	}
	return projectenv.TextEdit{Start: offset, End: offset, Replacement: doc.Newline + joined}
}

func clearEdit(doc document, current assignment) projectenv.TextEdit {
	replacement := ""
	if current.TrailingComment != "" {
		replacement = current.Indent + current.TrailingComment
		if current.StatementEnd > current.StatementStart && doc.Data[current.StatementEnd-1] == '\n' {
			replacement += doc.Newline
		}
	}
	return projectenv.TextEdit{Start: current.StatementStart, End: current.StatementEnd, Replacement: replacement}
}

func nativeDiff(original []byte, edits []projectenv.TextEdit) []string {
	lines := []string{}
	for _, edit := range edits {
		before := strings.TrimSuffix(string(original[edit.Start:edit.End]), "\n")
		before = strings.TrimSuffix(before, "\r")
		after := strings.TrimSuffix(edit.Replacement, "\n")
		after = strings.TrimSuffix(after, "\r")
		for _, line := range strings.Split(before, "\n") {
			line = strings.TrimSuffix(line, "\r")
			if line != "" {
				lines = append(lines, "-"+line)
			}
		}
		for _, line := range strings.Split(after, "\n") {
			line = strings.TrimSuffix(line, "\r")
			if line != "" {
				lines = append(lines, "+"+line)
			}
		}
	}
	return lines
}
