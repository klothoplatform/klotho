package kubernetes

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/yaml"
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

type DeploymentManifestData struct {
	ExecUnitName       string
	FargateEnabled     string
	ReplicaCount       string
	ServiceAccountName string
	Image              string
	Namespace          string
}

func addDeploymentManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := DeploymentManifestData{
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
	ExecUnitName string
}

func addHorizontalPodAutoscalerManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := HorizontalPodAutoscalerManifestData{
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
	ExecUnitName string
	Namespace    string
}

func addServiceAccountManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := ServiceAccountManifestData{
		ExecUnitName: unit.Name,
		Namespace:    unit.Namespace,
	}
	buf := new(bytes.Buffer)
	err := serviceAccount.Execute(buf, data)
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", serviceAccount.Name())
	}
	newF, err := yaml.NewFile(fmt.Sprintf("%s/templates/%s-serviceaccount.yaml", kch.Name, unit.Name), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return core.WrapErrf(err, "error executing template %s", serviceAccount.Name())
	}
	kch.Files = append(kch.Files, newF)
	unit.ServiceAccount = newF
	return nil
}

type ServiceManifestData struct {
	ExecUnitName string
	Namespace    string
}

func addServiceManifest(kch *HelmChart, unit *HelmExecUnit) error {
	data := ServiceManifestData{
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
