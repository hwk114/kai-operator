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
	"context"
	"fmt"

	"github.com/hwk114/kai/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InferenceTaskClient struct {
	client    client.Client
	namespace string
}

func (c *InferenceTaskClient) Create(ctx context.Context, task *v1alpha1.InferenceTask, opts ...client.CreateOption) error {
	return c.client.Create(ctx, task, opts...)
}

func (c *InferenceTaskClient) Update(ctx context.Context, task *v1alpha1.InferenceTask, opts ...client.UpdateOption) error {
	return c.client.Update(ctx, task, opts...)
}

func (c *InferenceTaskClient) UpdateStatus(ctx context.Context, task *v1alpha1.InferenceTask, opts ...client.SubResourceUpdateOption) error {
	return c.client.Status().Update(ctx, task, opts...)
}

func (c *InferenceTaskClient) Delete(ctx context.Context, name string, opts ...client.DeleteOption) error {
	task := &v1alpha1.InferenceTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
		},
	}
	return c.client.Delete(ctx, task, opts...)
}

func (c *InferenceTaskClient) DeleteAllOf(ctx context.Context, opts ...client.DeleteAllOfOption) error {
	return c.client.DeleteAllOf(ctx, &v1alpha1.InferenceTask{}, opts...)
}

func (c *InferenceTaskClient) Get(ctx context.Context, name string, opts ...client.GetOption) (*v1alpha1.InferenceTask, error) {
	task := &v1alpha1.InferenceTask{}
	err := c.client.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, task, opts...)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (c *InferenceTaskClient) List(ctx context.Context, opts ...client.ListOption) ([]*v1alpha1.InferenceTask, error) {
	list := &v1alpha1.InferenceTaskList{}
	if err := c.client.List(ctx, list, append([]client.ListOption{client.InNamespace(c.namespace)}, opts...)...); err != nil {
		return nil, err
	}

	result := make([]*v1alpha1.InferenceTask, len(list.Items))
	for i := range list.Items {
		result[i] = &list.Items[i]
	}
	return result, nil
}

func (c *InferenceTaskClient) Exists(ctx context.Context, name string) (bool, error) {
	_, err := c.Get(ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *InferenceTaskClient) Patch(ctx context.Context, task *v1alpha1.InferenceTask, patch client.Patch, opts ...client.PatchOption) error {
	return c.client.Patch(ctx, task, patch, opts...)
}

func (c *InferenceTaskClient) CreateOrPatch(ctx context.Context, task *v1alpha1.InferenceTask, modifyFn func(*v1alpha1.InferenceTask)) error {
	existing, err := c.Get(ctx, task.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			modifyFn(task)
			return c.Create(ctx, task)
		}
		return err
	}

	modifyFn(existing)
	existing.ResourceVersion = task.ResourceVersion
	return c.Update(ctx, existing)
}

func (c *InferenceTaskClient) GetPhase(ctx context.Context, name string) (v1alpha1.TaskPhase, error) {
	task, err := c.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return task.Status.Phase, nil
}

func (c *InferenceTaskClient) IsRunning(ctx context.Context, name string) (bool, error) {
	phase, err := c.GetPhase(ctx, name)
	if err != nil {
		return false, err
	}
	return phase == v1alpha1.TaskPhaseRunning, nil
}

func (c *InferenceTaskClient) IsCompleted(ctx context.Context, name string) (bool, error) {
	phase, err := c.GetPhase(ctx, name)
	if err != nil {
		return false, err
	}
	return phase == v1alpha1.TaskPhaseCompleted || phase == v1alpha1.TaskPhaseFailed, nil
}

func (c *InferenceTaskClient) GetEndpoint(ctx context.Context, name string) (string, error) {
	task, err := c.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return task.Status.ExternalURL, nil
}

func (c *InferenceTaskClient) GetServiceURL(ctx context.Context, name string) (string, error) {
	task, err := c.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return task.Status.ServiceURL, nil
}

func (c *InferenceTaskClient) WaitForRunning(ctx context.Context, name string, opts ...WaitOption) (*v1alpha1.InferenceTask, error) {
	opt := parseWaitOptions(opts...)
	task, err := c.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if task.Status.Phase == v1alpha1.TaskPhaseRunning {
		return task, nil
	}
	if task.Status.Phase == v1alpha1.TaskPhaseFailed {
		return task, fmt.Errorf("task failed: %s", task.Status.Message)
	}
	err = waitForPhase(ctx, func() (v1alpha1.TaskPhase, error) {
		t, err := c.Get(ctx, name)
		if err != nil {
			return "", err
		}
		return t.Status.Phase, nil
	}, v1alpha1.TaskPhaseRunning, opt)
	if err != nil {
		t, _ := c.Get(ctx, name)
		return t, err
	}
	return c.Get(ctx, name)
}

func (c *InferenceTaskClient) WaitForCompleted(ctx context.Context, name string, opts ...WaitOption) (*v1alpha1.InferenceTask, error) {
	opt := parseWaitOptions(opts...)
	err := waitForTerminal(ctx, func() (v1alpha1.TaskPhase, error) {
		t, err := c.Get(ctx, name)
		if err != nil {
			return "", err
		}
		return t.Status.Phase, nil
	}, opt)
	if err != nil {
		t, _ := c.Get(ctx, name)
		return t, err
	}
	return c.Get(ctx, name)
}

func (c *InferenceTaskClient) WaitForDeletion(ctx context.Context, name string, opts ...WaitOption) error {
	return waitForDeletion(ctx, func() (bool, error) {
		_, err := c.Get(ctx, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}, parseWaitOptions(opts...))
}

func (c *InferenceTaskClient) UpdateWithRetry(ctx context.Context, name string, modify func(*v1alpha1.InferenceTask) error) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		task, err := c.Get(ctx, name)
		if err != nil {
			return err
		}
		return modify(task)
	})
}
