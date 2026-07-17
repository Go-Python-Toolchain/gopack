// Package bundle implements the self-extracting mechanism at the heart of
// gopack. A payload archive is appended to a small launcher executable, followed
// by a fixed trailer that records where the payload begins and how long it is.
// At runtime the launcher reads its own file, finds the payload, extracts it to
// a content-addressed cache, and runs the entry command described by the
// payload's manifest.
package bundle

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// magic marks the trailer of a gopack executable.
const magic = "GOPACK01"

// trailerSize is the fixed size of the trailer: magic, payload offset, and
// payload length.
const trailerSize = 8 + 8 + 8

// ErrNoBundle is returned when a file has no gopack payload appended.
var ErrNoBundle = errors.New("no gopack payload found")

// Manifest describes how to run a bundled application. Paths in Entry are
// relative to the extraction root unless absolute. Env values may contain the
// token ${ROOT}, which is replaced with the extraction root at launch.
type Manifest struct {
	Name  string            `json:"name,omitempty"`
	Entry []string          `json:"entry"`
	Env   map[string]string `json:"env,omitempty"`
}

// MakeTrailer builds the fixed trailer that records where the payload begins and
// how long it is.
func MakeTrailer(offset, length uint64) []byte {
	tr := make([]byte, trailerSize)
	copy(tr[0:8], magic)
	binary.BigEndian.PutUint64(tr[8:16], offset)
	binary.BigEndian.PutUint64(tr[16:24], length)
	return tr
}

// Append concatenates a launcher, a payload, and a trailer into a single
// executable image.
func Append(launcher, payload []byte) []byte {
	out := make([]byte, 0, len(launcher)+len(payload)+trailerSize)
	out = append(out, launcher...)
	out = append(out, payload...)
	out = append(out, MakeTrailer(uint64(len(launcher)), uint64(len(payload)))...)
	return out
}

// Reader gives access to the payload appended to an executable. Close it when
// done to release the underlying file.
type Reader struct {
	f       *os.File
	Section *io.SectionReader
	Size    int64
}

// Open locates the payload in the executable at path.
func Open(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	size := info.Size()
	if size < trailerSize {
		f.Close()
		return nil, ErrNoBundle
	}

	tr := make([]byte, trailerSize)
	if _, err := f.ReadAt(tr, size-trailerSize); err != nil {
		f.Close()
		return nil, err
	}
	if string(tr[0:8]) != magic {
		f.Close()
		return nil, ErrNoBundle
	}

	offset := binary.BigEndian.Uint64(tr[8:16])
	length := binary.BigEndian.Uint64(tr[16:24])
	if int64(offset)+int64(length)+trailerSize > size {
		f.Close()
		return nil, ErrNoBundle
	}

	return &Reader{
		f:       f,
		Section: io.NewSectionReader(f, int64(offset), int64(length)),
		Size:    int64(length),
	}, nil
}

// Close releases the underlying file.
func (r *Reader) Close() error {
	if r == nil || r.f == nil {
		return nil
	}
	return r.f.Close()
}
