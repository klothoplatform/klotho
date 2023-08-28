package visualizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

// TestAllTypesHaveIcons is an integration test — not a unit test!
//
// To run it, set the env var KLOTHO_VIZ_URL_BASE to a visualizer service endpoint. For local testing, this is probably
// "http://localhost:3000" or "https://viz-dev.klo.dev". If the env var isn't set, this test will be skipped.
func TestAllTypesHaveIcons(t *testing.T) {
	if !visualizerBaseUrlEnv.IsSet() {
		t.Skipf(`Skipping because %s isn't set`, visualizerBaseUrlEnv)
		return
	}
	allResources := resources.ListAll()
	testedTypes := make(coretesting.TypeRefSet)

	api := visApi{client: http.DefaultClient}
	typeNamesBuf := bytes.Buffer{}
	for _, res := range allResources {
		_, typeNames := typeNamesForResource(res)
		for _, typeName := range typeNames {
			typeNamesBuf.WriteString(typeName + "\n")
		}
	}
	validationBytes, err := api.request(http.MethodPost, `validate-types`, `application/text`, "", &typeNamesBuf)
	if !assert.NoError(t, err) {
		statusCode, isBadStatus := err.(httpStatusBad)
		if !isBadStatus || (statusCode != 400) {
			// If it's a 400 we can keep going — that just means there were unvalidated types, which we'll check below.
			// Anything else is unrecoverable.
			return
		}
	}

	var validations map[string]bool
	err = json.Unmarshal(validationBytes, &validations)
	if !assert.NoError(t, err) {
		return
	}

	for _, res := range allResources {
		provider, typeNames := typeNamesForResource(res)
		for _, typeName := range typeNames {
			t.Run(fmt.Sprintf("%s:%s", provider, typeName), func(t *testing.T) {
				testedTypes.Add(res)
				known, found := validations[typeName]
				if assert.True(t, found) {
					assert.True(t, known)
				}
			})
		}
	}

	t.Run("all types tested", func(t *testing.T) {
		for _, ref := range coretesting.FindAllResources(assert.New(t), allResources) {
			t.Run(ref.Name, func(t *testing.T) {
				testedTypes.Check(t, ref, `struct implements construct.Resource but isn't tested; add it to this test's '"allResources" var`)
			})
		}
	})

}

// typeNamesForResource returns all the possible type names for this resource, as well as the resource's provider (which
// we assume is always the same for a given resource). Keep this func in sync with resource_types.go's TypeFor.
func typeNamesForResource(res construct.Resource) (string, []string) {
	resId := res.Id()

	var typeNames []string
	// keep this in sync with resource_types.go
	switch res.(type) {
	case *resources.Subnet:
		typeNames = append(typeNames, "subnet") // not "vpc_subnet"
	case *resources.VpcEndpoint:
		typeNames = append(typeNames, "vpc_endpoint_interface", "vpc_endpoint_gateway")
	default:
		typeNames = append(typeNames, resId.Type)
	}
	return resId.Provider, typeNames
}
