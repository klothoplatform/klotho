(decorator
  (call
    function: (attribute
                object: (identifier) @appName
                attribute: (identifier) @verb
                )
    arguments: [
                 (argument_list . (string) @path) ; e.g. ("/path")
                 (argument_list
                   (keyword_argument
                     name: (identifier) @argname
                     value: (string) @path
                     )
                   ) ; e.g. (path="/path") or (other="other", path="/path")
                 ]
    )
  )
