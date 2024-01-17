package reader

import knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		QualifiedTypeName string `json:"qualified_type_name" yaml:"qualified_type_name"`

		DisplayName string `json:"display_name" yaml:"display_name"`

		Properties Properties `json:"properties" yaml:"properties"`

		Classification knowledgebase.Classification `json:"classification" yaml:"classification"`

		PathSatisfaction knowledgebase.PathSatisfaction `json:"path_satisfaction" yaml:"path_satisfaction"`

		AdditionalRules []knowledgebase.AdditionalRule `json:"additional_rules" yaml:"additional_rules"`

		Consumption knowledgebase.Consumption `json:"consumption" yaml:"consumption"`

		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext knowledgebase.DeleteContext `json:"delete_context" yaml:"delete_context"`
		// Views defines the views that the resource should be added to as a distinct node
		Views map[string]string `json:"views" yaml:"views"`

		NoIac bool `json:"no_iac" yaml:"no_iac"`

		SanitizeNameTmpl string `yaml:"sanitize_name"`
	}
)

func (r *ResourceTemplate) Convert() (*knowledgebase.ResourceTemplate, error) {
	kbProperties, err := r.Properties.Convert()
	if err != nil {
		return nil, err
	}
	var sanitizeTmpl *knowledgebase.SanitizeTmpl
	if r.SanitizeNameTmpl != "" {
		sanitizeTmpl, err = knowledgebase.NewSanitizationTmpl(r.QualifiedTypeName, r.SanitizeNameTmpl)
		if err != nil {
			return nil, err
		}
	}
	return &knowledgebase.ResourceTemplate{
		QualifiedTypeName: r.QualifiedTypeName,
		DisplayName:       r.DisplayName,
		Properties:        kbProperties,
		AdditionalRules:   r.AdditionalRules,
		Classification:    r.Classification,
		PathSatisfaction:  r.PathSatisfaction,
		Consumption:       r.Consumption,
		DeleteContext:     r.DeleteContext,
		Views:             r.Views,
		NoIac:             r.NoIac,
		SanitizeNameTmpl:  sanitizeTmpl,
	}, nil
}
