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

package sdk

import (
	"github.com/hwk114/kai-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InferenceTaskBuilder struct {
	task *v1alpha1.InferenceTask
}

func NewInferenceTask(name, namespace string) *InferenceTaskBuilder {
	return &InferenceTaskBuilder{
		task: &v1alpha1.InferenceTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.InferenceTaskSpec{},
		},
	}
}

func (b *InferenceTaskBuilder) WithVLLM(model string, opts ...func(*v1alpha1.VLLMSpec)) *InferenceTaskBuilder {
	spec := v1alpha1.VLLMSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "vllm/vllm:latest",
			Port:  8000,
		},
		ModelPath: model,
		ModelName: model,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.VLLM = &spec
	b.task.Spec.Model = model
	b.task.Spec.Engine = "vllm"
	return b
}

func (b *InferenceTaskBuilder) WithLlamaCpp(model string, opts ...func(*v1alpha1.LlamaCppSpec)) *InferenceTaskBuilder {
	spec := v1alpha1.LlamaCppSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "ghcr.io/ggml-org/llama.cpp:server",
		},
		ModelPath: model,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.LlamaCpp = &spec
	b.task.Spec.Model = model
	b.task.Spec.Engine = "llama.cpp"
	return b
}

func (b *InferenceTaskBuilder) WithTGI(model string, opts ...func(*v1alpha1.TGISpec)) *InferenceTaskBuilder {
	spec := v1alpha1.TGISpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "ghcr.io/huggingface/text-generation-inference:latest",
			Port:  8080,
		},
		ModelPath: model,
		ModelID:   model,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.TGI = &spec
	b.task.Spec.Model = model
	b.task.Spec.Engine = "tgi"
	return b
}

func (b *InferenceTaskBuilder) WithResources(cpu, memory, gpu string) *InferenceTaskBuilder {
	b.task.Spec.Resources = &v1alpha1.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	return b
}

func (b *InferenceTaskBuilder) WithGPUCount(count int32) *InferenceTaskBuilder {
	if b.task.Spec.VLLM != nil {
		b.task.Spec.VLLM.TensorParallelSize = count
	}
	return b
}

func (b *InferenceTaskBuilder) WithGPUMemoryUtilization(util string) *InferenceTaskBuilder {
	if b.task.Spec.VLLM != nil {
		b.task.Spec.VLLM.GPUMemoryUtilization = util
	}
	return b
}

func (b *InferenceTaskBuilder) WithVolume(name, capacity, mountPath string) *InferenceTaskBuilder {
	b.task.Spec.Volumes = append(b.task.Spec.Volumes, v1alpha1.VolumeSpec{
		Name:      name,
		Type:      "PersistentVolumeClaim",
		Capacity:  capacity,
		MountPath: mountPath,
	})
	return b
}

func (b *InferenceTaskBuilder) WithEnv(name, value string) *InferenceTaskBuilder {
	env := v1alpha1.EnvVar{Name: name, Value: value}
	if b.task.Spec.VLLM != nil {
		b.task.Spec.VLLM.Env = append(b.task.Spec.VLLM.Env, env)
	}
	if b.task.Spec.LlamaCpp != nil {
		b.task.Spec.LlamaCpp.Env = append(b.task.Spec.LlamaCpp.Env, env)
	}
	if b.task.Spec.TGI != nil {
		b.task.Spec.TGI.Env = append(b.task.Spec.TGI.Env, env)
	}
	return b
}

func (b *InferenceTaskBuilder) WithLabels(labels map[string]string) *InferenceTaskBuilder {
	if b.task.Labels == nil {
		b.task.Labels = make(map[string]string)
	}
	for k, v := range labels {
		b.task.Labels[k] = v
	}
	return b
}

func (b *InferenceTaskBuilder) WithTTL(seconds int32) *InferenceTaskBuilder {
	b.task.Spec.TTLSeconds = &seconds
	return b
}

func (b *InferenceTaskBuilder) WithAutoScaling(min, max int32) *InferenceTaskBuilder {
	spec := &v1alpha1.AutoScalingSpec{
		MinReplicas: min,
		MaxReplicas: max,
	}
	if b.task.Spec.VLLM != nil {
		b.task.Spec.VLLM.AutoScaling = spec
	}
	if b.task.Spec.TGI != nil {
		b.task.Spec.TGI.AutoScaling = spec
	}
	return b
}

func (b *InferenceTaskBuilder) WithRouting(enabled bool, opts ...func(*v1alpha1.RoutingSpec)) *InferenceTaskBuilder {
	spec := &v1alpha1.RoutingSpec{Enabled: enabled}
	for _, opt := range opts {
		opt(spec)
	}
	if b.task.Spec.VLLM != nil {
		b.task.Spec.VLLM.Routing = spec
	}
	if b.task.Spec.LlamaCpp != nil {
		b.task.Spec.LlamaCpp.Routing = spec
	}
	if b.task.Spec.TGI != nil {
		b.task.Spec.TGI.Routing = spec
	}
	return b
}

func (b *InferenceTaskBuilder) Build() *v1alpha1.InferenceTask {
	return b.task
}
