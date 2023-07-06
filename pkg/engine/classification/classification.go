package classification

import (
	"embed"
	"encoding/json"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Classifier interface {
		GetFunctionality(resource core.Resource) Functionality
	}

	ClassificationDocument struct {
		classifications map[string]Classification
	}

	Classification struct {
		Is    []string `json:"is"`
		Gives []string `json:"gives"`
	}

	Functionality string
)



const (
	Compute Functionality = "compute"
	Cluster Functionality = "cluster"
	Storage Functionality = "storage"
	Network Functionality = "network"
	Api     Functionality = "api"
	Unknown Functionality = "Unknown"
)

func (c *ClassificationDocument) GetClassification(resource core.Resource) Classification {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	return c.classifications[bareRes.Id().String()]
}

func (c *ClassificationDocument) GetFunctionality(resource core.Resource) Functionality {
	bareRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	classification := c.GetClassification(bareRes)
	if len(classification.Is) == 0 {
		return Unknown
	}
	var functionality Functionality
	for _, c := range classification.Is {
		matched := true
		alreadySet := functionality != ""
		switch c {
		case "compute":
			functionality = Compute
		case "cluster":
			functionality = Cluster
		case "storage":
			functionality = Storage
		case "network":
			functionality = Network
		case "api":
			functionality = Api
		default:
			matched = false
		}
		if matched && alreadySet {
			return Unknown
		}
	}
	return functionality
}

func (c *ClassificationDocument) ResourceContainsClassifications(resource core.Resource, needs []string) bool {
	classifications := c.GetClassification(resource)
	for _, need := range needs {
		if !collectionutil.Contains(classifications.Is, need) {
			return false
		}
	}
	return true
}

func ReadClassificationDoc(path string, fs embed.FS) (*ClassificationDocument, error) {
	classificationDoc := &ClassificationDocument{}
	if path == "" {
		classificationDoc.classifications = map[string]Classification{}
		return classificationDoc, nil
	}
	f, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(f, &classificationDoc.classifications)
	if err != nil {
		return nil, err
	}
	return classificationDoc, nil
}