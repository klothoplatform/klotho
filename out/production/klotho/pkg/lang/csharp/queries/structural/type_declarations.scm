[
  (class_declaration             ;;; [public ...] class MyClass [: BaseClass, ...] {...}
    name: (identifier) @name
    bases: (base_list) ? @bases
    ) @class_declaration
  (interface_declaration         ;;; [public ...] interface Interface [: BaseInterface, ...] {...}
    name: (identifier) @name
    bases: (base_list) ? @bases
    ) @interface_declaration
  (struct_declaration            ;;; [public ...] struct MyStruct [: BaseStruct, ...] {...}
    name: (identifier) @name
    bases: (base_list) ? @bases
    ) @struct_declaration
  (record_declaration           ;;; [public ...] record MyRecord [: BaseRecord, ...] {...}
    name: (identifier) @name
    bases: (base_list) ? @bases
    ) @record_declaration
  ]