(indexer_declaration
  type: (_) @type
  [accessors:
    (accessor_list
      (accessor_declaration)? @get (#match? @get "^get")
      (accessor_declaration)? @set (#match? @set "^set")
      )
    value: (_) @value
    ]
  ) @indexer_declaration