(invocation_expression ;;; var.useEndpoints(endpoints => {...})
  function: (member_access_expression
              expression: (_) @var (#eq? @var "%s")
              name: (_) @name (#eq? @name "UseEndpoints")

              )
  arguments: (argument_list
               (argument
                 (lambda_expression ;;; TODO: support local or anonymous functions
                   (identifier) @endpoints_param
                   )
                 )
               )
  ) @expression