(class_declaration body:
  (declaration_list (
                      (method_declaration
                        name : (_) @method_name (#eq? @method_name "Configure")
                        parameters: (parameter_list (
                                                      (parameter
                                                        type: (_) @param_type (#eq? @param_type "IApplicationBuilder")
                                                        name: (_) @param_name
                                                        )
                                                      )
                                      )
                        ) @method_declaration
                      )
    )) @class_declaration