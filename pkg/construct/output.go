package construct

type Output struct {
	Ref   PropertyRef `json:"ref,omitempty" yaml:"ref,omitempty"`
	Value any         `json:"value,omitempty" yaml:"value,omitempty"`
}
