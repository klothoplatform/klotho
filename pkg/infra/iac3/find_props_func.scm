(program
  (function_declaration
    name: (identifier) @function_name
    parameters: (formal_parameters
      (required_parameter
        pattern: (identifier) @object.name
        type: (type_annotation (_) @object.type)
      )
      (required_parameter
        pattern: (identifier) @args.name
        type: (type_annotation (_) @args.type)
      )
    )
    body: (statement_block) @body
  )
  (#eq? @function_name "properties")
  (#eq? @object.name "object")
  (#eq? @args.name "args")
)
