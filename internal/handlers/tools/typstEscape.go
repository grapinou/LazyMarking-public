package tools

import (
	"fmt"
	"strings"
)

func typstStringLiteral(value string) string {
	var builder strings.Builder
	builder.Grow(len(value) + 2)
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&builder, `\u{%x}`, r)
			} else {
				builder.WriteRune(r)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}
