(call_expression
	function: (member_expression
		object: (_) @obj
		property: (_) @prop
	)
	arguments: (arguments
		.
		(string) @path
	)
)

;; TODO: check scoping! This will catch other uses of 
;; varName in different scopes resulting in incorrect behaviour.
