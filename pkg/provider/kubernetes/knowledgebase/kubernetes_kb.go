package knowledgebase

import (
	"fmt"
	"path"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	k8sSanitizer "github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

var KubernetesKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Deployment]{
		Configure: func(service *resources.Service, deployment *resources.Deployment, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Object == nil {
				return fmt.Errorf("service %s has no object", service.Name)
			}
			if deployment.Object == nil {
				return fmt.Errorf("%s has no object", deployment.Id())
			}
			service.Object.Spec.Selector = resources.KlothoIdSelector(deployment.Object)

			if err := service.MapContainerPorts(deployment.Object.Name, deployment.Object.Spec.Template.Spec.Containers); err != nil {
				return err
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Pod]{
		Configure: func(service *resources.Service, pod *resources.Pod, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Object == nil {
				return fmt.Errorf("%s has no object", service.Id())
			}
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			service.Object.Spec.Selector = resources.KlothoIdSelector(pod.Object)
			if err := service.MapContainerPorts(pod.Object.Name, pod.Object.Spec.Containers); err != nil {
				return err
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Namespace]{
		Configure: func(pod *resources.Pod, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(pod, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Namespace]{
		Configure: func(service *resources.Service, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(service, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Namespace]{
		Configure: func(deployment *resources.Deployment, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(deployment, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.ServiceAccount, *resources.Namespace]{
		Configure: func(serviceAccount *resources.ServiceAccount, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(serviceAccount, namespace)
		},
	}, knowledgebase.EdgeBuilder[*resources.PersistentVolume, *resources.Namespace]{
		Configure: func(persistentVolume *resources.PersistentVolume, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(persistentVolume, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.PersistentVolumeClaim, *resources.Namespace]{
		Configure: func(persistentVolumeClaim *resources.PersistentVolumeClaim, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(persistentVolumeClaim, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.StorageClass, *resources.Namespace]{
		Configure: func(storageClass *resources.StorageClass, namespace *resources.Namespace, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			return SetNamespace(storageClass, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.ServiceAccount]{
		Configure: func(pod *resources.Pod, serviceAccount *resources.ServiceAccount, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			if serviceAccount.Object == nil {
				return fmt.Errorf("service account %s has no object", serviceAccount.Name)
			}
			pod.Object.Spec.ServiceAccountName = serviceAccount.Object.GetName()
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.ServiceAccount]{
		Configure: func(deployment *resources.Deployment, serviceAccount *resources.ServiceAccount, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Object == nil {
				return fmt.Errorf("deployment %s has no object", deployment.Name)
			}
			if serviceAccount.Object == nil {
				return fmt.Errorf("service account %s has no object", serviceAccount.Name)
			}
			deployment.Object.Spec.Template.Spec.ServiceAccountName = serviceAccount.Object.GetName()
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.TargetGroupBinding, *resources.Service]{
		Configure: func(targetGroupBinding *resources.TargetGroupBinding, service *resources.Service, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Object == nil {
				return fmt.Errorf("%s has no object", service.Id())
			}

			// enable pod readiness gate injection for all pods associated with the service's target group
			for _, res := range dag.GetDownstreamResources(service) {
				switch res := res.(type) {
				case *resources.Pod:
					if res.Object == nil {
						return fmt.Errorf("pod %s has no object", res.Id())
					}
					if res.Object.Labels == nil {
						res.Object.Labels = map[string]string{}
					}
					res.Object.Labels["elbv2.k8s.aws/pod-readiness-gate-inject"] = "enabled"
				case *resources.Deployment:
					if res.Object == nil {
						return fmt.Errorf("deployment %s has no object", res.Id())
					}
					if res.Object.Spec.Template.Labels == nil {
						res.Object.Spec.Template.Labels = map[string]string{}
						res.Object.Spec.Template.Labels["elbv2.k8s.aws/pod-readiness-gate-inject"] = "enabled"
					}
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ServiceExport, *resources.Service]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.ServiceExport, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Manifest]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Manifest]{},
	knowledgebase.EdgeBuilder[*resources.HelmChart, *resources.ServiceAccount]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.PersistentVolume]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.PersistentVolume]{
		Configure: func(deployment *resources.Deployment, persistentVolume *resources.PersistentVolume, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Object == nil {
				return fmt.Errorf("%s has no object", deployment.Id())
			}
			if persistentVolume.Object == nil {
				return fmt.Errorf("%s has no object", persistentVolume.Id())
			}

			claim, err := construct.GetSingleDownstreamResourceOfType[*resources.PersistentVolumeClaim](dag, persistentVolume)
			if err != nil {
				return err
			}

			volumeName := k8sSanitizer.RFC1035LabelSanitizer.Apply(fmt.Sprintf("%s-volume", persistentVolume.Name))
			mountPath := path.Join("/mnt/", persistentVolume.Name)
			volumeMount := corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
			}

			volume := corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claim.Object.Name,
					},
				},
			}

			if deployment.Object.Spec.Template.Spec.Containers == nil {
				return fmt.Errorf("%s has no containers", deployment.Id())
			}
			for i, container := range deployment.Object.Spec.Template.Spec.Containers {
				containerRef := &container
				mountAdded := false
				for i, existingMount := range containerRef.VolumeMounts {
					if volumeMount.Name == existingMount.Name {
						containerRef.VolumeMounts[i] = volumeMount
						mountAdded = true
						break
					}
				}
				if !mountAdded {
					containerRef.VolumeMounts = append(containerRef.VolumeMounts, volumeMount)
				}
				deployment.Object.Spec.Template.Spec.Containers[i] = *containerRef
			}
			volumeAdded := false
			for i, existingVolume := range deployment.Object.Spec.Template.Spec.Volumes {
				if volume.Name == existingVolume.Name {
					deployment.Object.Spec.Template.Spec.Volumes[i] = volume
					volumeAdded = true
					break
				}
			}
			if !volumeAdded {
				deployment.Object.Spec.Template.Spec.Volumes = append(deployment.Object.Spec.Template.Spec.Volumes, volume)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.PersistentVolume, *resources.PersistentVolumeClaim]{
		Configure: func(persistentVolume *resources.PersistentVolume, persistentVolumeClaim *resources.PersistentVolumeClaim, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if persistentVolume.Object == nil {
				return fmt.Errorf("%s has no object", persistentVolume.Id())
			}
			if persistentVolumeClaim.Object == nil {
				return fmt.Errorf("%s has no object", persistentVolumeClaim.Id())
			}
			persistentVolumeClaim.Object.Spec.VolumeName = persistentVolume.Object.Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.PersistentVolumeClaim, *resources.StorageClass]{
		Configure: func(persistentVolumeClaim *resources.PersistentVolumeClaim, storageClass *resources.StorageClass, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if persistentVolumeClaim.Object == nil {
				return fmt.Errorf("%s has no object", persistentVolumeClaim.Id())
			}
			if storageClass.Object == nil {
				return fmt.Errorf("%s has no object", storageClass.Id())
			}
			persistentVolumeClaim.Object.Spec.StorageClassName = &storageClass.Object.Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.PersistentVolume, *resources.StorageClass]{
		Configure: func(persistentVolume *resources.PersistentVolume, storageClass *resources.StorageClass, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if persistentVolume.Object == nil {
				return fmt.Errorf("%s has no object", persistentVolume.Id())
			}
			if storageClass.Object == nil {
				return fmt.Errorf("%s has no object", storageClass.Id())
			}
			persistentVolume.Object.Spec.StorageClassName = storageClass.Object.Name
			return nil
		},
	},
)

func SetNamespace(resource resources.ManifestFile, namespace *resources.Namespace) error {
	object := resource.GetObject()
	if object == nil {
		return fmt.Errorf("%s has no object", resource.Id())
	}
	if namespace.Object == nil {
		return fmt.Errorf("%s has no resource", namespace.Id())
	}
	object.SetNamespace(namespace.Object.GetName())
	return nil
}
