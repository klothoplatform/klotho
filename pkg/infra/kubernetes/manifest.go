package kubernetes

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/yaml"
	sanitize "github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	"github.com/klothoplatform/klotho/pkg/templateutils"
)

//go:embed manifests/*
var files embed.FS

var deployment = templateutils.MustTemplate(files, "manifests/deployment.yaml.tmpl")
var horizontalPodAutoscaler = templateutils.MustTemplate(files, "manifests/horizontal_pod_autoscaler.yaml.tmpl")
var serviceAccount = templateutils.MustTemplate(files, "manifests/service_account.yaml.tmpl")
var service = templateutils.MustTemplate(files, "manifests/service.yaml.tmpl")
var serviceExport = templateutils.MustTemplate(files, "manifests/service_export.yaml.tmpl")
var targetGroupBinding = templateutils.MustTemplate(files, "manifests/target_group_binding.yaml.tmpl")

const (
	MANIFEST_TYPE = "manifest"
)

type (
	Manifest struct {
		Name             string
		ConstructRefs    []core.AnnotationKey
		FilePath         string
		Transformations  map[string]core.IaCValue
		ClustersProvider core.IaCValue
	}

	DeploymentManifestData struct {
		Name               string
		ExecUnitName       string
		FargateEnabled     string
		ReplicaCount       string
		ServiceAccountName string
		Image              string
		Namespace          string
	}
)

// KlothoConstructRef returns a slice containing the ids of any Klotho constructs is correlated to
func (manifest *Manifest) KlothoConstructRef() []core.AnnotationKey { return manifest.ConstructRefs }

func (manifest *Manifest) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     MANIFEST_TYPE,
		Name:     manifest.Name,
	}
}

func addDeploymentManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := DeploymentManifestData{
		Name:               sanitize.MetadataNameSanitizer.Apply(unit.Name),
		ExecUnitName:       unit.Name,
		Namespace:          unit.Namespace,
		ServiceAccountName: unit.getServiceAccountName(),
	}
	buf := new(bytes.Buffer)
	err := deployment.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", deployment.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-deployment.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", deployment.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.Deployment = newF
	return nil
}

type HorizontalPodAutoscalerManifestData struct {
	Name         string
	ExecUnitName string
}

func addHorizontalPodAutoscalerManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := HorizontalPodAutoscalerManifestData{
		Name:         sanitize.MetadataNameSanitizer.Apply(unit.Name),
		ExecUnitName: unit.Name,
	}
	buf := new(bytes.Buffer)
	err := horizontalPodAutoscaler.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", horizontalPodAutoscaler.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-horizontal-pod-autoscaler.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", horizontalPodAutoscaler.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.HorizontalPodAutoscaler = newF
	return nil
}

type ServiceAccountManifestData struct {
	Name      string
	Namespace string
	IRSA      bool
}

func GenerateServiceAccountManifest(name string, namespace string, irsa bool) (string, []byte, error) {
	saName := sanitize.MetadataNameSanitizer.Apply(name)
	data := ServiceAccountManifestData{
		Name:      saName,
		Namespace: namespace,
		IRSA:      irsa,
	}
	buf := new(bytes.Buffer)
	err := serviceAccount.Execute(buf, data)
	if err != nil {
		return saName, nil, core.WrapErrf(err, "error executing template %s", serviceAccount.Name())
	}
	return saName, buf.Bytes(), nil
}

func addServiceAccountManifest(kch *HelmChart, unit *HelmExecUnit) error {
	_, buf, err := GenerateServiceAccountManifest(unit.Name, unit.Namespace, false)
	if err != nil {
		return err
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-serviceaccount.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", serviceAccount.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.ServiceAccount = newF
	return nil
}

type ServiceManifestData struct {
	Name         string
	ExecUnitName string
	Namespace    string
}

func addServiceManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := ServiceManifestData{
		Name:         sanitize.MetadataNameSanitizer.Apply(unit.Name),
		ExecUnitName: unit.Name,
		Namespace:    unit.Namespace,
	}
	buf := new(bytes.Buffer)
	err := service.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", service.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-service.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", service.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.Service = newF
	return nil
}

type TargetGroupBindingManifestData struct {
	ServiceName string
}

func addTargetGroupBindingManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := TargetGroupBindingManifestData{
		ServiceName: unit.getServiceName(),
	}
	buf := new(bytes.Buffer)
	err := targetGroupBinding.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", targetGroupBinding.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-targetgroupbinding.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", targetGroupBinding.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.TargetGroupBinding = newF
	return nil
}

type ServiceExportManifestData struct {
	ServiceName string
	Namespace   string
}

func addServiceExportManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := ServiceExportManifestData{
		ServiceName: unit.getServiceName(),
		Namespace:   unit.Namespace,
	}
	buf := new(bytes.Buffer)
	err := serviceExport.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", serviceExport.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-serviceexport.yaml", kch.Name, unit.Name), buf)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", serviceExport.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.ServiceExport = newF
	return nil
}
