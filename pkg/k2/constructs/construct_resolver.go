package constructs

func NewContext(inputs map[string]any, constructId ConstructId) *ConstructContext {
	inputs["Name"] = constructId.InstanceId
	return NewConstructContext(constructId, inputs)
}
