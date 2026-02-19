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

type FineTuneTaskBuilder struct {
	task *v1alpha1.FineTuneTask
}

func NewFineTuneTask(name, namespace string) *FineTuneTaskBuilder {
	return &FineTuneTaskBuilder{
		task: &v1alpha1.FineTuneTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.FineTuneTaskSpec{},
		},
	}
}

func (b *FineTuneTaskBuilder) WithLoRA(modelPath string, opts ...func(*v1alpha1.LoRASpec)) *FineTuneTaskBuilder {
	spec := v1alpha1.LoRASpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "ghcr.io/huggingface/peft:latest",
		},
		ModelPath: modelPath,
		LoraRank:  8,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.LoRA = &spec
	return b
}

func (b *FineTuneTaskBuilder) WithLlamaFactory(modelPath string, opts ...func(*v1alpha1.LlamaFactorySpec)) *FineTuneTaskBuilder {
	spec := v1alpha1.LlamaFactorySpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "hiyouga/llamafactory:latest",
		},
		ModelPath:      modelPath,
		FinetuningType: "lora",
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.LlamaFactory = &spec
	return b
}

func (b *FineTuneTaskBuilder) WithResources(cpu, memory, gpu string) *FineTuneTaskBuilder {
	b.task.Spec.Resources = &v1alpha1.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	return b
}

func (b *FineTuneTaskBuilder) WithVolume(name, capacity, mountPath string) *FineTuneTaskBuilder {
	b.task.Spec.Volumes = append(b.task.Spec.Volumes, v1alpha1.VolumeSpec{
		Name:      name,
		Type:      "PersistentVolumeClaim",
		Capacity:  capacity,
		MountPath: mountPath,
	})
	return b
}

func (b *FineTuneTaskBuilder) WithTrainData(data string) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.TrainData = data
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.TrainData = data
	}
	return b
}

func (b *FineTuneTaskBuilder) WithValidData(data string) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.ValidData = data
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.ValidData = data
	}
	return b
}

func (b *FineTuneTaskBuilder) WithOutputDir(dir string) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.OutputDir = dir
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.OutputDir = dir
	}
	return b
}

func (b *FineTuneTaskBuilder) WithEpochs(epochs int32) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.Epochs = epochs
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.NumEpochs = epochs
	}
	return b
}

func (b *FineTuneTaskBuilder) WithBatchSize(size int32) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.BatchSize = size
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.BatchSize = size
	}
	return b
}

func (b *FineTuneTaskBuilder) WithLearningRate(lr string) *FineTuneTaskBuilder {
	if b.task.Spec.LoRA != nil {
		b.task.Spec.LoRA.LearningRate = lr
	}
	if b.task.Spec.LlamaFactory != nil {
		b.task.Spec.LlamaFactory.LearningRate = lr
	}
	return b
}

func (b *FineTuneTaskBuilder) WithLabels(labels map[string]string) *FineTuneTaskBuilder {
	if b.task.Labels == nil {
		b.task.Labels = make(map[string]string)
	}
	for k, v := range labels {
		b.task.Labels[k] = v
	}
	return b
}

func (b *FineTuneTaskBuilder) WithTTL(seconds int32) *FineTuneTaskBuilder {
	b.task.Spec.TTLSeconds = &seconds
	return b
}

func (b *FineTuneTaskBuilder) Build() *v1alpha1.FineTuneTask {
	return b.task
}
