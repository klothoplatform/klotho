(program
  (function_declaration
    name: (identifier) @function_name
    parameters: (formal_parameters
      .
      (required_parameter
        pattern: (identifier)
        type: (type_annotation (_) @input_type)
      )
      .
    )
    return_type: (type_annotation (_) @return_type)
    body: (statement_block
      (return_statement (_) @return_body)
    )
  )
  (#eq? @function_name "create")
  (#eq? @input_type "Args")
)
