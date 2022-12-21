(assignment_expression 
    left: (identifier) @name
    right: (call_expression
        function: (_) @func
        arguments: (arguments
            (array
                (call_expression
                    function: (parenthesized_expression
                        (sequence_expression
                            right: (member_expression
                                object: (identifier) @import
                                property: (property_identifier) @method
                            )
                        )
                    )
                    arguments: (arguments
                        (string) @basePath
                    )
                ) @call
            )
        ) @args
    )
)

;; UsersController = __decorate([
;;    (0, common_1.Controller)('users'),
;;    __metadata("design:paramtypes", [app_service_1.AppService])
;; ], UsersController);

