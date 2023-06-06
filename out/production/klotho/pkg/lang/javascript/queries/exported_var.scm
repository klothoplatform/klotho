(
  (assignment_expression 
		left: (member_expression
      object: (identifier) @obj
      property: (property_identifier) @prop
    )
		right: (_) @right
	) @assign
  (#match? @obj "^exports$") ;; not supported in go-tree-sitter
)

;; TODO: not supported syntaxes:
;; - module.exports.a = _
;; - module.exports = {a: _} or {a}
;; - exports = {a: _} or {a}
