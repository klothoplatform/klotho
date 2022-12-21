[
    (assignment_expression 
        left: (identifier) @name
        right: (call_expression
            function: (_) @func
            arguments: (arguments
                (array
                    (call_expression
                        (parenthesized_expression
                            (sequence_expression
                                right: (member_expression
                                    object: (identifier) @import
                                    property: (property_identifier) @method
                                )
                            )
                        )
                        (arguments
                            (object
                            	(pair
                                	key: (property_identifier) @pairKey
                                    value: (array
                                    	(member_expression
                                        	object: (identifier) @controllerImport
                                            property: (property_identifier) @controllerName
                                        )@controllers
                                    )
                                )
                                (#match? @pairKey "^controllers$")
                            )
                        )
                    )
                )
                (identifier) @moduleName
            )
        )
    )
]

;; AppModule = __decorate([
;;    (0, common_1.Module)({
;;        imports: [],
;;        controllers: [app_controller_1.UsersController, app_controller_1.OrgController],
;;        providers: [app_service_1.AppService],
;;    })
;; ], AppModule);

;; Will grab controller names and match to controllers