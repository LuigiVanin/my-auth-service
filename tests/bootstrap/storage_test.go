package bootstrap_test

import (
	"testing"

	"auth_service/infra/bootstrap"
	"auth_service/infra/config"
	"auth_service/shared/storage"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// The shape of env/.env.example.yaml.
const storageYaml = `
storage:
  provider: AWS-S3
  public_base_url: https://d123.cloudfront.net
  aws:
    s3_bucket_name: bucket-test
    region: us-east-1
    access_key_id: AKIAEXAMPLE
    secret_access_key: secret
avatar:
  max_bytes: 1024
  grant_ttl_seconds: 60
`

func TestTheStorageBlockIsReadFromYaml(t *testing.T) {
	var cfg config.Config

	assert.NoError(t, yaml.Unmarshal([]byte(storageYaml), &cfg))

	assert.Equal(t, config.StorageConfig{
		Provider:      "AWS-S3",
		PublicBaseUrl: "https://d123.cloudfront.net",
		Aws: config.StorageAwsConfig{
			S3BucketName:    "bucket-test",
			Region:          "us-east-1",
			AccessKeyId:     "AKIAEXAMPLE",
			SecretAccessKey: "secret",
		},
	}, cfg.Storage)
	assert.Equal(t, config.AvatarConfig{MaxBytes: 1024, GrantTtlSeconds: 60}, cfg.Avatar)
}

// The shape of env/.env.example.
func TestTheStorageBlockIsReadFromTheEnvironment(t *testing.T) {
	t.Setenv("DB_NAME", "auth")
	t.Setenv("DB_PASSWORD", "password")
	t.Setenv("STORAGE_PROVIDER", "AWS-S3")
	t.Setenv("STORAGE_PUBLIC_BASE_URL", "https://d123.cloudfront.net")
	t.Setenv("STORAGE_AWS_S3_BUCKET_NAME", "bucket-test")
	t.Setenv("STORAGE_AWS_REGION", "us-east-1")
	t.Setenv("STORAGE_AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("STORAGE_AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AVATAR_MAX_BYTES", "not a number")

	cfg, err := config.NewConfigFromSysEnv("test")

	assert.NoError(t, err)
	assert.Equal(t, "AWS-S3", cfg.Storage.Provider)
	assert.Equal(t, "https://d123.cloudfront.net", cfg.Storage.PublicBaseUrl)
	assert.Equal(t, "bucket-test", cfg.Storage.Aws.S3BucketName)
	assert.Equal(t, "us-east-1", cfg.Storage.Aws.Region)
	assert.Equal(t, "AKIAEXAMPLE", cfg.Storage.Aws.AccessKeyId)
	assert.Equal(t, "secret", cfg.Storage.Aws.SecretAccessKey)
	assert.Zero(t, cfg.Avatar.MaxBytes, "unparsable falls back to the default")
}

func TestWithoutAProviderTheServiceBootsWithoutStorage(t *testing.T) {
	files := bootstrap.NewStorage(&config.Config{}, zap.NewNop())

	assert.False(t, storage.IsConfigured(files))
}

// Refusing the boot here would be a silent exit 1 under fx.NopLogger, for a
// feature most environments do not use.
func TestAnIncompleteStorageBootsWithoutStorageAndNamesWhatIsMissing(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)

	files := bootstrap.NewStorage(&config.Config{
		Storage: config.StorageConfig{
			Provider: "AWS-S3",
			Aws:      config.StorageAwsConfig{S3BucketName: "bucket-test"},
		},
	}, zap.New(core))

	assert.False(t, storage.IsConfigured(files))
	assert.Equal(t, 1, logs.Len())
	assert.ElementsMatch(t, []any{
		"STORAGE_PUBLIC_BASE_URL",
		"STORAGE_AWS_REGION",
		"STORAGE_AWS_ACCESS_KEY_ID",
		"STORAGE_AWS_SECRET_ACCESS_KEY",
	}, logs.All()[0].ContextMap()["missing"])

	unknown := bootstrap.NewStorage(&config.Config{Storage: config.StorageConfig{Provider: "GCS"}}, zap.NewNop())

	assert.False(t, storage.IsConfigured(unknown))
}

func TestACompleteConfigurationBuildsTheS3Adapter(t *testing.T) {
	var cfg config.Config
	assert.NoError(t, yaml.Unmarshal([]byte(storageYaml), &cfg))

	assert.True(t, storage.IsConfigured(bootstrap.NewStorage(&cfg, zap.NewNop())))
}
