(invocation_expression ;;; e.g. var.MapGet("/path", () => {});
  function:
  (member_access_expression expression: (_) @var
    name: (_) @method_name (#match? @method_name "^Map(Get|Post|Put|Delete)?$")
    )
  arguments: (argument_list
               . (argument
                   [(string_literal)(verbatim_string_literal)] @path
                   )
               )
  )