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

type TrainTaskBuilder struct {
	task *v1alpha1.TrainTask
}

func NewTrainTask(name, namespace string) *TrainTaskBuilder {
	return &TrainTaskBuilder{
		task: &v1alpha1.TrainTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.TrainTaskSpec{},
		},
	}
}

func (b *TrainTaskBuilder) WithPyTorch(script string, opts ...func(*v1alpha1.PyTorchSpec)) *TrainTaskBuilder {
	spec := v1alpha1.PyTorchSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "pytorch/pytorch:latest",
		},
		ScriptPath: script,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.PyTorch = &spec
	return b
}

func (b *TrainTaskBuilder) WithTensorFlow(script string, opts ...func(*v1alpha1.TensorFlowSpec)) *TrainTaskBuilder {
	spec := v1alpha1.TensorFlowSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: "tensorflow/tensorflow:latest",
		},
		ScriptPath: script,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.TensorFlow = &spec
	return b
}

func (b *TrainTaskBuilder) WithResources(cpu, memory, gpu string) *TrainTaskBuilder {
	b.task.Spec.Resources = &v1alpha1.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	return b
}

func (b *TrainTaskBuilder) WithVolume(name, capacity, mountPath string) *TrainTaskBuilder {
	b.task.Spec.Volumes = append(b.task.Spec.Volumes, v1alpha1.VolumeSpec{
		Name:      name,
		Type:      "PersistentVolumeClaim",
		Capacity:  capacity,
		MountPath: mountPath,
	})
	return b
}

func (b *TrainTaskBuilder) WithEnv(name, value string) *TrainTaskBuilder {
	env := v1alpha1.EnvVar{Name: name, Value: value}
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.Env = append(b.task.Spec.PyTorch.Env, env)
	}
	if b.task.Spec.TensorFlow != nil {
		b.task.Spec.TensorFlow.Env = append(b.task.Spec.TensorFlow.Env, env)
	}
	return b
}

func (b *TrainTaskBuilder) WithLabels(labels map[string]string) *TrainTaskBuilder {
	if b.task.Labels == nil {
		b.task.Labels = make(map[string]string)
	}
	for k, v := range labels {
		b.task.Labels[k] = v
	}
	return b
}

func (b *TrainTaskBuilder) WithTTL(seconds int32) *TrainTaskBuilder {
	b.task.Spec.TTLSeconds = &seconds
	return b
}

func (b *TrainTaskBuilder) WithHyperparameters(params map[string]string) *TrainTaskBuilder {
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.Hyperparameters = params
	}
	if b.task.Spec.TensorFlow != nil {
		b.task.Spec.TensorFlow.Hyperparameters = params
	}
	return b
}

func (b *TrainTaskBuilder) WithBatchSize(size int32) *TrainTaskBuilder {
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.BatchSize = size
	}
	if b.task.Spec.TensorFlow != nil {
		b.task.Spec.TensorFlow.BatchSize = size
	}
	return b
}

func (b *TrainTaskBuilder) WithLearningRate(lr string) *TrainTaskBuilder {
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.LearningRate = lr
	}
	if b.task.Spec.TensorFlow != nil {
		b.task.Spec.TensorFlow.LearningRate = lr
	}
	return b
}

func (b *TrainTaskBuilder) WithMaxEpochs(epochs int32) *TrainTaskBuilder {
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.MaxEpochs = epochs
	}
	return b
}

func (b *TrainTaskBuilder) WithMaxSteps(steps int32) *TrainTaskBuilder {
	if b.task.Spec.TensorFlow != nil {
		b.task.Spec.TensorFlow.MaxSteps = steps
	}
	return b
}

func (b *TrainTaskBuilder) WithWorldSize(size int32) *TrainTaskBuilder {
	if b.task.Spec.PyTorch != nil {
		b.task.Spec.PyTorch.WorldSize = size
		b.task.Spec.PyTorch.NprocPerNode = size
	}
	return b
}

func (b *TrainTaskBuilder) Build() *v1alpha1.TrainTask {
	return b.task
}
