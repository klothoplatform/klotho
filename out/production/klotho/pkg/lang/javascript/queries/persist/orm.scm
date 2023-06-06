
[
    (variable_declarator
        name: (identifier)@name
        value: (new_expression
            constructor: [
                (member_expression ;; new sequelize.Sequelize()
                    object: (identifier) @ctor.obj
                    property: (property_identifier) @type
                )
                (identifier) @type
            ]
            arguments: (arguments (_)@argstring)
        ) @expression
    )
    (assignment_expression ;; exports.client = new Sequelize()
      left: [
        (member_expression
          object: (identifier) @var.obj
          property: (property_identifier) @name
        )
        (identifier) @name
      ] @var
      right: (new_expression
        constructor: [
          (member_expression ;; new sequelize.Sequelize()
            object: (identifier) @ctor.obj
            property: (property_identifier) @type
          )
          (identifier) @type  ;; new Sequelize()
          ]
        arguments: (arguments (_)@argstring)
      ) @value
    )
]