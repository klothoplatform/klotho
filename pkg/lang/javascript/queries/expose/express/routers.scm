(call_expression
	function: (member_expression
				object: (_) @obj
				property: (_) @prop
			)
	arguments: [
		(arguments
			[
				(identifier) @mwObj
				(member_expression
					object: (identifier) @mwObj
					property: (property_identifier) @mwProp
				)
				(call_expression ;; app.use('/', server.getMiddleware());
                	function: (member_expression
						object: (identifier) @mwObj
						property: (property_identifier) @mwProp
					)
                )
			]
		)
		(arguments
			(string) @path
			[
				(identifier) @mwObj
				(member_expression
					object: (identifier) @mwObj
					property: (property_identifier) @mwProp
				)
				(call_expression ;; app.use('/', server.getMiddleware());
                	function: (member_expression
						object: (identifier) @mwObj
						property: (property_identifier) @mwProp
					)
                )
			]
		)
	]
) @expr

;; TODO check if `mwObj` is defined/initialized as a Router. Currently, this will capture
;; all middleware, leading to some potentially incorrect behaviour.