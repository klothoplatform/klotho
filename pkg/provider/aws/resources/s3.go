package resources

import "github.com/klothoplatform/klotho/pkg/config"

func GetPayloadsBucketName(config config.Application) string {
	return SanitizeS3BucketName(config.AppName)
}
