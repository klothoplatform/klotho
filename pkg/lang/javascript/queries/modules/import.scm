
;;; commonJS require()
;;; side-effect import: require(<"module">)[.<name> ...] -- because any node type can be a parent, this expression will also match assigned imports
[
  (call_expression
  function: (identifier) @func
  arguments: (arguments . (string) @source .)
  ) @func.expr @cjs.sideEffect.requireStatement
(member_expression) @func.wrapper @sideEffect.func.wrapper
]
;;; this expression captures a duplicate of each require(<"module">).<name> to help with deduplication
(member_expression object: (call_expression
                             function: (identifier) @func
                             arguments: (arguments . (string) .)
                             )
  ) @dedup

;;; import is associated with a declared variable (<const|let|var> = <require(<"module">)[.<name>]>)
(_
  (variable_declarator
    name: [
            (identifier) @local.name
            (object_pattern
              [
                ;;; { <x> }
                (shorthand_property_identifier_pattern) @destructured.source.name
                ;;; { <x>: <y> }
                (pair_pattern
                  key: (_) @destructured.source.name
                  value: (_) @local.name
                  ) @destructuredPair
                ]
              )
            ]
    value: (
             [ ;;; require(<"module">)
               (call_expression
                 function: (identifier) @func
                 arguments: (arguments
                              . (string) @source .
                              )
                 ) @func.expr
               ;;; require(<"module">).<name>
               [(member_expression
                 object: [(call_expression
                            function: (identifier) @func
                            arguments: (arguments
                                         . (string) @source .)
                            ) @func.expr
                           ;;; require(<"module">).<name>.<field>[.<field> ...] requires additional recursive processing
                           (member_expression) @func.wrapper
                           ]
                 property: (_) @func.source.name
                 )
                 ]
               (
                 ;;; __importStar(require(<"module">))
                 (call_expression
                   function: (identifier) @ts.wrapper
                   arguments: (arguments
                                .
                                (call_expression
                                  function: (identifier) @func
                                  arguments: (arguments
                                               . (string) @source .) .
                                  ) @func.expr
                                )
                   )
                 )
               ]
             )
    ) @declarator
  ) @cjs.requireStatement

;;; ES Import
(import_statement
  .
  (import_clause
    [
      ;;; import { * as <alias> } from <"source">
      (namespace_import (identifier) @alias) @namespaceImport

      ;;; import { [<name>, ...][, <name>: <alias>, ...] } from <"source">
      (named_imports (import_specifier name: ([
                                                (identifier)
                                                (string)
                                                ]) @export alias: (identifier) ? @alias))
      ;;; alias for default export -- e.g. import <alias>[, [<*> as <name>][, { <prop>[, ...] }]] from <"source">
      (identifier) @alias
      ]
    ) ?
  (string) @source
  ) @es.importStatement
