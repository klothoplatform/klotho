(interface_declaration
  name: (type_identifier)@name
  body: (object_type
    [
        (property_signature
            name: (property_identifier)@property_name
            type: (type_annotation
                [
                    (_
                        (nested_type_identifier)@nested
                    ) @property_type
                    (_)@property_type
                    (generic_type
                        name: (type_identifier)@nested
                    )@property_type
                ]
            )
        )
    ]
  )
  (#eq? @name "Args")
)