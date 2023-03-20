package annotation

import (
	"fmt"
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

func ParseCapabilities(s string) ([]*Capability, error) {
	var capabilities []*Capability
	matches := capabilityStartRegex.FindAllStringSubmatchIndex(s, -1)
	previousEnd := -1
	for _, match := range matches {
		if len(match) == 0 {
			continue
		}

		// nested annotations are not supported (though not necessarily syntactically incorrect)
		if previousEnd > -1 && match[0] <= previousEnd {
			continue
		}

		bodyExprStart := match[4]
		var bodyExprEnd int
		var err error

		cap := &Capability{
			Name: s[match[2]:match[3]],
		}

		if bodyExprStart != -1 {
			bodyExprEnd = bodyExprStart + getExprEndIndex(s[bodyExprStart:], "", '{', '}')
			cap.Directives, err = ParseDirectives(s[bodyExprStart+1 : bodyExprEnd])
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
				return nil, fmt.Errorf("'id' can only contain alphanumeric, -, _, ., :, and /. 'id' was %s", id)
			}
		}
		cap.ID = id
		capabilities = append(capabilities, cap)
		previousEnd = match[1] - 1
	}
	return capabilities, nil
}

// getExprEndIndex gets the index of end delimiter in a balanced expression of start and end delimiters.
//
// The escape argument is used for detecting escaped delimiters and should either be `\` or `\\`
// depending on the format of the input string.
func getExprEndIndex(input, escape string, start, end rune) int {
	escapedStartPattern := regexp.MustCompile(fmt.Sprintf(`^[^%c]*?((?:%s)*)\%c`, start, escape, start))
	escapedEndPattern := regexp.MustCompile(fmt.Sprintf(`^[^%c]*?((?:%s)*)\%c`, end, escape, end))
	sCount := 0
	eCount := 0
	lastMatchIndex := -1
	for i := 0; i < len(input); i++ {
		switch rune(input[i]) {
		case start:
			match := escapedStartPattern.FindStringSubmatch(input[lastMatchIndex+1:])
			if match[1] == "" || len(match[1])%len(escape) != 0 {
				sCount++
			}
			lastMatchIndex = i
		case end:
			match := escapedEndPattern.FindStringSubmatch(input[lastMatchIndex+1:])
			if match[1] == "" || len(match[1])%len(escape) != 0 {
				eCount++
			}
			lastMatchIndex = i
		}
		if sCount == eCount {
			return i
		}
	}
	return -1
}

const ExecutionUnitCapability = "execution_unit"
const ExposeCapability = "expose"
const PersistCapability = "persist"
const AssetCapability = "embed_assets"
const StaticUnitCapability = "static_unit"
const PubSubCapability = "pubsub"
const ConfigCapability = "config"
