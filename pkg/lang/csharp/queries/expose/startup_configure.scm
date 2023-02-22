(class_declaration body: ; finds Configure(T1 param1, T2 param2){...} method declaration in a class declaration's body
  (declaration_list (
                      (method_declaration
                        name : (_) @method_name (#eq? @method_name "Configure")
                        parameters: (parameter_list
                                      .
                                      (parameter
                                        type: (_) @param1_type
                                        name: (_) @param_name
                                        )
                                      .
                                      (parameter
                                        type: (_) @param2_type
                                        )
                                      .
                                      )
                        ) @method_declaration
                      )
    )) @class_declaration
