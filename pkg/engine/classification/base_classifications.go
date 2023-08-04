package classification

var BaseClassificationDocument = &ClassificationDocument{
	Classifications: map[string]Classification{
		"aws:app_runner_service:":  {Gives: []Gives{}, Is: []string{"compute", "serverless"}},
		"aws:dynamodb_table:":      {Gives: []Gives{}, Is: []string{"storage", "kv", "nosql"}},
		"aws:ec2_instance:":        {Gives: []Gives{}, Is: []string{"compute", "instance"}},
		"aws:ecs_cluster:":         {Gives: []Gives{}, Is: []string{"cluster"}},
		"aws:ecs_service:":         {Gives: []Gives{}, Is: []string{"compute"}},
		"aws:eks_cluster:":         {Gives: []Gives{}, Is: []string{"cluster", "kubernetes"}},
		"aws:elasticache_cluster:": {Gives: []Gives{}, Is: []string{"storage", "redis", "cache"}},
		"aws:lambda_function:":     {Gives: []Gives{}, Is: []string{"compute", "serverless"}},
		"aws:rds_instance:":        {Gives: []Gives{}, Is: []string{"storage", "relational"}},
		"aws:rest_api:":            {Gives: []Gives{}, Is: []string{"api"}},
		"aws:s3_bucket:":           {Gives: []Gives{}, Is: []string{"storage", "blob"}},
		"aws:secret:":              {Gives: []Gives{}, Is: []string{"storage", "secret"}},
		"aws:vpc:":                 {Gives: []Gives{}, Is: []string{"network"}},
		"kubernetes:deployment:":   {Gives: []Gives{}, Is: []string{"compute", "kubernetes"}},
	},
}
