;; Finds all the imports
;; @statement - the whole import statement
;; @name - the module name as used by the importer
;; @module - the module source name
;; @relative_to - (optional) the relative import prefix
;; @attribute - (optional) the attribute name when using 'import .. from ..'
[
  (import_statement 
    name: [
      (dotted_name) @name @module
      (aliased_import
        name: (dotted_name) @module
        alias: (identifier) @name
      )
    ]
  )
  (import_from_statement
    . module_name: [
      (dotted_name) @name @module
      (relative_import .
        (import_prefix) @relative_to
        (dotted_name) ? @module
      )
    ]
    [
      name: (dotted_name) @attribute
      name: (aliased_import
        name: (dotted_name) @attribute
        alias: (identifier) @name)

    ]
  )
] @statement
