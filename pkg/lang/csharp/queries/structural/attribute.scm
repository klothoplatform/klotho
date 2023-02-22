;;; This query captures C# attributes independent of specific attribute lists
(attribute name: (identifier) @attr_name ;;; Attr("v1", "v2", Arg3 = "v3") -- parentheses and their contents are optional
  (attribute_argument_list
    (attribute_argument
      .
      (name_equals (identifier) @arg_name) ? ;;; Arg =
      .
      (_) @arg_value
      .
      )
    ) ?
  )