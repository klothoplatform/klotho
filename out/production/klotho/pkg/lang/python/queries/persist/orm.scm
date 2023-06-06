(expression_statement
	(assignment
    	left: (identifier) @engineVar
        right: (call
        	function: [
            	(identifier) @funcCall
                (attribute
                	object: (identifier) @module
                    attribute: (identifier) @funcCall
                )
            ]
            arguments: (argument_list
            	. (_) @connString
            )
        )
    )
) @expression