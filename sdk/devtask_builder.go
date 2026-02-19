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

type DevTaskBuilder struct {
	task *v1alpha1.DevTask
}

func NewDevTask(name, namespace string) *DevTaskBuilder {
	return &DevTaskBuilder{
		task: &v1alpha1.DevTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.DevTaskSpec{},
		},
	}
}

func (b *DevTaskBuilder) WithCodeServer(image string, opts ...func(*v1alpha1.CodeServerSpec)) *DevTaskBuilder {
	spec := v1alpha1.CodeServerSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: image,
			Port:  8080,
		},
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.CodeServer = &spec
	return b
}

func (b *DevTaskBuilder) WithJupyter(image string, opts ...func(*v1alpha1.JupyterSpec)) *DevTaskBuilder {
	spec := v1alpha1.JupyterSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: image,
			Port:  8888,
		},
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.Jupyter = &spec
	return b
}

func (b *DevTaskBuilder) WithNoVNC(image string, opts ...func(*v1alpha1.NoVNCSpec)) *DevTaskBuilder {
	spec := v1alpha1.NoVNCSpec{
		BaseToolSpec: v1alpha1.BaseToolSpec{
			Image: image,
			Port:  6080,
		},
	}
	for _, opt := range opts {
		opt(&spec)
	}
	b.task.Spec.NoVNC = &spec
	return b
}

func (b *DevTaskBuilder) WithResources(cpu, memory, gpu string) *DevTaskBuilder {
	b.task.Spec.Resources = &v1alpha1.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	return b
}

func (b *DevTaskBuilder) WithVolume(name, capacity, mountPath string) *DevTaskBuilder {
	b.task.Spec.Volumes = append(b.task.Spec.Volumes, v1alpha1.VolumeSpec{
		Name:      name,
		Type:      "PersistentVolumeClaim",
		Capacity:  capacity,
		MountPath: mountPath,
	})
	return b
}

func (b *DevTaskBuilder) WithDataSource(uri, mountPath string) *DevTaskBuilder {
	b.task.Spec.DataSources = append(b.task.Spec.DataSources, v1alpha1.DataSourceSpec{
		Type:      "PVC",
		URI:       uri,
		MountPath: mountPath,
	})
	return b
}

func (b *DevTaskBuilder) WithEnv(name, value string) *DevTaskBuilder {
	env := v1alpha1.EnvVar{Name: name, Value: value}
	if b.task.Spec.CodeServer != nil {
		b.task.Spec.CodeServer.Env = append(b.task.Spec.CodeServer.Env, env)
	}
	if b.task.Spec.Jupyter != nil {
		b.task.Spec.Jupyter.Env = append(b.task.Spec.Jupyter.Env, env)
	}
	if b.task.Spec.NoVNC != nil {
		b.task.Spec.NoVNC.Env = append(b.task.Spec.NoVNC.Env, env)
	}
	return b
}

func (b *DevTaskBuilder) WithLabels(labels map[string]string) *DevTaskBuilder {
	if b.task.Labels == nil {
		b.task.Labels = make(map[string]string)
	}
	for k, v := range labels {
		b.task.Labels[k] = v
	}
	return b
}

func (b *DevTaskBuilder) WithTTL(seconds int32) *DevTaskBuilder {
	b.task.Spec.TTLSeconds = &seconds
	return b
}

func (b *DevTaskBuilder) WithRouting(enabled bool, opts ...func(*v1alpha1.RoutingSpec)) *DevTaskBuilder {
	spec := &v1alpha1.RoutingSpec{Enabled: enabled}
	for _, opt := range opts {
		opt(spec)
	}
	if b.task.Spec.CodeServer != nil {
		b.task.Spec.CodeServer.Routing = spec
	}
	if b.task.Spec.Jupyter != nil {
		b.task.Spec.Jupyter.Routing = spec
	}
	if b.task.Spec.NoVNC != nil {
		b.task.Spec.NoVNC.Routing = spec
	}
	return b
}

func (b *DevTaskBuilder) Build() *v1alpha1.DevTask {
	return b.task
}
