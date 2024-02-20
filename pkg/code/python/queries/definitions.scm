;; Finds all assignments (since Python does not distinguish between definition and assignment)
;; and function definitions.
;; @name - name of the variable or function
;; @statement - the whole matching assignment/definition
;; @value - (optional) value of the variable
;; @params - (optional) parameters of the function
;; @body - (optional) body of the function
[
	(assignment
    	left: (identifier) @name
        right: (_) @value
    )
    (function_definition
    	name: (identifier) @name
        parameters: (_) @params
        body: (_) @body
    )
] @statement
