(with_statement
	(with_clause
     	(with_item
    		value: (call
            		function: (attribute
                    	object: (identifier) @module
                        attribute: (identifier) @moduleMethod
                    )
                    arguments: (argument_list
                    [
                    	(string) @path
                        (_)
                    ]
                    )
            )
        	alias: (identifier) @varOut
    	) 
    )@withItem
    body: (block
    	(expression_statement [
            (assignment
                right: (await
                    (call
                        function: (attribute
                            object: (identifier) @varIn
                            attribute: (identifier) @func
                        ) 
                    )
                )
            )
            (await
            	(call
                	function: (attribute
                    	object: (identifier) @varIn
                        attribute: (identifier) @func
                    )
                )
            )
        ]
        )
    )
) @withStatement