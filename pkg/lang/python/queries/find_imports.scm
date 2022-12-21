[
  (import_statement [
      name: (dotted_name) @module
      name: (aliased_import
        name: (dotted_name) @aliasedModule
        alias: (identifier) @alias)
    ]
  ) @standardImport
  (import_from_statement
    . module_name: [
                     (dotted_name) @module
                     (relative_import . (import_prefix) @importPrefix (dotted_name) ? @module)
                     ]
    [
      name: (dotted_name) @attribute
      name: (aliased_import
              name: (dotted_name) @attribute
              alias: (identifier) @alias)

      ]
    ) @fromImport
  ] @importStatement
