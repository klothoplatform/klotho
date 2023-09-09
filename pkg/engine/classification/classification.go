package classification

import (
	"embed"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
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

func (c *ClassificationDocument) GivesAttributeForFunctionality(resource construct.Resource, attribute string, functionality construct.Functionality) bool {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(construct.Resource)
	for _, give := range c.Classifications[bareRes.Id().String()].Gives {
		if give.Attribute == attribute && (collectionutil.Contains(give.Functionality, string(functionality)) || collectionutil.Contains(give.Functionality, "*")) {
			return true
		}
	}
	return false
}

func (c *ClassificationDocument) GetClassification(id construct.ResourceId) Classification {
	return c.Classifications[id.QualifiedTypeName()]
}

func (c *ClassificationDocument) GetFunctionality(id construct.ResourceId) construct.Functionality {
	classification := c.GetClassification(id)
	if len(classification.Is) == 0 {
		return construct.Unknown
	}
	var functionality construct.Functionality
	for _, c := range classification.Is {
		matched := true
		alreadySet := functionality != ""
		switch c {
		case "compute":
			functionality = construct.Compute
		case "cluster":
			functionality = construct.Cluster
		case "storage":
			functionality = construct.Storage
		case "api":
			functionality = construct.Api
		case "messaging":
			functionality = construct.Messaging
		default:
			matched = false
		}
		if matched && alreadySet {
			return construct.Unknown
		}
	}
	if functionality == "" {
		return construct.Unknown
	}
	return functionality
}

func (c *ClassificationDocument) ResourceContainsClassifications(id construct.ResourceId, needs []string) bool {
	classifications := c.GetClassification(id)
	for _, need := range needs {
		if !collectionutil.Contains(classifications.Is, need) && id.Type != need {
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
