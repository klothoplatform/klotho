// Package iac2 provides the [compiler.IaCPlugin] for our AWS Pulumi implementation. It consists of a few parts.
//
//   - a templates directory, with templates for IaC fragments
//   - a ResourceCreationTemplate, which represents one parsed template
//   - a TemplatesCompiler, which takes a graph of `core.Resources` and renders it into Pulumi IaC using the templates
//
// # Templates
//
// Within the templates directory are subdirectories, one per template. Each template directory's name is the name of
// a provider struct in lower snake case. For example, a struct named FizzBuzz would have a directory named fizz_buzz.
// Within each directory are a factory.ts file, a package.json, and a package-lock.json. This means each template
// directory is a full, self-contained TypeScript program (albeit not a very useful one on its own).
//
// The factory.ts file should contain:
//
//   - imports (as any TypeScript file owuld need)
//   - An `interface Args`
//   - A `function create(args: Args): YourType`. The function name (create) and argument (arg: Args) must be exactly
//     those, but the return type (YourType) should correspond to whatever the fragment actually returns.
//
// For example:
//
//	import * as aws from '@pulumi/aws'
//
//	interface Args {
//	  Name: string
//	}
//
//	function create(args: Args): aws.iam.Role {
//	  return new aws.iam.Role(args.Name);
//	}
//
// # ResourceCreationTemplate
//
// ResourceCreationTemplate is a parsed representation of one of these factory.ts files. It contains:
//
//  1. ResourceCreationSignature.InputTypes, a map that corresponds to the Args interface. Its keys are the interface
//     field names, and the values are the field types, literally as they appear in the source. There is no
//     attempt to resolve the types using imports.
//  2. ResourceCreationSignature.OutputType, a string which corresponds to the `create` function's return type. As with
//     InputTypes, this is just the literal string in the source file.
//  3. ResourceCreationTemplate.ExpressionTemplate, a string which corresponds to the return statement's expression.
//     Any `args.Foo` reference is replaced with a Go-[text/template] string of `{{.Foo}}`. You can think of the
//     factory.ts as a "template for a template". and this field as the template derived from that file.
//  4. Imports, a set of string representing the TypeScript imports. Again, these are taken literally from the source.
//
// Template input values in a factory.ts file are wrapped in structs implementing the templateValue interface.
//
// To access an input's raw values in the factory template, invoke its `Raw` function.
//
// For Example:
//
//	const tableName = {{ .Table.Raw.Name }};
//
// To manually access an input's rendered value in a factory template, invoke the `parseTS` template function:
//
// For Example:
//
//	const table = {{ renderTS .Table }}
//
// # Nested Struct Rendering
//
// Within a Resource struct, the default TemplatesCompiler rendering behavior allows only fields of primitive types,
// slices, maps, and structs implementing the Resource or IacValue interfaces.
// This behavior can be modified with the following using the following tas on struct fields:
//   - `render:"document"` - tells TemplatesCompiler to render a field as a TypeScript object
//   - `render:"template"` - tells TemplatesCompiler to render a field using a standard Go-[text/template]
//     with a filename matching the nested struct's name in lower-snake-case format
//     in the same directory as the parent resource's factory.ts file.
//
// # TemplatesCompiler
//
// The TemplatesCompiler is responsible for putting all of the above together, along with the resources graph. It
// traverses the graph in topological order, and renders each node by fetching the ResourceCreationTemplate for that
// node's struct and passing the node into the ExpressionTemplate.
//
// # Why a template for a template?
//
// As mentioned above, the factory.ts is a template-for-a-template:
//
//	╭────────────────────────────────╮   ╭─────────────────────────────────────────╮   ╭──────────────────────────────╮
//	│ // factory.ts                  │ → │ ExpressionTemplate                      │ → │ // with Role{Name: "hello"}  │
//	│ return aws.iam.Role(args.Name) │ → │ return aws.iam.Role(parseTS {{.Name}} ) │ → │ return aws.iam.Role("hello") │
//	╰────────────────────────────────╯   ╰─────────────────────────────────────────╯   ╰──────────────────────────────╯
//
// This approach lets us do various checks at compile/unit-test time:
//
//   - Because each template directory is a full, valid program, we can verify that it is free of syntax errors, that
//     the types all match up, and that its imports and package.json dependencies are correct.
//   - We can easily check that all fields in the `interface Args` are also in the corresponding Go struct. Since we
//     know that all the usages of `args` within the return expression correspond to fields within the Args interface
//     (since the TypeScript compiles), this tells us that the Go struct does indeed have all the fields that the
//     TypeScript needs.
//   - We can validate that one struct's outputs match the others inputs.
//
// For that last one, imagine two provider structs:
//
//	type SomethingProvider struct { /* ... * }
//
//	type SomethingUser struct {
//		Something SomethingProvider
//	}
//
// When we look at something_user/factory.ts, it will contain something like:
//
//	interface Args {
//	  Something: fizz.buzz.Something,
//	}
//
//	return new foo.bar.SomethingUser({something: args.Something});
//
// Since the TypeScript compiles, we can be confident that the SomethingUser options really do include a field named
// `something` with a type `fizz.buzz.Something`. But when this is all rendered, that `args.Something` field will come
// from a variable whose type is... whatever SomethingProvider's `create(...)` returns. So to be sure that everything
// matches, we can write a test that confirms that if SomethingProvider's Args.Something is a `fizz.buzz.Something`,
// and that it comes from the Go struct SomethingProvider.Something, then we should look at that struct's Something
// field, look up template, and confirm that its ResourceCreationSignature.OutputType is `fizz.buzz.Something`.
//
// Combined, all of these tests should give us confidence that all pieces will fit together, without having to write
// integration tests for each combination.
package iac2
