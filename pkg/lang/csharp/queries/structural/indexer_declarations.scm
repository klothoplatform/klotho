;;; TODO: parse C# indexers with this query
(indexer_declaration
  type: (_) @type
  parameters: (_) @parameters
  [accessors:
    (accessor_list
      (accessor_declaration "get")? @get ;;; get {...}
      (accessor_declaration "set")? @set ;;; set {...}
      )
    value: (_) @value ;;; => arr[i] (arrow function body)
    ]
  ) @indexer_declaration