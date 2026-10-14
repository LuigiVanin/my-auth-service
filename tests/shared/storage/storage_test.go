package storage_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	avatar "auth_service/app/modules/avatar/services"
	"auth_service/infra/config"
	"auth_service/shared/storage"
	"auth_service/shared/storage/adapters"

	"github.com/stretchr/testify/assert"
)

var policy = avatar.AvatarPolicy(config.AvatarConfig{})

func TestAPendingKeyHasTheShapeItIsMatchedAgainst(t *testing.T) {
	key, err := policy.NewPendingKey("users/42", "image/webp")

	assert.NoError(t, err)
	assert.Regexp(t, `^pending/avatar/users/42/[0-9a-f]{16}\.webp$`, key)
	assert.True(t, policy.OwnsPendingKey("users/42", key))
}

func TestAPendingKeyNeverRepeats(t *testing.T) {
	first, _ := policy.NewPendingKey("users/42", "image/png")
	second, _ := policy.NewPendingKey("users/42", "image/png")

	assert.NotEqual(t, first, second)
}

func TestAnUnacceptedTypeHasNoKey(t *testing.T) {
	_, err := policy.NewPendingKey("users/42", "image/svg+xml")

	assert.Error(t, err)
}

// RC-3: the key arrives from the client, so it is matched whole against what the
// owner of the route could have been granted.
func TestOnlyTheOwnersOwnPendingKeyIsAccepted(t *testing.T) {
	version := "9f1c3a7e2b4d6081"

	cases := map[string]struct {
		key      string
		accepted bool
	}{
		"its own upload":                    {"pending/avatar/users/42/" + version + ".webp", true},
		"every accepted extension":          {"pending/avatar/users/42/" + version + ".jpg", true},
		"another user":                      {"pending/avatar/users/57/" + version + ".webp", false},
		"a shorter id that prefixes it":     {"pending/avatar/users/4/" + version + ".webp", false},
		"a longer id it prefixes":           {"pending/avatar/users/420/" + version + ".webp", false},
		"already published":                 {"avatar/users/42/" + version + ".webp", false},
		"another subject":                   {"pending/avatar/organizations/42/" + version + ".webp", false},
		"climbing out of the prefix":        {"pending/avatar/users/42/../57/" + version + ".webp", false},
		"a version of another shape":        {"pending/avatar/users/42/NOT-HEX-AT-ALL!.webp", false},
		"a version of another length":       {"pending/avatar/users/42/9f1c.webp", false},
		"an extension that is not accepted": {"pending/avatar/users/42/" + version + ".svg", false},
		"a nested path":                     {"pending/avatar/users/42/x/" + version + ".webp", false},
		"empty":                             {"", false},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.accepted, policy.OwnsPendingKey("users/42", test.key))
		})
	}
}

func TestPromotionKeepsThePathUnderThePublicPrefix(t *testing.T) {
	assert.Equal(
		t,
		"avatar/apps/a1/9f1c3a7e2b4d6081.png",
		policy.Promoted("pending/avatar/apps/a1/9f1c3a7e2b4d6081.png"),
	)
}

func TestAStoredObjectIsAcceptedOnlyWithinWhatWasSigned(t *testing.T) {
	key := "pending/avatar/users/42/9f1c3a7e2b4d6081.webp"

	assert.True(t, policy.Accepts(key, storage.Object{Size: 1, ContentType: "image/webp"}))
	assert.True(t, policy.Accepts(key, storage.Object{Size: policy.MaxBytes, ContentType: "image/webp"}))

	assert.False(t, policy.Accepts(key, storage.Object{Size: 0, ContentType: "image/webp"}))
	assert.False(t, policy.Accepts(key, storage.Object{Size: policy.MaxBytes + 1, ContentType: "image/webp"}))
	assert.False(t, policy.Accepts(key, storage.Object{Size: 10, ContentType: "image/png"}), "type differs from the extension")
	assert.False(t, policy.Accepts(key, storage.Object{Size: 10, ContentType: "text/html"}))
}

func TestThePolicyDefaultsAndOverrides(t *testing.T) {
	assert.Equal(t, int64(2*1024*1024), policy.MaxBytes)
	assert.Equal(t, 5*time.Minute, policy.GrantTtl)

	custom := avatar.AvatarPolicy(config.AvatarConfig{MaxBytes: 1024, GrantTtlSeconds: 60})

	assert.Equal(t, int64(1024), custom.MaxBytes)
	assert.Equal(t, time.Minute, custom.GrantTtl)
}

func TestThePublicUrlIsDerivedFromTheKey(t *testing.T) {
	resolver := storage.NewUrlResolver("https://d123.cloudfront.net/")
	key := "avatar/users/42/9f1c3a7e2b4d6081.webp"
	empty := ""

	assert.Equal(t, "https://d123.cloudfront.net/avatar/users/42/9f1c3a7e2b4d6081.webp", *resolver.PublicUrl(&key))
	assert.Nil(t, resolver.PublicUrl(nil))
	assert.Nil(t, resolver.PublicUrl(&empty))
	assert.Nil(t, storage.NewUrlResolver("").PublicUrl(&key), "no base url means no url, not a relative one")
}

func TestOnlyTheUnconfiguredStorageReportsItself(t *testing.T) {
	assert.False(t, storage.IsConfigured(storage.Unconfigured{}))
	assert.True(t, storage.IsConfigured(adapters.NewS3Adapter(adapters.S3Options{Region: "us-east-1"})))
}

// The signature is computed locally, so the policy document the bucket will
// enforce can be read back without credentials or network.
func TestThePresignedPostPinsEveryCondition(t *testing.T) {
	adapter := adapters.NewS3Adapter(adapters.S3Options{
		Bucket:          "bucket-test",
		Region:          "us-east-1",
		AccessKeyId:     "AKIAEXAMPLE",
		SecretAccessKey: "secret",
	})

	key := "pending/avatar/users/42/9f1c3a7e2b4d6081.webp"

	grant, err := adapter.PresignUpload(key, "image/webp", policy)

	assert.NoError(t, err)
	assert.Equal(t, key, grant.Key)
	assert.Contains(t, grant.Url, "bucket-test")
	assert.Equal(t, policy.MaxBytes, grant.MaxBytes)
	assert.WithinDuration(t, time.Now().Add(policy.GrantTtl), grant.ExpiresAt, 5*time.Second)

	assert.Equal(t, key, grant.Fields["key"])
	assert.Equal(t, "image/webp", grant.Fields["Content-Type"])
	assert.Equal(t, policy.CacheControl, grant.Fields["Cache-Control"])
	assert.NotEmpty(t, grant.Fields["X-Amz-Signature"])

	encoded, err := base64.StdEncoding.DecodeString(grant.Fields["policy"])
	assert.NoError(t, err)

	var document struct {
		Expiration string `json:"expiration"`
		Conditions []any  `json:"conditions"`
	}

	assert.NoError(t, json.Unmarshal(encoded, &document))

	conditions, _ := json.Marshal(document.Conditions)

	assert.Contains(t, string(conditions), `{"key":"`+key+`"}`)
	assert.Contains(t, string(conditions), `{"Content-Type":"image/webp"}`)
	assert.Contains(t, string(conditions), `{"Cache-Control":"`+policy.CacheControl+`"}`)
	assert.Contains(t, string(conditions), `["content-length-range",1,2097152]`)
	assert.Contains(t, string(conditions), `{"bucket":"bucket-test"}`)

	expiration, err := time.Parse(time.RFC3339, strings.Replace(document.Expiration, ".000Z", "Z", 1))
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(policy.GrantTtl), expiration, 5*time.Second)
}
