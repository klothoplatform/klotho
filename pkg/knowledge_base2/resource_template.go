package knowledgebase2

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/set"
	"gopkg.in/yaml.v3"
)

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		QualifiedTypeName string `json:"qualified_type_name" yaml:"qualified_type_name"`

		DisplayName string `json:"display_name" yaml:"display_name"`

		Properties Properties `json:"properties" yaml:"properties"`

		Classification Classification `json:"classification" yaml:"classification"`

		HasServiceApi bool `json:"has_service_api" yaml:"has_service_api"`

		PathSatisfaction PathSatisfaction `json:"path_satisfaction" yaml:"path_satisfaction"`

		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext DeleteContext `json:"delete_context" yaml:"delete_context"`
		// Views defines the views that the resource should be added to as a distinct node
		Views map[string]string `json:"views" yaml:"views"`

		NoIac bool `json:"no_iac" yaml:"no_iac"`

		SanitizeNameTmpl string `yaml:"sanitize_name"`
	}

	Properties map[string]*Property

	Property struct {
		Name string `json:"name" yaml:"name"`
		// Type defines the type of the property
		Type string `json:"type" yaml:"type"`

		Namespace bool `json:"namespace" yaml:"namespace"`

		DefaultValue any `json:"default_value" yaml:"default_value"`

		Required bool `json:"required" yaml:"required"`

		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`

		DeployTime bool `json:"deploy_time" yaml:"deploy_time"`

		OperationalRule *OperationalRule `json:"operational_rule" yaml:"operational_rule"`

		Properties map[string]*Property `json:"properties" yaml:"properties"`

		Path string `json:"-" yaml:"-"`
	}

	Classification struct {
		Is    []string `json:"is" yaml:"is"`
		Gives []Gives  `json:"gives" yaml:"gives"`
	}

	Gives struct {
		Attribute     string
		Functionality []string
	}

	// DeleteContext is supposed to tell us when we are able to delete a resource based on its dependencies
	DeleteContext struct {
		// RequiresNoUpstream is a boolean that tells us if deletion relies on there being no upstream resources
		RequiresNoUpstream bool `yaml:"requires_no_upstream" toml:"requires_no_upstream"`
		// RequiresNoDownstream is a boolean that tells us if deletion relies on there being no downstream resources
		RequiresNoDownstream bool `yaml:"requires_no_downstream" toml:"requires_no_downstream"`
		// RequiresNoUpstreamOrDownstream is a boolean that tells us if deletion relies on there being no upstream or downstream resources
		RequiresNoUpstreamOrDownstream bool `yaml:"requires_no_upstream_or_downstream" toml:"requires_no_upstream_or_downstream"`
	}

	PathSatisfaction struct {
		AsTarget []PathSatisfactionRoute `json:"as_target" yaml:"as_target"`
		AsSource []PathSatisfactionRoute `json:"as_source" yaml:"as_source"`
	}

	PathSatisfactionRoute struct {
		Classification    string                            `json:"classification" yaml:"classification"`
		PropertyReference string                            `json:"property_reference" yaml:"property_reference"`
		Validity          PathSatisfactionValidityOperation `json:"validity" yaml:"validity"`
	}

	PathSatisfactionValidityOperation string

	Functionality string
)

const (
	Compute   Functionality = "compute"
	Cluster   Functionality = "cluster"
	Storage   Functionality = "storage"
	Api       Functionality = "api"
	Messaging Functionality = "messaging"
	Unknown   Functionality = "Unknown"

	DownstreamOperation PathSatisfactionValidityOperation = "downstream"
)

func (p *PathSatisfactionRoute) UnmarshalYAML(n *yaml.Node) error {
	type h PathSatisfactionRoute
	var p2 h
	err := n.Decode(&p2)
	if err != nil {
		routeString := n.Value
		routeParts := strings.Split(routeString, "#")
		p2.Classification = routeParts[0]
		if len(routeParts) > 1 {
			p2.PropertyReference = strings.Join(routeParts[1:], "#")
		}
		*p = PathSatisfactionRoute(p2)
		return nil
	}
	p2.Validity = PathSatisfactionValidityOperation(strings.ToLower(string(p2.Validity)))
	*p = PathSatisfactionRoute(p2)
	return nil
}

func (p *Properties) UnmarshalYAML(n *yaml.Node) error {
	type h Properties
	var p2 h
	err := n.Decode(&p2)
	if err != nil {
		return err
	}
	for name, property := range p2 {
		property.Name = name
		property.Path = name
		setChildPaths(property, name)
		p2[name] = property
	}
	*p = Properties(p2)
	return nil
}

func setChildPaths(property *Property, currPath string) {
	for name, child := range property.Properties {
		child.Name = name
		path := currPath + "." + name
		child.Path = path
		setChildPaths(child, path)
		property.Properties[name] = child
	}
}

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

func (g *Gives) UnmarshalYAML(n *yaml.Node) error {
	givesString := n.Value
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

func (template ResourceTemplate) Id() construct.ResourceId {
	args := strings.Split(template.QualifiedTypeName, ":")
	return construct.ResourceId{
		Provider: args[0],
		Type:     args[1],
	}
}

// CreateResource creates an empty resource for the given ID, running any sanitization rules on the ID.
// NOTE: Because of sanitization, once created callers must use the resulting ID for all future operations
// and not the input ID.
func CreateResource(kb TemplateKB, id construct.ResourceId) (*construct.Resource, error) {
	rt, err := kb.GetResourceTemplate(id)
	if err != nil {
		return nil, fmt.Errorf("could not create resource: get template err: %w", err)
	}
	id.Name, err = rt.SanitizeName(id.Name)
	if err != nil {
		return nil, fmt.Errorf("could not create resource: %w", err)
	}
	return &construct.Resource{
		ID:         id,
		Properties: make(construct.Properties),
	}, nil
}

func SanitizeEdge(kb TemplateKB, e construct.SimpleEdge) (construct.SimpleEdge, error) {
	sRT, err := kb.GetResourceTemplate(e.Source)
	if err != nil {
		return e, fmt.Errorf("could not get source resource template: %w", err)
	}
	tRT, err := kb.GetResourceTemplate(e.Target)
	if err != nil {
		return e, fmt.Errorf("could not get target resource template: %w", err)
	}
	e.Source.Name, err = sRT.SanitizeName(e.Source.Name)
	if err != nil {
		return e, fmt.Errorf("could not sanitize source name: %w", err)
	}
	e.Target.Name, err = tRT.SanitizeName(e.Target.Name)
	if err != nil {
		return e, fmt.Errorf("could not sanitize target name: %w", err)
	}
	return e, nil
}

func (rt ResourceTemplate) SanitizeName(name string) (string, error) {
	if rt.SanitizeNameTmpl == "" {
		return name, nil
	}
	nt, err := template.New(rt.QualifiedTypeName + "/sanitize_name").
		Funcs(template.FuncMap{
			"replace": func(pattern, replace, name string) (string, error) {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return name, err
				}
				return re.ReplaceAllString(name, replace), nil
			},

			"length": func(min, max int, name string) string {
				if len(name) < min {
					return name + strings.Repeat("0", min-len(name))
				}
				if len(name) > max {
					base := name[:max-8]
					h := sha256.New()
					fmt.Fprint(h, name)
					x := fmt.Sprintf("%x", h.Sum(nil))
					return base + x[:8]
				}
				return name
			},

			"lower": strings.ToLower,
			"upper": strings.ToUpper,
		}).
		Parse(rt.SanitizeNameTmpl)
	if err != nil {
		return name, fmt.Errorf("could not parse sanitize name template %q: %w", rt.SanitizeNameTmpl, err)
	}
	buf := new(bytes.Buffer)
	err = nt.Execute(buf, name)
	if err != nil {
		return name, fmt.Errorf("could not execute sanitize name template on %q: %w", name, err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (template ResourceTemplate) GivesAttributeForFunctionality(attribute string, functionality Functionality) bool {
	for _, give := range template.Classification.Gives {
		if give.Attribute == attribute && (collectionutil.Contains(give.Functionality, string(functionality)) || collectionutil.Contains(give.Functionality, "*")) {
			return true
		}
	}
	return false
}

func (template ResourceTemplate) GetFunctionality() Functionality {
	if len(template.Classification.Is) == 0 {
		return Unknown
	}
	var functionality Functionality
	for _, c := range template.Classification.Is {
		matched := true
		alreadySet := functionality != ""
		switch c {
		case "compute":
			functionality = Compute
		case "cluster":
			functionality = Cluster
		case "storage":
			functionality = Storage
		case "api":
			functionality = Api
		case "messaging":
			functionality = Messaging
		default:
			matched = false
		}
		if matched && alreadySet {
			return Unknown
		}
	}
	if functionality == "" {
		return Unknown
	}
	return functionality
}

func (template ResourceTemplate) ResourceContainsClassifications(needs []string) bool {
	for _, need := range needs {
		if !collectionutil.Contains(template.Classification.Is, need) && template.QualifiedTypeName != need {
			return false
		}
	}
	return true
}

func (template ResourceTemplate) GetNamespacedProperty() *Property {
	for _, property := range template.Properties {
		if property.Namespace {
			return property
		}
	}
	return nil
}

func (template ResourceTemplate) GetProperty(name string) *Property {
	fields := strings.Split(name, ".")
	properties := template.Properties
	for i, field := range fields {
		currFieldName := strings.Split(field, "[")[0]
		found := false
		for _, property := range properties {
			if property.Name != currFieldName {
				continue
			}
			found = true
			if len(fields) == i+1 {
				return property
			} else {
				pType, err := property.PropertyType()
				if err != nil {
					return nil
				}
				// If the property types are a list or map, without sub fields
				//  we want to just return the property since we are setting an index or key of the end value
				switch p := pType.(type) {
				case *MapPropertyType:
					if p.Value != "" {
						return &Property{
							Type: p.Value,
							Path: name,
						}
					}
				case *ListPropertyType:
					if p.Value != "" {
						return &Property{
							Type: p.Value,
							Path: name,
						}
					}
				}
				properties = property.Properties
			}
		}
		if !found {
			return nil
		}
	}
	return nil
}

var ErrStopWalk = errors.New("stop walk")

func (tmpl ResourceTemplate) LoopProperties(res *construct.Resource, addProp func(*Property) error) error {
	queue := []Properties{tmpl.Properties}
	var props Properties
	var errs error
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]
		for _, prop := range props {
			err := addProp(prop)
			if err != nil {
				if errors.Is(err, ErrStopWalk) {
					return nil
				}
				errs = errors.Join(errs, err)
				continue
			}

			if strings.HasPrefix(prop.Type, "list") || strings.HasPrefix(prop.Type, "set") {
				p, err := res.GetProperty(prop.Path)
				if err != nil || p == nil {
					continue
				}
				// Because lists/sets will start as empty, do not recurse into their sub-properties if its not set.
				// To allow for defaults within list objects and operational rules to be run, we will look in the property
				// to see if there are values.
				if strings.HasPrefix(prop.Type, "list") {
					length := reflect.ValueOf(p).Len()
					for i := 0; i < length; i++ {
						subProperties := make(Properties)
						for subK, subProp := range prop.Properties {
							propTemplate := subProp.Clone()
							propTemplate.ReplacePath(prop.Path, fmt.Sprintf("%s[%d]", prop.Path, i))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}
				} else if strings.HasPrefix(prop.Type, "set") {
					hs := p.(set.HashedSet[string, any])
					for k := range hs.ToMap() {
						subProperties := make(Properties)
						for subK, subProp := range prop.Properties {
							propTemplate := subProp.Clone()
							propTemplate.ReplacePath(prop.Path, fmt.Sprintf("%s[%s]", prop.Path, k))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}

				}
			} else if prop.Properties != nil {
				queue = append(queue, prop.Properties)
			}
		}
	}
	return errs
}
