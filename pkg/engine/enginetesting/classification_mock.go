package enginetesting

import (
	"github.com/klothoplatform/klotho/pkg/engine/classification"
)

var BaseClassificationDocument = &classification.ClassificationDocument{
	Classifications: map[string]classification.Classification{
		"mock:mock1:": {Gives: []classification.Gives{}, Is: []string{"compute", "kv", "nosql"}},
		"mock:mock2:": {Gives: []classification.Gives{}, Is: []string{"compute", "instance", "storage"}},
		"mock:mock3:": {Gives: []classification.Gives{{Attribute: "serverless", Functionality: []string{"compute"}}}, Is: []string{"relational", "storage"}},
		"mock:mock4:": {Gives: []classification.Gives{{Attribute: "highly_available", Functionality: []string{"compute"}}}, Is: []string{}},
	},
}
