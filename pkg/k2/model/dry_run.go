package model

type DryRun int

const (
	// DryRunNone is the default value, no dry run
	DryRunNone DryRun = iota

	// DryRunPreview is a dry run that uses Pulumi preview
	DryRunPreview

	// DryRunCompile is a dry run that only runs `tsc` on the resulting IaC
	DryRunCompile

	// DryRunFileOnly is a dry run that only writes the files to disk
	DryRunFileOnly
)
