package constructs

func ResolveConstruct(inputs map[string]any, constructId ConstructId) (*Construct, error) {
	inputs["Name"] = constructId.InstanceId
	context := NewConstructContext(constructId, inputs)
	return context.evaluateConstruct(), nil
}

func NewContext(inputs map[string]any, constructId ConstructId) *ConstructContext {
	inputs["Name"] = constructId.InstanceId
	return NewConstructContext(constructId, inputs)
}
