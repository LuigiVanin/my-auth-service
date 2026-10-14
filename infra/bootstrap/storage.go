package bootstrap

import (
	"auth_service/infra/config"
	"auth_service/shared/storage"
	"auth_service/shared/storage/adapters"

	"go.uber.org/zap"
)

const StorageProviderAwsS3 = "AWS-S3"

// A missing or incomplete storage never stops the boot, the same as a missing
// RESEND_API_KEY: everything else keeps working and the avatar routes answer 501.
func NewStorage(cfg *config.Config, logger *zap.Logger) storage.IStorage {
	settings := cfg.Storage

	switch settings.Provider {
	case "":
		logger.Warn("STORAGE_PROVIDER is not set - the avatar routes will answer 501")
		return storage.Unconfigured{}

	case StorageProviderAwsS3:
		missing := missingAwsSettings(settings)

		if len(missing) > 0 {
			logger.Error(
				"Storage is incomplete - the avatar routes will answer 501",
				zap.String("provider", settings.Provider),
				zap.Strings("missing", missing),
			)
			return storage.Unconfigured{}
		}

		return adapters.NewS3Adapter(adapters.S3Options{
			Bucket:          settings.Aws.S3BucketName,
			Region:          settings.Aws.Region,
			AccessKeyId:     settings.Aws.AccessKeyId,
			SecretAccessKey: settings.Aws.SecretAccessKey,
		})

	default:
		logger.Error("Unknown storage provider - the avatar routes will answer 501", zap.String("provider", settings.Provider))
		return storage.Unconfigured{}
	}
}

func missingAwsSettings(settings config.StorageConfig) []string {
	required := []struct {
		name  string
		value string
	}{
		{"STORAGE_PUBLIC_BASE_URL", settings.PublicBaseUrl},
		{"STORAGE_AWS_S3_BUCKET_NAME", settings.Aws.S3BucketName},
		{"STORAGE_AWS_REGION", settings.Aws.Region},
		{"STORAGE_AWS_ACCESS_KEY_ID", settings.Aws.AccessKeyId},
		{"STORAGE_AWS_SECRET_ACCESS_KEY", settings.Aws.SecretAccessKey},
	}

	missing := []string{}

	for _, setting := range required {
		if setting.value == "" {
			missing = append(missing, setting.name)
		}
	}

	return missing
}

func NewUrlResolver(cfg *config.Config) storage.IUrlResolver {
	return storage.NewUrlResolver(cfg.Storage.PublicBaseUrl)
}
