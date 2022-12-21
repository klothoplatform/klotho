; Finds the function name and arguments of all function calls
(call function: (_) @function
  arguments:
  (argument_list [
                   (_) @arg
                   (keyword_argument
                     name: (identifier) @argName
                     value: (_) @arg)
                   ]) ?)
