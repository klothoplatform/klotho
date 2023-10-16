(program
  (import_statement
    source: (string)@source
  ) @import
  (#not-eq? @source "'../../wrappers'")
  (#not-eq? @source "'../../globals'")
)
