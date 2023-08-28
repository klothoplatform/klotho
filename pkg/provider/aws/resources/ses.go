package resources

import "github.com/klothoplatform/klotho/pkg/construct"

type (
	SesEmailIdentity struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		EmailIdentity string
	}
)

const (
	SES_EMAIL_IDENTITY = "ses_email_identity"
)

func (q *SesEmailIdentity) BaseConstructRefs() construct.BaseConstructSet {
	return q.ConstructRefs
}

// Id returns the id of the cloud resource
func (q *SesEmailIdentity) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     SES_EMAIL_IDENTITY,
		Name:     q.Name,
	}
}

func (q *SesEmailIdentity) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
