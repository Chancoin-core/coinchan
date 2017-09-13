// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http"
	"strconv"
	"time"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/thumbnailer"
)

var (
	// Map of MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":      common.JPEG,
		"image/png":       common.PNG,
		"image/gif":       common.GIF,
		"application/pdf": common.PDF,
		"video/webm":      common.WEBM,
		"application/ogg": common.OGG,
		"video/mp4":       common.MP4,
		"audio/mpeg":      common.MP3,
		mime7Zip:          common.SevenZip,
		mimeTarGZ:         common.TGZ,
		mimeTarXZ:         common.TXZ,
		mimeZip:           common.ZIP,
		"audio/x-flac":    common.FLAC,
		mimeText:          common.TXT,
	}

	// MIME types from thumbnailer to accept
	allowedMimeTypes = map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/gif":       true,
		"application/pdf": true,
		"video/webm":      true,
		"application/ogg": true,
		"video/mp4":       true,
		"audio/mpeg":      true,
		mimeZip:           true,
		mime7Zip:          true,
		mimeTarGZ:         true,
		mimeTarXZ:         true,
		"audio/x-flac":    true,
		mimeText:          true,
	}

	errTooLarge = errors.New("file too large")
	isTest      bool
)

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	r.Body = http.MaxBytesReader(w, r.Body, int64(config.Get().MaxSize<<20))

	code, id, err := ParseUpload(r)
	if err != nil {
		LogError(w, r, code, err)
	}
	w.Write([]byte(id))
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends and SHA1 hash
// of the file it wants to upload. The server looks up, if such a file is
// thumbnailed. If yes, generates and sends a new image allocation token to
// the client.
func UploadImageHash(w http.ResponseWriter, req *http.Request) {
	buf, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, 40))
	if err != nil {
		LogError(w, req, 500, err)
		return
	}
	hash := string(buf)

	_, err = db.GetImage(hash)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return
	default:
		LogError(w, req, 500, err)
		return
	}

	token, err := db.NewImageToken(hash)
	if err != nil {
		LogError(w, req, 500, err)
	}
	w.Write([]byte(token))
}

// LogError send the client file upload errors and logs them server-side
func LogError(w http.ResponseWriter, r *http.Request, code int, err error) {
	text := err.Error()
	http.Error(w, text, code)
	if !isTest {
		ip, err := auth.GetIP(r)
		if err != nil {
			ip = "invalid IP"
		}
		log.Printf("upload error: %s: %s\n", ip, text)
	}
}

// ParseUpload parses the upload form. Separate function for cleaner error
// handling and reusability. Returns the HTTP status code of the response and an
// error, if any.
func ParseUpload(req *http.Request) (code int, token string, err error) {
	err = parseUploadForm(req)
	if err != nil {
		code = 400
		return
	}

	file, _, err := req.FormFile("image")
	if err != nil {
		code = 400
		return
	}
	defer file.Close()

	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		code = 500
		return
	}
	sum := hash.Sum(nil)
	SHA1 := hex.EncodeToString(sum[:])

	img, err := db.GetImage(SHA1)
	switch err {
	case nil: // Already have a thumbnail
		return newImageToken(SHA1)
	case sql.ErrNoRows:
		img.SHA1 = SHA1
		return newThumbnail(file, img)
	default:
		return 500, "", err
	}
}

func newImageToken(SHA1 string) (int, string, error) {
	token, err := db.NewImageToken(SHA1)
	code := 200
	if err != nil {
		code = 500
	}
	return code, token, err
}

// Parse and validate the form of the upload request
func parseUploadForm(req *http.Request) error {
	length, err := strconv.ParseUint(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}
	if length > uint64(config.Get().MaxSize<<20) {
		return errTooLarge
	}
	return req.ParseMultipartForm(512) // Sniff size of thumbnailer
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client
func newThumbnail(rs io.ReadSeeker, img common.ImageCommon) (
	int, string, error,
) {
	conf := config.Get()
	src, thumb, err := processFile(rs, &img, thumbnailer.Options{
		JPEGQuality: conf.JPEGQuality,
		MaxSourceDims: thumbnailer.Dims{
			Width:  uint64(conf.MaxWidth),
			Height: uint64(conf.MaxHeight),
		},
		ThumbDims: thumbnailer.Dims{
			Width:  150,
			Height: 150,
		},
		AcceptedMimeTypes: allowedMimeTypes,
	})
	switch err {
	case nil:
	case thumbnailer.ErrUnsupportedMIME:
		return 400, "", err
	default:
		return 500, "", err
	}
	defer thumbnailer.PutBytes(src)
	defer thumbnailer.PutBytes(thumb)

	if err := db.AllocateImage(src, thumb, img); err != nil {
		return 500, "", err
	}
	return newImageToken(img.SHA1)
}

// Separate function for easier testability
func processFile(
	rs io.ReadSeeker,
	img *common.ImageCommon,
	opts thumbnailer.Options,
) (
	srcData []byte,
	thumbData []byte,
	err error,
) {
	src, thumb, err := thumbnailer.Process(rs, opts)
	switch err {
	case nil:
	case thumbnailer.ErrNoThumb:
		err = nil
	default:
		return
	}
	thumbData = thumb.Data

	// Read everything into memory for further processing
	srcBuf := thumbnailer.GetBuffer()
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}
	_, err = srcBuf.ReadFrom(rs)
	if err != nil {
		return
	}
	srcData = srcBuf.Bytes()

	img.FileType = mimeTypes[src.Mime]
	if img.FileType == common.PNG {
		img.APNG = apngdetector.Detect(srcData)
	}
	if thumb.Data == nil {
		img.ThumbType = common.NoFile
	} else if thumb.IsPNG {
		img.ThumbType = common.PNG
	}

	img.Audio = src.HasAudio
	img.Video = src.HasVideo
	img.Length = uint32(src.Length / time.Second)
	img.Size = len(srcData)
	img.Artist = src.Artist
	img.Title = src.Title
	img.Dims = [4]uint16{
		uint16(src.Width),
		uint16(src.Height),
		uint16(thumb.Width),
		uint16(thumb.Height),
	}

	sum := md5.Sum(srcData)
	img.MD5 = base64.RawURLEncoding.EncodeToString(sum[:])

	return
}
