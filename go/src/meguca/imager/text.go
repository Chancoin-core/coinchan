package imager

import (
	"io"
	"unicode/utf8"
)

const mimeText = "text/plain"

// Detect any arbitrary text-like file
func detectText(buf []byte, _ io.ReadSeeker) bool {
	return utf8.Valid(buf)
}
