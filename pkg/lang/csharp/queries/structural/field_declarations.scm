(field_declaration
  (variable_declaration
    type: (_) @type
    (variable_declarator
      (identifier) @name
      (equals_value_clause) ? @equals_value_clause
      ) @variable_declarator
    )
  ) @field_declaration