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
      (#match? @method "OpenBucket")
      )
  )
)@expression ;; bucket, err := fileblob.OpenBucket(myDir, myConnector)
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
      (#match? @method "OpenBucket")
      )
  )
)@expression ;; bucket, err = fileblob.OpenBucket(myDir, myConnector)
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
          (#match? @method "OpenBucket")
        )
      )
    )
)@expression ;; var bucket, err = fileblob.OpenBucket(myDir, myConnector)
]

