package codexadapter

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"

	"rootyard/internal/projectenv"
)

type assignment struct {
	Key             string
	Value           string
	StatementStart  int
	ValueStart      int
	ValueEnd        int
	StatementEnd    int
	TrailingComment string
	Indent          string
}

type document struct {
	Data        []byte
	Newline     string
	FirstTable  int
	Assignments map[string]assignment
}

func parseDocument(data []byte) (document, error) {
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return document{}, invalidTOMLError(err)
	}
	for _, key := range projectenv.SupportedCodexKeys {
		value, ok := root[key]
		if !ok {
			continue
		}
		if _, ok := value.(string); !ok {
			return document{}, &projectenv.Error{Code: projectenv.ErrInvalidCodexConfig, Message: "managed Codex field must be a string", Detail: key}
		}
	}

	parsed := document{
		Data: append([]byte(nil), data...), Newline: "\n", FirstTable: len(data),
		Assignments: map[string]assignment{},
	}
	if bytes.Contains(data, []byte("\r\n")) {
		parsed.Newline = "\r\n"
	}
	managed := make(map[string]struct{}, len(projectenv.SupportedCodexKeys))
	for _, key := range projectenv.SupportedCodexKeys {
		managed[key] = struct{}{}
	}

	parser := unstable.Parser{KeepComments: true}
	parser.Reset(data)
	insideTable := false
	for parser.NextExpression() {
		expression := parser.Expression()
		switch expression.Kind {
		case unstable.Table, unstable.ArrayTable:
			if !insideTable {
				key := expression.Key()
				if key.Next() {
					parsed.FirstTable = lineStart(data, int(key.Node().Raw.Offset))
				}
			}
			insideTable = true
		case unstable.KeyValue:
			if insideTable {
				continue
			}
			keyIterator := expression.Key()
			if !keyIterator.Next() {
				continue
			}
			keyNode := keyIterator.Node()
			key := string(keyNode.Data)
			if keyIterator.Next() {
				continue
			}
			if _, ok := managed[key]; !ok {
				continue
			}
			valueNode := expression.Value()
			valueStart := int(valueNode.Raw.Offset)
			valueEnd := valueStart + int(valueNode.Raw.Length)
			statementStart := lineStart(data, int(keyNode.Raw.Offset))
			statementEnd := lineEnd(data, valueEnd)
			trailing := ""
			if comment := expression.Next(); comment != nil && comment.Kind == unstable.Comment {
				trailing = strings.TrimSuffix(string(parser.Raw(comment.Raw)), "\r")
			}
			value, _ := root[key].(string)
			parsed.Assignments[key] = assignment{
				Key: key, Value: value, StatementStart: statementStart,
				ValueStart: valueStart, ValueEnd: valueEnd, StatementEnd: statementEnd,
				TrailingComment: trailing,
				Indent:          string(data[statementStart:int(keyNode.Raw.Offset)]),
			}
		}
	}
	if err := parser.Error(); err != nil {
		return document{}, invalidTOMLError(err)
	}
	return parsed, nil
}

func invalidTOMLError(err error) error {
	detail := "TOML syntax is invalid"
	var decodeErr *toml.DecodeError
	if errors.As(err, &decodeErr) {
		line, column := decodeErr.Position()
		detail = fmt.Sprintf("line %d, column %d", line, column)
	}
	return &projectenv.Error{Code: projectenv.ErrInvalidCodexConfig, Message: "invalid Codex project configuration", Detail: detail, Cause: err}
}

func lineStart(data []byte, offset int) int {
	if offset <= 0 {
		return 0
	}
	return bytes.LastIndex(data[:offset], []byte("\n")) + 1
}

func lineEnd(data []byte, offset int) int {
	if offset >= len(data) {
		return len(data)
	}
	newline := bytes.IndexByte(data[offset:], '\n')
	if newline < 0 {
		return len(data)
	}
	return offset + newline + 1
}

func normalizedPathEqual(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
