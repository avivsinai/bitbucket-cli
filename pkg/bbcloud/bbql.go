package bbcloud

import "strings"

var bbqlStringEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func bbqlStringLiteral(value string) string {
	return `"` + bbqlStringEscaper.Replace(value) + `"`
}

func bbqlEquals(field, value string) string {
	return field + " = " + bbqlStringLiteral(value)
}

func bbqlContains(field, value string) string {
	return field + " ~ " + bbqlStringLiteral(value)
}
