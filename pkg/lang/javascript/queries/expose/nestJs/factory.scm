[
    (variable_declarator ;; const app = await core_1.NestFactory.create(app_module_1.AppModule); const app = await NestFactory.create(AppModule);
        name: (identifier) @var
        value: (await_expression
                (call_expression 
                    function: [
                        (member_expression
                            object: (identifier) @import
                        ) @call 
                        (member_expression
                            (member_expression
                                object: (identifier) @import
                            ) 
                        ) @call 
                    ] 
                    arguments: [
                        (arguments)
                        (arguments
                            (member_expression
                                (identifier) @id
                                (property_identifier) @moduleProp
                            ) @member
                        )
                    ] @args
            )
        )
    )
    (assignment_expression ;; app = await core_1.NestFactory.create(app_module_1.AppModule); app = await NestFactory.create(AppModule);
        left: (identifier) @var
        right: (await_expression
                (call_expression 
                    function: [
                        (member_expression
                            object: (identifier) @import
                        ) @call 
                        (member_expression
                            (member_expression
                                object: (identifier) @import
                            ) 
                        ) @call 
                    ] 
                    arguments: [
                        (arguments)
                        (arguments
                            (member_expression
                                (identifier) @id
                                (property_identifier) @moduleProp
                            ) @member
                        )
                    ] @args
            )
        )
    )
]
