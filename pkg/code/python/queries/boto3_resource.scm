(
  (call
    function: (attribute
      object: (identifier) @object
      attribute: (identifier) @attribute
    )
    arguments: (argument_list
      (string) @type
    )
  )
  (#match? @object "boto3")
  (#match? @attribute "resource")
)
