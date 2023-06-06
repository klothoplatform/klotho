[
    (lexical_declaration ;; const app = express()
        (variable_declarator
            name: (identifier) @var
            value: (call_expression
                function: (identifier) @express
            )
        )
    )
    (expression_statement ;; exports.app = express()
        (assignment_expression
            left: (member_expression
                object: (identifier) @exports
                property: (property_identifier)
            ) @var
            right: (call_expression
                function: (identifier) @express
            )
        )
    )
    (assignment_expression ;; let app ; app = express()
        left: (identifier) @var
        right: (call_expression
            function: (identifier) @express
        )
    )
]