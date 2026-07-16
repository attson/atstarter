// Package cmdparse 在单行命令字符串与 (command, args) 结构之间转换。
// 存储层用结构化的 command+args;UI 层用单行字符串。
package cmdparse

import (
	"errors"
	"strings"

	"github.com/google/shlex"
)

// ErrEmpty 表示输入为空或仅含空白。
var ErrEmpty = errors.New("cmdparse: empty command")

// Parse 把单行命令拆成可执行文件与参数。
// 使用 shell 词法规则,正确处理引号与空格。
// args 永远非 nil(可能为空切片),便于与 JSON 序列化保持稳定。
func Parse(line string) (command string, args []string, err error) {
	if strings.TrimSpace(line) == "" {
		return "", nil, ErrEmpty
	}
	tokens, err := shlex.Split(line)
	if err != nil {
		return "", nil, err
	}
	if len(tokens) == 0 {
		return "", nil, ErrEmpty
	}
	return tokens[0], tokens[1:], nil
}

// Join 把 command+args 拼回可读的单行字符串,供 UI 回显。
// 注意:这是展示用途,不保证与原始输入逐字节一致(引号可能规范化)。
func Join(command string, args []string) string {
	parts := append([]string{command}, args...)
	return strings.Join(parts, " ")
}
