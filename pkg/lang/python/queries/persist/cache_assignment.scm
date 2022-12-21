(expression_statement
  . (assignment
      left: (identifier) @name
      right:
      (call
        [
          function: (identifier) @function
          function: (attribute object: (_) @functionHost attribute: (identifier) @function)
          ]
        arguments: (argument_list) @args)
      )) @expression
