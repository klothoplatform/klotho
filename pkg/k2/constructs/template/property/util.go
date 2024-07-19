package property

import (
	"errors"
	"strings"
)

const ErrRequiredProperty = "required property %s is not set"

var ErrStopWalk = errors.New("stop walk")

// ReplacePath runs a simple [strings.ReplaceAll] on the path of the property and all of its sub properties.
// NOTE: this mutates the property, so make sure to [Property.Clone] it first if you don't want that.
func ReplacePath(p Property, original, replacement string) {
	p.Details().Path = strings.ReplaceAll(p.Details().Path, original, replacement)
	for _, prop := range p.SubProperties() {
		ReplacePath(prop, original, replacement)
	}
}
