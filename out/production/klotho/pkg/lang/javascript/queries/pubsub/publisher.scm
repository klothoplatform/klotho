(
  (call_expression
    function: (member_expression
      object: [
        (identifier)
        (member_expression)
      ] @emitter
      property: (property_identifier) @func
    )
    arguments: (arguments
      .
      (string) @topic
    )
  )
  (#match? @func "^emit$") ;; not supported in go-tree-sitter
)
