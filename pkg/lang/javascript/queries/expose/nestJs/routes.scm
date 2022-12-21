[
    (expression_statement
        (call_expression
            function: (identifier)
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
                        arguments: [
                            (arguments
                                (string) @path
                            )
                            (_)
                        ] 
                    )
                )
                (member_expression
                    object: (identifier) @controller
                )
                (string) @function
            )
        )
    )
]

;;__decorate([
;;    (0, common_1.Get)(':id'),
;;    __metadata("design:type", Function),
;;    __metadata("design:paramtypes", []),
;;    __metadata("design:returntype", String)
;;], OrgController.prototype, "getOrg", null);

