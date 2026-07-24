// Package bundle implements the self-extracting mechanism at the heart of
// gopack. A payload archive is appended to a small launcher executable, followed
// by a fixed trailer that records where the payload begins, how long it is, and
// its content key. At runtime the launcher reads its own file, finds the
// payload, extracts it to a cache directory named by the content key, and runs
// the entry command described by the payload's manifest. Recording the key in
// the trailer means a warm start reads a few bytes rather than rehashing the
// whole payload.
package bundle

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

// The trailer at the end of a gopack executable records where the payload is.
// There are two formats.
//
// v1 (GOPACK01) is magic, payload offset, and payload length. Reading the
// payload's content key from a v1 bundle means hashing the whole payload, and
// the launcher did that on every start, warm or cold.
//
// v2 (GOPACK02) adds the payload's content key, computed once at build time.
// The launcher reads it in a few bytes instead of rehashing hundreds of
// megabytes on every warm start. Bundles built before v2 still carry a v1
// trailer, so the launcher reads both.
const (
	magicV1 = "GOPACK01"
	magicV2 = "GOPACK02"

	// KeyLen is the length of the hex content key stored in a v2 trailer.
	KeyLen = 16

	trailerSizeV1 = 8 + 8 + 8
	trailerSizeV2 = 8 + 8 + 8 + KeyLen
)

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

// MakeTrailer builds a v2 trailer recording where the payload begins, how long
// it is, and its content key. The key must be exactly KeyLen characters.
func MakeTrailer(offset, length uint64, key string) []byte {
	tr := make([]byte, trailerSizeV2)
	copy(tr[0:8], magicV2)
	binary.BigEndian.PutUint64(tr[8:16], offset)
	binary.BigEndian.PutUint64(tr[16:24], length)
	copy(tr[24:24+KeyLen], padKey(key))
	return tr
}

// padKey normalizes a key to exactly KeyLen bytes, truncating or padding so the
// trailer is always fixed width.
func padKey(key string) []byte {
	b := make([]byte, KeyLen)
	copy(b, key)
	return b
}

// Append concatenates a launcher and a payload into a single executable image,
// writing a v2 trailer with the payload's content key.
func Append(launcher, payload []byte) []byte {
	key := ContentKey(payload)
	out := make([]byte, 0, len(launcher)+len(payload)+trailerSizeV2)
	out = append(out, launcher...)
	out = append(out, payload...)
	out = append(out, MakeTrailer(uint64(len(launcher)), uint64(len(payload)), key)...)
	return out
}

// Reader gives access to the payload appended to an executable. Close it when
// done to release the underlying file.
type Reader struct {
	f       *os.File
	Section *io.SectionReader
	Size    int64
	// Key is the payload's content key when the trailer recorded one (v2), or
	// empty for a v1 bundle, where it must be computed from the payload.
	Key string
}

// Open locates the payload in the executable at path, reading either trailer
// format.
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

	offset, length, key, trailerSize, err := readTrailer(f, size)
	if err != nil {
		f.Close()
		return nil, err
	}
	if int64(offset)+int64(length)+int64(trailerSize) > size {
		f.Close()
		return nil, ErrNoBundle
	}

	return &Reader{
		f:       f,
		Section: io.NewSectionReader(f, int64(offset), int64(length)),
		Size:    int64(length),
		Key:     key,
	}, nil
}

// readTrailer reads the trailer at the end of a bundle, trying the v2 format
// first and falling back to v1. It returns the payload offset and length, the
// content key (empty for v1), and the size of the trailer that was read.
func readTrailer(f *os.File, size int64) (offset, length uint64, key string, trailerSize int, err error) {
	if size >= trailerSizeV2 {
		tr := make([]byte, trailerSizeV2)
		if _, err := f.ReadAt(tr, size-trailerSizeV2); err != nil {
			return 0, 0, "", 0, err
		}
		if string(tr[0:8]) == magicV2 {
			offset = binary.BigEndian.Uint64(tr[8:16])
			length = binary.BigEndian.Uint64(tr[16:24])
			key = strings.TrimRight(string(tr[24:24+KeyLen]), "\x00")
			return offset, length, key, trailerSizeV2, nil
		}
	}
	if size >= trailerSizeV1 {
		tr := make([]byte, trailerSizeV1)
		if _, err := f.ReadAt(tr, size-trailerSizeV1); err != nil {
			return 0, 0, "", 0, err
		}
		if string(tr[0:8]) == magicV1 {
			offset = binary.BigEndian.Uint64(tr[8:16])
			length = binary.BigEndian.Uint64(tr[16:24])
			return offset, length, "", trailerSizeV1, nil
		}
	}
	return 0, 0, "", 0, ErrNoBundle
}

// Close releases the underlying file.
func (r *Reader) Close() error {
	if r == nil || r.f == nil {
		return nil
	}
	return r.f.Close()
}

// keyPrefixLen is how many hex characters of the payload digest form a content
// key. It matches KeyLen, so a key computed here fits a v2 trailer exactly.
const keyPrefixLen = KeyLen

// ContentKey is the content key of a payload: a prefix of its sha256, in hex. It
// is what names a bundle's extraction directory, so identical payloads share a
// cache and different ones cannot collide.
func ContentKey(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])[:keyPrefixLen]
}

// ContentKeyOf computes the content key by streaming a reader, for a payload too
// large to hold in memory.
func ContentKeyOf(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:keyPrefixLen], nil
}

// Key returns the payload's content key, using the value the trailer recorded
// when present (v2) and otherwise computing it from the payload (v1). It is the
// launcher's fast path: a v2 bundle answers this without reading the payload.
func (r *Reader) key() (string, error) {
	if r.Key != "" {
		return r.Key, nil
	}
	if _, err := r.Section.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return ContentKeyOf(r.Section)
}
