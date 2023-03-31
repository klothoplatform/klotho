package annotation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/parseutils"
	"github.com/lithammer/dedent"
	"github.com/pkg/errors"
)

type (
	Capability struct {
		Name       string     `json:"name"`
		ID         string     `json:"id"`
		Directives Directives `json:"directives"`
	}

	CapabilityResult struct {
		Capability *Capability
		Error      error
	}
)

var capabilityStartRegex = regexp.MustCompile(`\s*@klotho::(\w+)\s*(\{)?`)
var idRegex = regexp.MustCompile(`^[\w-_.:/]+$`)
var extractBodyExpr = parseutils.ExpressionExtractor("", '{', '}')

func ParseCapabilities(s string) []CapabilityResult {
	var results []CapabilityResult
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

		var annotErrs multierr.Error
		bodyExprStart := submatch[4]

		cap := &Capability{
			Name: s[submatch[2]:submatch[3]],
		}

		var bodyExpr string
		if bodyExprStart != -1 {
			if exprs := extractBodyExpr(s[bodyExprStart:], 1); len(exprs) == 1 {
				bodyExpr = exprs[0]
				var err error
				if cap.Directives, err = ParseDirectives(bodyExpr[1 : len(bodyExpr)-1]); err != nil {
					annotErrs.Append(errors.Wrap(err, "could not parse directives for annotation"))
				}
			} else {
				annotErrs.Append(fmt.Errorf("incorrect number of annotation body expressions: expected=1, detected=%d", len(exprs)))
			}
		}

		id, _ := cap.Directives.String("id")
		if id != "" {
			if len(id) > 25 {
				annotErrs.Append(fmt.Errorf("'id' must be less than 25 characters in length. 'id' was %s", id))
			}
			if !idRegex.MatchString(id) {
				annotErrs.Append(fmt.Errorf("'id must match the pattern: '%s', but 'id' was %s", idRegex.String(), id))
			}
		}
		cap.ID = id
		previousEnd = submatch[1] - 1

		var err error
		if annotErrs.ErrOrNil() != nil {
			annotText := strings.TrimSpace(dedent.Dedent(s[submatch[0] : submatch[1]+len(bodyExpr)-1]))
			err = errors.Wrapf(annotErrs.ErrOrNil(), "could not parse capability:\n%s\n", annotText)
		}

		results = append(results, CapabilityResult{
			Capability: cap,
			Error:      err,
		})
	}

	return results
}

const ExecutionUnitCapability = "execution_unit"
const ExposeCapability = "expose"
const PersistCapability = "persist"
const AssetCapability = "embed_assets"
const StaticUnitCapability = "static_unit"
const PubSubCapability = "pubsub"
const ConfigCapability = "config"
