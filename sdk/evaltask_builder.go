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

type EvalTaskBuilder struct {
	task *v1alpha1.EvalTask
}

func NewEvalTask(name, namespace string) *EvalTaskBuilder {
	return &EvalTaskBuilder{
		task: &v1alpha1.EvalTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.EvalTaskSpec{},
		},
	}
}

func (b *EvalTaskBuilder) WithOpenCompass(opts ...func(*v1alpha1.OpenCompassSpec)) *EvalTaskBuilder {
	spec := v1alpha1.OpenCompassSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "opencompass/opencompass:latest",
		},
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.OpenCompass = &spec
	return b
}

func (b *EvalTaskBuilder) WithLmEval(opts ...func(*v1alpha1.LmEvalSpec)) *EvalTaskBuilder {
	spec := v1alpha1.LmEvalSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "ghcr.io/EleutherAI/lm-evaluation-harness:latest",
		},
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.LmEval = &spec
	return b
}

func (b *EvalTaskBuilder) WithModel(model, modelPath string) *EvalTaskBuilder {
	if b.task.Spec.OpenCompass != nil {
		b.task.Spec.OpenCompass.Models = append(b.task.Spec.OpenCompass.Models, v1alpha1.ModelEvalConfig{
			Model:     model,
			ModelPath: modelPath,
		})
	}
	if b.task.Spec.LmEval != nil {
		b.task.Spec.LmEval.ModelPath = modelPath
	}
	return b
}

func (b *EvalTaskBuilder) WithDatasets(datasets ...string) *EvalTaskBuilder {
	if b.task.Spec.OpenCompass != nil {
		b.task.Spec.OpenCompass.Datasets = datasets
	}
	if b.task.Spec.LmEval != nil {
		b.task.Spec.LmEval.Tasks = datasets
	}
	return b
}

func (b *EvalTaskBuilder) WithResources(cpu, memory, gpu string) *EvalTaskBuilder {
	b.task.Spec.Resources = &v1alpha1.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	return b
}

func (b *EvalTaskBuilder) WithVolume(name, capacity, mountPath string) *EvalTaskBuilder {
	b.task.Spec.Volumes = append(b.task.Spec.Volumes, v1alpha1.VolumeSpec{
		Name:      name,
		Type:      "PersistentVolumeClaim",
		Capacity:  capacity,
		MountPath: mountPath,
	})
	return b
}

func (b *EvalTaskBuilder) WithLabels(labels map[string]string) *EvalTaskBuilder {
	if b.task.Labels == nil {
		b.task.Labels = make(map[string]string)
	}
	for k, v := range labels {
		b.task.Labels[k] = v
	}
	return b
}

func (b *EvalTaskBuilder) WithTTL(seconds int32) *EvalTaskBuilder {
	b.task.Spec.TTLSeconds = &seconds
	return b
}

func (b *EvalTaskBuilder) Build() *v1alpha1.EvalTask {
	return b.task
}
