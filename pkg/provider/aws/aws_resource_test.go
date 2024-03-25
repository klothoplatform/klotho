package aws

import (
	"slices"
	"testing"

	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/templates"
)

var (
	nonTaggableResource = []string{
		"aws:SERVICE_API",
		"aws:ecr_image",
		"aws:security_group_rule",
		"aws:secret_version",
		"aws:s3_bucket_policy",
		"aws:route_table_association",
		"aws:availability_zone",
		"aws:region",
		"aws:listener_certificate",
		"aws:efs_mount_target",
		"aws:rds_proxy_target_group",
		"aws:api_integration",
		"aws:lambda_permission",
		"aws:lambda_event_source_mapping",
		"aws:cloudfront_origin_access_identity",
		"aws:iam_role_policy_attachment",
		"aws:api_deployment",
		"aws:api_method",
		"aws:api_resource",
		"aws:ses_email_identity",
		"aws:ecs_cluster_capacity_provider",
		"aws:sns_topic_subscription",
		"aws:cloudwatch_dashboard",
	}
)

func Test_AwsResourcesSupportTags(t *testing.T) {
	// Test_AwsResourcesSupportTags tests that all AWS resources support tags
	// and that the tag keys are unique.

	checked_types := make(map[string]bool)

	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := kb.Edges()
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if edge.Source.Id().Provider == "aws" && !checked_types[edge.Source.Id().Type] {
			properties := edge.Source.Properties
			if properties["Tags"] == nil && !slices.Contains(nonTaggableResource, edge.Source.QualifiedTypeName) {
				t.Errorf("edge %s does not support tags", edge.Source.Id())
			}
			checked_types[edge.Source.Id().Type] = true
		}
		if edge.Target.Id().Provider == "aws" && !checked_types[edge.Target.Id().Type] {
			properties := edge.Target.Properties
			if properties["Tags"] == nil && !slices.Contains(nonTaggableResource, edge.Target.QualifiedTypeName) {
				t.Errorf("edge %s does not support tags", edge.Target.Id())
			}
			checked_types[edge.Target.Id().Type] = true
		}
	}
}
