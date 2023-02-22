(using_directive . ;;; [global] using [static] A[.B]; OR using C = A[.B];
  "global" ? @global
  .
  "using"
  .
  "static" ? @static
  .
  [
    (qualified_name) @identifier ;;; A.B
    (identifier) @identifier ;;; A
    ((name_equals (identifier) @alias) ;;; C = A.B
      .
      (
        [(qualified_name)
          (identifier)]
        ) @identifier
      )
    ]) @using_directive