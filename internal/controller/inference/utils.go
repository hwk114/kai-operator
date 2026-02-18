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

package inference

import (
	"fmt"
	"strings"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	"github.com/hwk114/kai/internal/controller/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildInitContainers(inference *kaiiov1alpha1.InferenceTask, framework, hfModelID string) []corev1.Container {
	if hfModelID == "" {
		return nil
	}

	env := []corev1.EnvVar{}
	if inference.Spec.HFMirror != "" {
		env = append(env, corev1.EnvVar{Name: "HF_ENDPOINT", Value: inference.Spec.HFMirror})
	}

	initContainer := corev1.Container{
		Name:    "download-model",
		Image:   "python:3.11-slim",
		Env:     env,
		Command: []string{"sh", "-c"},
		Args: []string{
			fmt.Sprintf(`
pip install huggingface_hub -q
python -c "
import os
os.environ['HF_HUB_OFFLINE'] = '0'
from huggingface_hub import snapshot_download
model_id = '%s'
try:
    snapshot_download(model_id, cache_dir='/models', local_dir='/models')
    print(f'Downloaded model to /models')
except Exception as e:
    print(f'Error: {e}')
    import sys
    sys.exit(1)
"
`, hfModelID),
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/models"},
		},
	}

	return []corev1.Container{initContainer}
}

func GetModelPath(inference *kaiiov1alpha1.InferenceTask) (modelPath string, modelID string) {
	model := inference.Spec.Model
	if model == "" {
		return "", ""
	}

	if strings.HasPrefix(model, "s3://") {
		return strings.TrimPrefix(model, "s3://"), ""
	}
	if strings.HasPrefix(model, "hf://") {
		return "", strings.TrimPrefix(model, "hf://")
	}
	return model, ""
}

func GetEngine(inference *kaiiov1alpha1.InferenceTask) string {
	if inference.Spec.Engine != "" {
		return inference.Spec.Engine
	}
	if inference.Spec.VLLM != nil {
		return "vllm"
	}
	if inference.Spec.TGI != nil {
		return "tgi"
	}
	if inference.Spec.LlamaCpp != nil {
		return "llama-cpp"
	}
	return ""
}

func GetInferencePort(inference *kaiiov1alpha1.InferenceTask) int32 {
	if inference.Spec.VLLM != nil && inference.Spec.VLLM.Port > 0 {
		return inference.Spec.VLLM.Port
	}
	if inference.Spec.TGI != nil && inference.Spec.TGI.Port > 0 {
		return inference.Spec.TGI.Port
	}
	if inference.Spec.LlamaCpp != nil && inference.Spec.LlamaCpp.Port > 0 {
		return inference.Spec.LlamaCpp.Port
	}
	return 8000
}

func GetInferenceImage(inference *kaiiov1alpha1.InferenceTask) string {
	if inference.Spec.VLLM != nil {
		if inference.Spec.VLLM.Image != "" {
			return inference.Spec.VLLM.Image
		}
		return "vllm/vllm-openai:nightly-aarch64"
	}
	if inference.Spec.TGI != nil {
		if inference.Spec.TGI.Image != "" {
			return inference.Spec.TGI.Image
		}
		return "ghcr.io/huggingface/text-generation-inference:latest"
	}
	if inference.Spec.LlamaCpp != nil {
		if inference.Spec.LlamaCpp.Image != "" {
			return inference.Spec.LlamaCpp.Image
		}
		return "ghcr.io/ggml-org/llama.cpp/server"
	}
	return "alpine:latest"
}

func BuildInferenceDeployment(inference *kaiiov1alpha1.InferenceTask) *appsv1.Deployment {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	deployName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)
	image := GetInferenceImage(inference)

	var args []string
	var env []corev1.EnvVar

	s3Path, hfModelID := GetModelPath(inference)

	if s3Path != "" {
		args = append(args, "--model", "/models/"+s3Path)
	}

	if inference.Spec.VLLM != nil && hfModelID != "" {
		args = append(args, "--trust-remote-code")
		args = append(args, "--model", hfModelID)
		if inference.Spec.HFMirror != "" {
			env = append(env, corev1.EnvVar{Name: "HF_ENDPOINT", Value: inference.Spec.HFMirror})
		}
	}

	if inference.Spec.TGI != nil && hfModelID != "" {
		args = append(args, "--trust-remote-code")
		args = append(args, "--model", hfModelID)
		if inference.Spec.HFMirror != "" {
			env = append(env, corev1.EnvVar{Name: "HF_ENDPOINT", Value: inference.Spec.HFMirror})
		}
	}

	if inference.Spec.VLLM != nil {
		vllm := inference.Spec.VLLM
		args = append(args, "--host", "0.0.0.0")
		args = append(args, "--port", fmt.Sprintf("%d", GetInferencePort(inference)))

		if vllm.TensorParallelSize > 1 {
			args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", vllm.TensorParallelSize))
		}
		if vllm.GPUMemoryUtilization != "" {
			args = append(args, "--gpu-memory-utilization", vllm.GPUMemoryUtilization)
		}
		if vllm.MaxModelLen > 0 {
			args = append(args, "--max-model-len", fmt.Sprintf("%d", vllm.MaxModelLen))
		}
	}

	if inference.Spec.TGI != nil {
		tgi := inference.Spec.TGI
		args = append(args, "--host", "0.0.0.0")
		args = append(args, "--port", fmt.Sprintf("%d", GetInferencePort(inference)))

		if tgi.MaxBatchSize > 0 {
			args = append(args, "--max-batch-size", fmt.Sprintf("%d", tgi.MaxBatchSize))
		}
		if tgi.MaxInputLen > 0 {
			args = append(args, "--max-input-length", fmt.Sprintf("%d", tgi.MaxInputLen))
		}
		if tgi.MaxTotalTokens > 0 {
			args = append(args, "--max-total-tokens", fmt.Sprintf("%d", tgi.MaxTotalTokens))
		}
	}

	if inference.Spec.LlamaCpp != nil {
		llama := inference.Spec.LlamaCpp
		args = append(args, "--host", "0.0.0.0")
		args = append(args, "--port", fmt.Sprintf("%d", GetInferencePort(inference)))
		if llama.ContextSize > 0 {
			args = append(args, "-c", fmt.Sprintf("%d", llama.ContextSize))
		}
		if llama.GPULayers > 0 {
			args = append(args, "--n-gpu-layers", fmt.Sprintf("%d", llama.GPULayers))
		}
	}

	env = append(env, []corev1.EnvVar{
		{Name: "KAI_TASK_NAME", Value: inference.Name},
		{Name: "KAI_TASK_TYPE", Value: "inference"},
		{Name: "KAI_FRAMEWORK", Value: framework},
	}...)

	if inference.Spec.VLLM != nil && inference.Spec.VLLM.Env != nil {
		for _, e := range inference.Spec.VLLM.Env {
			env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}

	if inference.Spec.TGI != nil && inference.Spec.TGI.Env != nil {
		for _, e := range inference.Spec.TGI.Env {
			env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}

	if inference.Spec.LlamaCpp != nil && inference.Spec.LlamaCpp.Env != nil {
		for _, e := range inference.Spec.LlamaCpp.Env {
			env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}

	replicas := int32(1)
	if inference.Spec.VLLM != nil && inference.Spec.VLLM.AutoScaling != nil {
		replicas = inference.Spec.VLLM.AutoScaling.MinReplicas
	}
	if inference.Spec.TGI != nil && inference.Spec.TGI.AutoScaling != nil {
		replicas = inference.Spec.TGI.AutoScaling.MinReplicas
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: inference.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kai",
				"app.kubernetes.io/managed-by": "kai-operator",
				"kai.io/inference":             inference.Name,
				"kai.io/task-type":             "inference",
				"framework":                    framework,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kai.io/inference": inference.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       "kai",
						"app.kubernetes.io/managed-by": "kai-operator",
						"kai.io/inference":             inference.Name,
						"kai.io/task-type":             "inference",
						"framework":                    framework,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyAlways,
					InitContainers: buildInitContainers(inference, framework, hfModelID),
					Containers: []corev1.Container{
						{
							Name:  "inference",
							Image: image,
							Env:   env,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: GetInferencePort(inference),
								},
							},
							Args:      args,
							Resources: builder.BuildResources(inference.Spec.Resources),
						},
					},
				},
			},
		},
	}

	volumes := []corev1.Volume{
		{
			Name: "models",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{}

	if inference.Spec.Volumes != nil {
		for _, vol := range inference.Spec.Volumes {
			volume := corev1.Volume{
				Name: vol.Name,
			}
			switch vol.Type {
			case "emptyDir":
				volume.VolumeSource = corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				}
			case "persistentVolumeClaim":
				volume.VolumeSource = corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: vol.Name,
					},
				}
			case "configMap":
				volume.VolumeSource = corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: vol.Name},
					},
				}
			case "secret":
				volume.VolumeSource = corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: vol.Name,
					},
				}
			}
			volumes = append(volumes, volume)
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      vol.Name,
				MountPath: vol.MountPath,
			})
		}
	}

	deployment.Spec.Template.Spec.Volumes = volumes
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	return deployment
}

func portToIntstr(port int32) intstr.IntOrString {
	return intstr.FromInt(int(port))
}

func ptrTo[T any](v T) *T {
	return &v
}
