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

	"github.com/hwk114/kai-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TrainTaskClient struct {
	client    client.Client
	namespace string
}

func (c *TrainTaskClient) Create(ctx context.Context, task *v1alpha1.TrainTask, opts ...client.CreateOption) error {
	return c.client.Create(ctx, task, opts...)
}

func (c *TrainTaskClient) Update(ctx context.Context, task *v1alpha1.TrainTask, opts ...client.UpdateOption) error {
	return c.client.Update(ctx, task, opts...)
}

func (c *TrainTaskClient) UpdateStatus(ctx context.Context, task *v1alpha1.TrainTask, opts ...client.SubResourceUpdateOption) error {
	return c.client.Status().Update(ctx, task, opts...)
}

func (c *TrainTaskClient) Delete(ctx context.Context, name string, opts ...client.DeleteOption) error {
	task := &v1alpha1.TrainTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
		},
	}
	return c.client.Delete(ctx, task, opts...)
}

func (c *TrainTaskClient) DeleteAllOf(ctx context.Context, opts ...client.DeleteAllOfOption) error {
	return c.client.DeleteAllOf(ctx, &v1alpha1.TrainTask{}, opts...)
}

func (c *TrainTaskClient) Get(ctx context.Context, name string, opts ...client.GetOption) (*v1alpha1.TrainTask, error) {
	task := &v1alpha1.TrainTask{}
	err := c.client.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, task, opts...)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (c *TrainTaskClient) List(ctx context.Context, opts ...client.ListOption) ([]*v1alpha1.TrainTask, error) {
	list := &v1alpha1.TrainTaskList{}
	if err := c.client.List(ctx, list, append([]client.ListOption{client.InNamespace(c.namespace)}, opts...)...); err != nil {
		return nil, err
	}

	result := make([]*v1alpha1.TrainTask, len(list.Items))
	for i := range list.Items {
		result[i] = &list.Items[i]
	}
	return result, nil
}

func (c *TrainTaskClient) Exists(ctx context.Context, name string) (bool, error) {
	_, err := c.Get(ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *TrainTaskClient) Patch(ctx context.Context, task *v1alpha1.TrainTask, patch client.Patch, opts ...client.PatchOption) error {
	return c.client.Patch(ctx, task, patch, opts...)
}

func (c *TrainTaskClient) CreateOrPatch(ctx context.Context, task *v1alpha1.TrainTask, modifyFn func(*v1alpha1.TrainTask)) error {
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

func (c *TrainTaskClient) GetPhase(ctx context.Context, name string) (v1alpha1.TaskPhase, error) {
	task, err := c.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return task.Status.Phase, nil
}

func (c *TrainTaskClient) IsRunning(ctx context.Context, name string) (bool, error) {
	phase, err := c.GetPhase(ctx, name)
	if err != nil {
		return false, err
	}
	return phase == v1alpha1.TaskPhaseRunning, nil
}

func (c *TrainTaskClient) IsCompleted(ctx context.Context, name string) (bool, error) {
	phase, err := c.GetPhase(ctx, name)
	if err != nil {
		return false, err
	}
	return phase == v1alpha1.TaskPhaseCompleted || phase == v1alpha1.TaskPhaseFailed, nil
}

func (c *TrainTaskClient) WaitForRunning(ctx context.Context, name string, opts ...WaitOption) (*v1alpha1.TrainTask, error) {
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

func (c *TrainTaskClient) WaitForCompleted(ctx context.Context, name string, opts ...WaitOption) (*v1alpha1.TrainTask, error) {
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

func (c *TrainTaskClient) WaitForDeletion(ctx context.Context, name string, opts ...WaitOption) error {
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

func (c *TrainTaskClient) UpdateWithRetry(ctx context.Context, name string, modify func(*v1alpha1.TrainTask) error) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		task, err := c.Get(ctx, name)
		if err != nil {
			return err
		}
		return modify(task)
	})
}
