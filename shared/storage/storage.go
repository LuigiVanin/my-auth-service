package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("storage is not configured")

// Policy is one kind of file. A new kind is a new Policy, not a new flow.
type Policy struct {
	Prefix        string
	PendingPrefix string
	MaxBytes      int64

	// Content type → extension of the key.
	ContentTypes map[string]string

	GrantTtl     time.Duration
	CacheControl string
}

func (this Policy) Extension(contentType string) (string, bool) {
	extension, ok := this.ContentTypes[contentType]
	return extension, ok
}

// versionLength is the hex length of the random part of a key. A key never
// repeats, which is what lets the object be cached as immutable.
const versionLength = 16

// NewPendingKey is `{PendingPrefix}{owner}/{version}.{extension}`, with owner a
// path such as `users/42`.
func (this Policy) NewPendingKey(owner string, contentType string) (string, error) {
	extension, ok := this.Extension(contentType)

	if !ok {
		return "", fmt.Errorf("content type %q is not accepted", contentType)
	}

	version := make([]byte, versionLength/2)

	if _, err := rand.Read(version); err != nil {
		return "", err
	}

	return this.PendingPrefix + owner + "/" + hex.EncodeToString(version) + "." + extension, nil
}

// OwnsPendingKey matches the whole key against the shape NewPendingKey gives
// that owner, not a prefix: a prefix test lets `users/4` claim `users/42/…`,
// and lets a key already under Prefix through.
func (this Policy) OwnsPendingKey(owner string, key string) bool {
	extensions := []string{}

	for _, extension := range this.ContentTypes {
		extensions = append(extensions, regexp.QuoteMeta(extension))
	}

	slices.Sort(extensions)

	pattern := "^" + regexp.QuoteMeta(this.PendingPrefix+owner+"/") +
		"[0-9a-f]{" + strconv.Itoa(versionLength) + "}" +
		`\.(` + strings.Join(extensions, "|") + ")$"

	return regexp.MustCompile(pattern).MatchString(key)
}

// Promoted is where a pending key lives once confirmed: the same path under
// Prefix.
func (this Policy) Promoted(pendingKey string) string {
	return this.Prefix + strings.TrimPrefix(pendingKey, this.PendingPrefix)
}

// Accepts reports whether a stored object is one this policy would have signed
// for the extension its key carries.
func (this Policy) Accepts(key string, object Object) bool {
	if object.Size < 1 || object.Size > this.MaxBytes {
		return false
	}

	extension, ok := this.Extension(object.ContentType)

	return ok && strings.HasSuffix(key, "."+extension)
}

// UploadGrant is a presigned POST: the client sends every entry of Fields as a
// form field, the file last under `file`, to Url.
type UploadGrant struct {
	Key       string            `json:"key"`
	Url       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
	MaxBytes  int64             `json:"max_bytes"`
}

type Object struct {
	Size        int64
	ContentType string
}

type IUrlResolver interface {
	PublicUrl(key *string) *string
}

type IStorage interface {
	PresignUpload(key string, contentType string, policy Policy) (*UploadGrant, error)

	// Head returns nil, nil when the object does not exist.
	Head(key string) (*Object, error)
	Copy(source string, destination string) error
	Delete(key string) error
}

type UrlResolver struct {
	baseUrl string
}

var _ IUrlResolver = &UrlResolver{}

func NewUrlResolver(baseUrl string) *UrlResolver {
	return &UrlResolver{baseUrl: strings.TrimRight(baseUrl, "/")}
}

func (this *UrlResolver) PublicUrl(key *string) *string {
	if key == nil || *key == "" || this.baseUrl == "" {
		return nil
	}

	url := this.baseUrl + "/" + strings.TrimLeft(*key, "/")

	return &url
}

type Unconfigured struct{}

func IsConfigured(storage IStorage) bool {
	_, unconfigured := storage.(Unconfigured)
	return !unconfigured
}

var _ IStorage = Unconfigured{}

func (Unconfigured) PresignUpload(string, string, Policy) (*UploadGrant, error) {
	return nil, ErrNotConfigured
}

func (Unconfigured) Head(string) (*Object, error) {
	return nil, ErrNotConfigured
}

func (Unconfigured) Copy(string, string) error {
	return ErrNotConfigured
}

func (Unconfigured) Delete(string) error {
	return ErrNotConfigured
}
