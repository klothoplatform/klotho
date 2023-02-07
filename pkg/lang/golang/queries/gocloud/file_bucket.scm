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
)@expression

;; bucket, err := fileblob.OpenBucket(myDir, myConnector)
