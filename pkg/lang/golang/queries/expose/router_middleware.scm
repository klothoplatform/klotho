(call_expression ;; router.Use(..)
	(selector_expression
		(identifier) @router_name
    (field_identifier) @method
    (#eq? @method "Use")
	)
  (argument_list) @args
)@call_expression
