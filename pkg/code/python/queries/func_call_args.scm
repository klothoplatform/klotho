;; Finds function calls and their arguments
;; @statement - the whole function call
;; @name - the name of the function
;; @object - (optional) the object the function is called on
;; @arg - (optional) the whole argument
;; @arg.name - (optional) the name of the argument, for named args
;; @arg.value - (optional) the value of the argument. Always present if @arg is present
(call
  function: [
    (identifier) @name
    (attribute
      object: (_) @object
      attribute: (identifier) @name 
    )
  ]
  arguments: (argument_list [
    (_) @arg @arg.value
    (keyword_argument
      name: (identifier) @arg.name
      value: (_) @arg.value)
  ])?
) @statement
