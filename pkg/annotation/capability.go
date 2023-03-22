package annotation

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/parseutils"
	"regexp"

	"github.com/pkg/errors"
)

type Capability struct {
	Name       string     `json:"name"`
	ID         string     `json:"id"`
	Directives Directives `json:"directives"`
}

var capabilityStartRegex = regexp.MustCompile(`\s*@klotho::(\w+)\s*(\{)?`)
var idRegex = regexp.MustCompile(`^[\w-_.:/]+$`)
var extractBodyExpr = parseutils.ExpressionExtractor("", '{', '}')

func ParseCapabilities(s string) ([]*Capability, error) {
	var capabilities []*Capability
	submatchIndices := capabilityStartRegex.FindAllStringSubmatchIndex(s, -1)
	previousEnd := -1
	for _, submatch := range submatchIndices {
		if len(submatch) == 0 {
			continue
		}

		// nested annotations are not supported (though not necessarily syntactically incorrect)
		if previousEnd > -1 && submatch[0] <= previousEnd {
			continue
		}

		bodyExprStart := submatch[4]
		var err error

		cap := &Capability{
			Name: s[submatch[2]:submatch[3]],
		}

		if bodyExprStart != -1 {
			if exprs := extractBodyExpr(s[bodyExprStart:], 1); len(exprs) == 1 {
				bodyExpr := exprs[0]
				cap.Directives, err = ParseDirectives(bodyExpr[1 : len(bodyExpr)-1])
			}
		}

		if err != nil {
			return nil, errors.Wrap(err, "could not parse directives")
		}
		id, _ := cap.Directives.String("id")
		if id != "" {
			if len(id) > 25 {
				return nil, fmt.Errorf("'id' must be less than 25 characters in length. 'id' was %s", id)
			}
			if !idRegex.MatchString(id) {
				return nil, fmt.Errorf("'id must match the pattern: '%s', but 'id' was %s", idRegex.String(), id)
			}
		}
		cap.ID = id
		capabilities = append(capabilities, cap)
		previousEnd = submatch[1] - 1
	}
	return capabilities, nil
}

const ExecutionUnitCapability = "execution_unit"
const ExposeCapability = "expose"
const PersistCapability = "persist"
const AssetCapability = "embed_assets"
const StaticUnitCapability = "static_unit"
const PubSubCapability = "pubsub"
const ConfigCapability = "config"
