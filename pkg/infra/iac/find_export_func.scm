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
      (required_parameter
        pattern: (identifier) @props.name
        type: (type_annotation (_) @props.type)
      )
    )
    body: (statement_block) @body
  )
  (#eq? @function_name "infraExports")
  (#eq? @object.name "object")
  (#eq? @args.name "args")
  (#eq? @props.name "props")
)
