/*
Copyright 2026 KAI Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"context"
	"fmt"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var CommonLabels = map[string]string{
	"app.kubernetes.io/name":       "kai",
	"app.kubernetes.io/managed-by": "kai-operator",
}

var DefaultPorts = map[string]int32{
	"codeserver": 8080,
	"jupyter":    8888,
	"novnc":      6080,
	"vllm":       8000,
	"tgi":        3000,
}

func BuildDeployment(name, namespace, framework, taskType, image string, replicas int32, container corev1.Container, labels map[string]string) *appsv1.Deployment {
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range CommonLabels {
		labels[k] = v
	}
	labels["framework"] = framework
	labels["kai.io/task-type"] = taskType

	deployName := fmt.Sprintf("%s-%s-%s", name, framework, namespace)
	if namespace == "" {
		deployName = fmt.Sprintf("%s-%s", name, framework)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kai.io/task-type": taskType,
					"framework":        framework,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers:    []corev1.Container{container},
				},
			},
		},
	}
}

func BuildContainer(image string, port int32, cmd []string, args []string, env []corev1.EnvVar, resources *kaiiov1alpha1.ResourceSpec) corev1.Container {
	if port == 0 {
		port = 8080
	}

	container := corev1.Container{
		Image: image,
		Name:  "main",
		Ports: []corev1.ContainerPort{{ContainerPort: port}},
	}

	if len(cmd) > 0 {
		container.Command = cmd
	}
	if len(args) > 0 {
		container.Args = args
	}
	if len(env) > 0 {
		container.Env = env
	}

	if resources != nil {
		container.Resources = BuildResources(resources)
	}

	return container
}

func BuildResources(resources *kaiiov1alpha1.ResourceSpec) corev1.ResourceRequirements {
	cpu := "500m"
	memory := "512Mi"
	gpu := "0"

	if resources != nil {
		if resources.CPU != "" {
			cpu = resources.CPU
		}
		if resources.Memory != "" {
			memory = resources.Memory
		}
		if resources.GPU != "" {
			gpu = resources.GPU
		}
	}

	cpuQty := resource.MustParse(cpu)
	memoryQty := resource.MustParse(memory)

	req := corev1.ResourceList{
		corev1.ResourceCPU:    cpuQty,
		corev1.ResourceMemory: memoryQty,
	}
	lim := corev1.ResourceList{
		corev1.ResourceCPU:    cpuQty,
		corev1.ResourceMemory: memoryQty,
	}

	if gpu != "0" {
		gpuQty, _ := resource.ParseQuantity(gpu)
		req["nvidia.com/gpu"] = gpuQty
		lim["nvidia.com/gpu"] = gpuQty
	}

	return corev1.ResourceRequirements{
		Requests: req,
		Limits:   lim,
	}
}

func BuildVolumes(volumes []kaiiov1alpha1.VolumeSpec, dataSources []kaiiov1alpha1.DataSourceSpec) ([]corev1.Volume, []corev1.VolumeMount) {
	var vols []corev1.Volume
	var mounts []corev1.VolumeMount

	for _, vol := range volumes {
		volume := corev1.Volume{Name: vol.Name}
		switch vol.Type {
		case "emptyDir":
			volume.VolumeSource = corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}
			if vol.Capacity != "" {
				size, err := resource.ParseQuantity(vol.Capacity)
				if err == nil {
					volume.VolumeSource.EmptyDir.SizeLimit = &size
				}
			}
		case "persistentVolumeClaim":
			volume.VolumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: vol.Name},
			}
		case "configMap":
			volume.VolumeSource = corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: vol.Name}},
			}
		case "secret":
			volume.VolumeSource = corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: vol.Name},
			}
		}
		vols = append(vols, volume)

		if vol.MountPath != "" {
			mounts = append(mounts, corev1.VolumeMount{Name: vol.Name, MountPath: vol.MountPath})
		}
	}

	for _, ds := range dataSources {
		vols = append(vols, corev1.Volume{
			Name:         fmt.Sprintf("datasource-%s", ds.Type),
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      fmt.Sprintf("datasource-%s", ds.Type),
			MountPath: ds.MountPath,
		})
	}

	return vols, mounts
}

func BuildService(name, namespace string, port int32, labels map[string]string) *corev1.Service {
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range CommonLabels {
		labels[k] = v
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Port: port, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: port}},
			},
		},
	}
}

func CreateService(ctx context.Context, c client.Client, owner client.Object, svc *corev1.Service, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(owner, svc, scheme); err != nil {
		return err
	}
	return c.Create(ctx, svc)
}

func EnsureService(ctx context.Context, c client.Client, owner client.Object, scheme *runtime.Scheme, name, namespace string, port int32, labels map[string]string) error {
	svc := BuildService(name, namespace, port, labels)

	existing := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing)
	if err != nil && errors.IsNotFound(err) {
		return CreateService(ctx, c, owner, svc, scheme)
	}
	return err
}

func DeleteService(ctx context.Context, c client.Client, name, namespace string) error {
	existing := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing)
	if err != nil {
		return nil
	}
	return c.Delete(ctx, existing)
}
