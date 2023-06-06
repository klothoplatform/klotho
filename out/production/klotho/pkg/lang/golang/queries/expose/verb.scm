(call_expression ; eg. r.Get("/", func())
	(selector_expression
    	(identifier)@routerName
        (field_identifier)@verb
    )
	(argument_list
    	(interpreted_string_literal)@path
	)
)@expression