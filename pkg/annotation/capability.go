package annotation

import (
	"regexp"

	"github.com/pkg/errors"
)

type Capability struct {
	Name       string     `json:"name"`
	ID         string     `json:"id"`
	Directives Directives `json:"directives"`
}

var capabilityRE = regexp.MustCompile(`@klotho::(\w+)(?:\s*\{\s*([^}]*)})?`)

func ParseCapability(s string) (*Capability, error) {
	matches := capabilityRE.FindStringSubmatch(s)
	if len(matches) < 2 {
		return nil, nil
	}

	cap := &Capability{
		Name: matches[1],
	}

	if len(matches) > 2 {
		var err error
		cap.Directives, err = ParseDirectives(matches[2])
		if err != nil {
			return cap, errors.Wrap(err, "could not parse directives")
		}
		cap.ID, _ = cap.Directives.String("id")
	}

	return cap, nil
}

const ExecutionUnitCapability = "execution_unit"
const ExposeCapability = "expose"
const PersistCapability = "persist"
const AssetCapability = "embed_assets"
const StaticUnitCapability = "static_unit"
const PubSubCapability = "pubsub"
