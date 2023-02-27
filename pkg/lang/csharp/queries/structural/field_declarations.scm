[
  (field_declaration ;;; [public ...] int [x = 1][, y = 2];
  (variable_declaration
    type: (_) @type
    (variable_declarator ;;; int x = 1[, y = 2];
      (identifier) @name
      (equals_value_clause) ? @equals_value_clause
      ) @variable_declarator
    )
  )
(event_field_declaration ;;; [public ...] event SomeDelegate MyEvent;
  (variable_declaration
    type: (_) @type
    (variable_declarator
      (identifier) @name
      ) @variable_declarator
    )
  )
] @field_declaration