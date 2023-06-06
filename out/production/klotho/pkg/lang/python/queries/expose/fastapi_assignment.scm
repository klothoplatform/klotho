(expression_statement
  (assignment
    left: (identifier) @identifier
    right: (call
             function: (identifier) @function
             arguments: (
                          argument_list
                          (keyword_argument
                            name: (identifier) @arg
                            value: (string) @val
                            ) ?
                          ) ?
             )
  )
) @expression






