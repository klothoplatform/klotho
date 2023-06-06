;Finds all "foo.bar" usages
; This includes:
; • foo.bar
; • foo.bar()
; • foo.bar.fizz — in this case, we'll match {obj_name=foo, attr_name=bar}
; • foo.bar.fizz.buzz — we'll still match {obj_name=foo, attr_name=bar}
(attribute
  object: (identifier) @obj_name
  attribute: (identifier) @attr_name
)
