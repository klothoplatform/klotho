package resources

import "github.com/mitchellh/mapstructure"

const AWS_PROVIDER = "aws"

type EksNodeType string

const (
	Fargate EksNodeType = "fargate"
	Node    EksNodeType = "node"
)

func getMapDecoder(result interface{}) *mapstructure.Decoder {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, Result: result})
	if err != nil {
		panic(err)
	}
	return decoder
}
