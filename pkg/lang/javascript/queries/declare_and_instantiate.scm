(
  [
    (assignment_expression ;; exports.MyEmitter = new EventEmitter()
      left: [
        (member_expression
          object: (identifier) @var.obj
          property: (property_identifier) @name
        )
        (identifier) @name
      ] @var
      right: (new_expression
        constructor: [
          (member_expression ;; new events.EventEmitter()
            object: (identifier) @ctor.obj
            property: (property_identifier) @type
          )
          (identifier) @type ;; new EventEmitter()
          ]
      ) @value
    )

    (variable_declarator ;; const MyEmitter = new EventEmitter()
      name: (identifier) @name
      value: (new_expression
        constructor: [
          (member_expression ;; new events.EventEmitter()
            object: (identifier) @ctor.obj
            property: (property_identifier) @type
          )
          (identifier) @type ;; new EventEmitter()
          ]
      )
    )
  ] @assignment
)
