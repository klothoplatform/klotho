(attribute name: (identifier) @attr (#match? @attr "^Http(Get|Put|Post|Delete|Patch|Head)$")
  (attribute_argument_list (
                             ((attribute_argument . (string_literal)) @template) ?
                             ((attribute_argument (name_equals (identifier) @order_id)) ? @order_arg
                               (#eq? @order_id "Order"))
                             ((attribute_argument (name_equals (identifier) @name_id)) ? @name_arg
                               (#eq? @name_id "Name"))

                             )
    ) ?
  )