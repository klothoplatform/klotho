[
	(variable_declarator
		name: (_) @name
		value: 
		(new_expression 
			constructor: (_) @constructor
		)@expression
	)
	(assignment_expression
		left: (member_expression
			object: (_)@object
			property: (_)@name
		)
		right: (new_expression
			constructor: (_)@constructor
		) @expression
	)
]