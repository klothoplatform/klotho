package pulumi_aws

import (
	"embed"

	"github.com/klothoplatform/klotho/pkg/templateutils"
)

//go:embed *.tmpl *.ts *.json iac/*.ts iac/k8s/* iac/sanitization/*
var files embed.FS

var index = templateutils.MustTemplate(files, "index.ts.tmpl")
var pulumiBase = templateutils.MustTemplate(files, "Pulumi.yaml.tmpl")
var pulumiStack = templateutils.MustTemplate(files, "Pulumi.dev.yaml.tmpl")
