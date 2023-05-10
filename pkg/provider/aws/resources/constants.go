package resources

const AWS_PROVIDER = "aws"

type EksNodeType string

const (
	Fargate EksNodeType = "fargate"
	Node    EksNodeType = "node"
)
