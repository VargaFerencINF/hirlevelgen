package hirlevel

import (
	"encoding/json"
	"strings"
)

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

// lf a Windowson esetleg CRLF-re alakított tesztfájlokat egységesíti.
func lf(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
