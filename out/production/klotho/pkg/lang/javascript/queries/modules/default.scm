(assignment_expression 
	left: [
		(member_expression
			object: (identifier) @obj
			property: (property_identifier) @prop
		)
		(identifier) @prop
	]
	right: (identifier) @last
)