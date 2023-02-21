[
  (short_var_declaration
   left: (expression_list
      	(identifier) @varName
       	(identifier)
        ) @variables
  right: (expression_list
    (call_expression
      function: (selector_expression
          operand: (identifier) @id
          field: (field_identifier) @method
          )
      arguments: (argument_list) @args
      (#match? @method "OpenVariable")
    )@call
  )
)@expression ;; v, err := runtimevar.OpenVariable(context.Background(), "my_secret.key?decoder=string")
(assignment_statement
   left: (expression_list
      	(identifier) @varName
       	(identifier)
        ) @variables
  right: (expression_list
    (call_expression
      function: (selector_expression
          operand: (identifier) @id
          field: (field_identifier) @method
          )
      arguments: (argument_list) @args
      (#match? @method "OpenVariable")
    )@call
  )
)@expression ;; v, err = runtimevar.OpenVariable(context.Background(), "my_secret.key?decoder=string")
(var_declaration
  	(var_spec
     name: (identifier) @varName
	   value: (expression_list
        (call_expression
          function: (selector_expression
            operand: (identifier) @id
            field: (field_identifier) @method
          )
          arguments: (argument_list) @args
          (#match? @method "OpenVariable")
        )@call
      )
    )
)@expression ;; var v, err = runtimevar.OpenVariable(context.Background(), "my_secret.key?decoder=string")

]

