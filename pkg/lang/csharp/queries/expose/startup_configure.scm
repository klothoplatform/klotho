(class_declaration body:
  (declaration_list (
                      (method_declaration
                        name : (_) @method_name (#eq? @method_name "Configure")
                        parameters: (parameter_list
                                      .
                                      (parameter
                                        type: (_) @param_type (#match? @param_type "^(Microsoft.AspNetCore.Builder.)?IApplicationBuilder$")
                                        name: (_) @param_name
                                        )
                                      .
                                      (parameter
                                        type: (_) @param2_type (#match? @param2_type "^(Microsoft.AspNetCore.Hosting.)?IWebHostEnvironment$")
                                        )
                                      .

                                      )
                        ) @method_declaration
                      )
    )) @class_declaration