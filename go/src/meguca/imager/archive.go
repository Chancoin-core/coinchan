package imager

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

const (
	mimeZip   = "application/zip"
	mime7Zip  = "application/x-7z-compressed"
	mimeTarGZ = "application/gzip"
	mimeTarXZ = "application/x-xz"
)

// Detect if file is a TAR archive compressed with GZIP
func detectTarGZ(buf []byte, _ io.ReadSeeker) bool {
	if !bytes.HasPrefix(buf, []byte("\x1F\x8B\x08")) {
		return false
	}

	r, err := gzip.NewReader(bytes.NewReader(buf))
	return err == nil && isTar(r)
}

// Read the start of the file and determine, if it is a TAR archive
func isTar(r io.Reader) bool {
	head := make([]byte, 262)
	read, err := r.Read(head)
	if err != nil || read != 262 {
		return false
	}
	return bytes.HasPrefix(head[257:], []byte("ustar"))
}

// Detect if file is a TAR archive compressed with XZ
func detectTarXZ(buf []byte, _ io.ReadSeeker) bool {
	if !bytes.HasPrefix(buf, []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}) {
		return false
	}

	r, err := xz.NewReader(bytes.NewReader(buf))
	return err == nil && isTar(r)
}

// Detect if file is a 7zip archive
func detect7z(buf []byte, _ io.ReadSeeker) bool {
	return bytes.HasPrefix(buf, []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C})
}

// Detect zip archives
func detectZip(data []byte, _ io.ReadSeeker) bool {
	return bytes.HasPrefix(data, []byte("\x50\x4B\x03\x04"))
}
