;;; TODO: parse C# enumerations with this query
(enum_declaration
  name: (_) @name
  bases: (_) ? @bases
  body: (_) @body
  ) @enum_declaration