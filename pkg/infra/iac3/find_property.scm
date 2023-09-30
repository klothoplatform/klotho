(return_statement
  (object
    (pair
      key: [
        (property_identifier) @key ;; CidrBlock: object.cidrBlock
        (string                    ;; 'key with spaces': object.something
          (string_fragment) @key
        )
      ]
      value: (_) @value
    )
  )
)
