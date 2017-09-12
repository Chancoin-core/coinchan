package imager

import "github.com/bakape/thumbnailer"

func init() {
	formats := [...]struct {
		mime, ext string
		fn        thumbnailer.MatchFunc
	}{
		{mime7Zip, "7z", detect7z},
		{mimeZip, "zip", detectZip},
		{mimeTarGZ, "tar.gz", detectTarGZ},
		{mimeTarXZ, "tar.xz", detectTarXZ},
		// Has to be last, in case any other formats are pure UTF-8
		{mimeText, ".txt", detectText},
	}

	for _, f := range formats {
		m := thumbnailer.NewFuncMatcher(f.mime, f.ext, f.fn)
		thumbnailer.RegisterMatcher(m)
		thumbnailer.RegisterProcessor(f.mime, noopProcessor)
	}
}

// Does nothing.
// Needed for the thumbnailer to accept these as validly processed.
func noopProcessor(src *thumbnailer.Source, _ thumbnailer.Options) (
	thumbnailer.Thumbnail, error,
) {
	return thumbnailer.Thumbnail{}, nil
}
