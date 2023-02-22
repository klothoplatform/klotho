;;; TODO: parse C# properties with this query
(property_declaration ;;; [public ...] int MyProperty { get {...} set{...} init{...} } OR
  type: (_) @type
  [accessors:
    (accessor_list
      (accessor_declaration "get") ? @get ;;; get {...}
      (accessor_declaration "set") ? @set ;;; set {...}
      (accessor_declaration "init") ? @init ;;; init {...}
      )
    value: (_) @value ;;; => value -- (arrow function body)
    ]
  ) @property_declaration