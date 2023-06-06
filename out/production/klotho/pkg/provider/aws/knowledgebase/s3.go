package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var S3KB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.S3Object, *resources.S3Bucket]{},
	knowledgebase.EdgeBuilder[*resources.S3BucketPolicy, *resources.OriginAccessIdentity]{
		Configure: func(policy *resources.S3BucketPolicy, oai *resources.OriginAccessIdentity, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			oai.Comment = "this is needed to set up S3 polices so that the S3 bucket is not public"
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.S3BucketPolicy, *resources.S3Bucket]{
		Configure: func(policy *resources.S3BucketPolicy, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy.Bucket = bucket
			return nil
		},
	},
)
