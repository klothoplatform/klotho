package classification

import (
	"embed"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Classifier interface {
		GetFunctionality(resource core.Resource) core.Functionality
		GetClassification(resource core.Resource) Classification
	}

	ClassificationDocument struct {
		Classifications map[string]Classification
	}

	Classification struct {
		Is    []string `json:"is"`
		Gives []Gives  `json:"gives"`
	}

	Gives struct {
		Attribute     string
		Functionality []string
	}
)

func (g *Gives) UnmarshalJSON(content []byte) error {
	givesString := string(content)
	if givesString == "" {
		return nil
	}
	gives := strings.Split(givesString, ":")
	g.Attribute = strings.ReplaceAll(gives[0], "\"", "")
	if len(gives) == 1 {
		g.Functionality = []string{"*"}
		return nil
	}
	g.Functionality = strings.Split(strings.ReplaceAll(gives[1], "\"", ""), ",")
	return nil
}

func (c *ClassificationDocument) GivesAttributeForFunctionality(resource core.Resource, attribute string, functionality core.Functionality) bool {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	for _, give := range c.Classifications[bareRes.Id().String()].Gives {
		if give.Attribute == attribute && (collectionutil.Contains(give.Functionality, string(functionality)) || collectionutil.Contains(give.Functionality, "*")) {
			return true
		}
	}
	return false
}

func (c *ClassificationDocument) GetClassification(resource core.Resource) Classification {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	return c.Classifications[bareRes.Id().String()]
}

func (c *ClassificationDocument) GetFunctionality(resource core.Resource) core.Functionality {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	classification := c.GetClassification(bareRes)
	if len(classification.Is) == 0 {
		return core.Unknown
	}
	var functionality core.Functionality
	for _, c := range classification.Is {
		matched := true
		alreadySet := functionality != ""
		switch c {
		case "compute":
			functionality = core.Compute
		case "cluster":
			functionality = core.Cluster
		case "storage":
			functionality = core.Storage
		case "api":
			functionality = core.Api
		case "messaging":
			functionality = core.Messaging
		default:
			matched = false
		}
		if matched && alreadySet {
			return core.Unknown
		}
	}
	if functionality == "" {
		return core.Unknown
	}
	return functionality
}

func (c *ClassificationDocument) ResourceContainsClassifications(resource core.Resource, needs []string) bool {
	classifications := c.GetClassification(resource)
	for _, need := range needs {
		if !collectionutil.Contains(classifications.Is, need) && resource.Id().Type != need {
			return false
		}
	}
	return true
}

func ReadClassificationDoc(path string, fs embed.FS) (*ClassificationDocument, error) {
	classificationDoc := &ClassificationDocument{}
	if path == "" {
		classificationDoc.Classifications = map[string]Classification{}
		return classificationDoc, nil
	}
	f, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(f, &classificationDoc.Classifications)
	if err != nil {
		return nil, err
	}
	return classificationDoc, nil
}
