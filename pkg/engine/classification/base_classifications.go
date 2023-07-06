package classification

var BaseClassificationDocument = &ClassificationDocument{
	classifications: map[string]Classification{
		"aws:dynamodb_table:":      {Gives: []string{}, Is: []string{"storage", "kv", "nosql"}},
		"aws:ec2_instance:":        {Gives: []string{}, Is: []string{"compute", "instance"}},
		"aws:ecs_cluster:":         {Gives: []string{}, Is: []string{"cluster"}},
		"aws:ecs_service:":         {Gives: []string{}, Is: []string{"compute"}},
		"aws:eks_cluster:":         {Gives: []string{}, Is: []string{"cluster", "kubernetes"}},
		"aws:elasticache_cluster:": {Gives: []string{}, Is: []string{"storage", "redis", "cache"}},
		"aws:lambda_function:":     {Gives: []string{}, Is: []string{"compute", "serverless"}},
		"aws:rds_instance:":        {Gives: []string{}, Is: []string{"storage", "relational"}},
		"aws:rest_api:":            {Gives: []string{}, Is: []string{"api"}},
		"aws:s3_bucket:":           {Gives: []string{}, Is: []string{"storage", "blob"}},
		"aws:secret:":              {Gives: []string{}, Is: []string{"storage", "secret"}},
		"aws:vpc:":                 {Gives: []string{}, Is: []string{"network"}},
		"kubernetes:deployment:":   {Gives: []string{}, Is: []string{"compute"}},
	},
}
