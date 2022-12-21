[
    (variable_declarator ;; const client = createClient({url: ""}); const client = (0, redis_1.createClient)();
        name: (identifier)@name
        value: (call_expression
            function: [
                (identifier) @method ;; createClient
                (parenthesized_expression ;; (0, redis_1.createClient)
                    (sequence_expression
                        left: [
                            (_)
                        ]
                        right: [
                            (member_expression
                                object: (identifier) @obj
                                property: (property_identifier) @method
                            )
                        ]
                    )
                )
            ]
            arguments: (_) @argstring ;; ({url: ""}) ()
        ) @expression
    )
    (assignment_expression ;; exports.client = createClient({url: ""}); client = (0, redis_1.createClient)()
        left: [
            (identifier)@name
            (member_expression ;; exports.client
                object: (identifier) @var.obj
                property: (property_identifier) @name
            )
        ]
        right: [
            (call_expression
                function: [
                    (identifier) @method
                    (parenthesized_expression ;; (0, redis_1.createClient)
                        (sequence_expression
                            left: [
                                (_)
                            ]
                            right: [
                                (member_expression
                                    object: (identifier) @obj
                                    property: (property_identifier) @method
                                )
                            ]
                        )
                    )
                ]
                arguments: (_) @argstring ;; ({url: ""}) ()
            ) @expression
        ]
    )
]