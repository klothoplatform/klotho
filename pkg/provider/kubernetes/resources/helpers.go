package resources

import (
	"fmt"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/core"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func sanitizeString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

func GenerateRoleArnPlaceholder(unit string) string {
	return fmt.Sprintf("%sRoleArn", sanitizeString(unit))
}

func GenerateImagePlaceholder(unit string) string {
	return fmt.Sprintf("%sImage", sanitizeString(unit))
}

func GenerateTargetGroupBindingPlaceholder(unit string) string {
	return fmt.Sprintf("%sTargetGroupArn", sanitizeString(unit))
}

func GenerateInstanceTypeKeyPlaceholder(unit string) string {
	return fmt.Sprintf("%sInstanceTypeKey", sanitizeString(unit))
}

func GenerateInstanceTypeValuePlaceholder(unit string) string {
	return fmt.Sprintf("%sInstanceTypeValue", sanitizeString(unit))

}
func GenerateEnvVarKeyValue(key string) (k string, v string) {
	k = key
	v = sanitizeString(key)
	return
}

func ListAll() []core.Resource {
	return []core.Resource{
		&Deployment{},
		&HelmChart{},
		&HorizontalPodAutoscaler{},
		&Manifest{},
		&KustomizeDirectory{},
		&Kubeconfig{},
		&Namespace{},
		&Pod{},
		&Service{},
		&ServiceAccount{},
		&ServiceExport{},
		&TargetGroupBinding{},
	}
}
