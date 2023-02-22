(class_declaration body: ;;; finds method declaration ConfigureServices(T param){...} in a class declaration's body
  (declaration_list (
                      (method_declaration
                        name : (_) @method_name (#eq? @method_name "ConfigureServices")
                        parameters: (parameter_list
                                      .
                                      (parameter
                                        type: (_) @param_type
                                        name: (_) @param_name
                                        )
                                      .
                                      )
                        ) @method_declaration
                      )
    )) @class_declaration