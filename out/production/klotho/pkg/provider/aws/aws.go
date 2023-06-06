// Package aws provides the [compiler.ProviderPlugin] to generate architectures on AWS.
//
// Within the package, in the resources subdirectories, the provider contains an internal representation of all
// aws resources (resource is defined as something which can be represented by an arn).
// These internal representations all implement the [core.Resource] interface so that they can be added to the [core.ResourceGraph]
package aws
