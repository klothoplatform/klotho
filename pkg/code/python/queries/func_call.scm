;; Finds function calls
;; @statement - the whole function call
;; @name - the name of the function
;; @object - (optional) the object the function is called on
(call
  function: [
    (identifier) @name
    (attribute
      object: (_) @object
      attribute: (identifier) @name 
    )
  ]
) @statement
