(using_directive . [
                     (qualified_name) @identifier
                     (identifier) @identifier
                     ((name_equals (identifier) @alias)
                       .
                       (
                         [(qualified_name)
                           (identifier)]
                         ) @identifier
                       )
                     ]) @using_directive