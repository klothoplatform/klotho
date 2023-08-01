package resources

import "github.com/klothoplatform/klotho/pkg/core"

type (
	SesEmailIdentity struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		EmailIdentity string
	}
)

const (
	SES_EMAIL_IDENTITY = "ses_email_identity"
)

func (q *SesEmailIdentity) BaseConstructRefs() core.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SesEmailIdentity) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SES_EMAIL_IDENTITY,
		Name:     q.Name,
	}
}

func (q *SesEmailIdentity) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
