;; look for an "interface Args" at the top-level of the source,
;; and find its property names and types (e.g. "foo: string,"

(program
  (interface_declaration
    name: (type_identifier) @interface_name
    body: (object_type
      (property_signature
        name: (property_identifier) @property_name
        type: (type_annotation (_) @property_type)
      ) @property
    )
  )
  (#eq? @interface_name "Args")
)
