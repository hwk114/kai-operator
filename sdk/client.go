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

	"github.com/hwk114/kai-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	k8sClient client.Client
	scheme    *runtime.Scheme

	DevTask       *DevTaskClient
	InferenceTask *InferenceTaskClient
	TrainTask     *TrainTaskClient
	FineTuneTask  *FineTuneTaskClient
	EvalTask      *EvalTaskClient
}

type Options struct {
	Namespace string
}

type ClientOption func(*Options)

func WithNamespace(namespace string) ClientOption {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

func NewClient(config *rest.Config, opts ...ClientOption) (*Client, error) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	options := &Options{Namespace: "default"}
	for _, opt := range opts {
		opt(options)
	}

	c := &Client{
		k8sClient: k8sClient,
		scheme:    scheme,
	}

	c.DevTask = &DevTaskClient{client: k8sClient, namespace: options.Namespace}
	c.InferenceTask = &InferenceTaskClient{client: k8sClient, namespace: options.Namespace}
	c.TrainTask = &TrainTaskClient{client: k8sClient, namespace: options.Namespace}
	c.FineTuneTask = &FineTuneTaskClient{client: k8sClient, namespace: options.Namespace}
	c.EvalTask = &EvalTaskClient{client: k8sClient, namespace: options.Namespace}

	return c, nil
}

func NewClientFromK8s(k8sClient client.Client, opts ...ClientOption) (*Client, error) {
	options := &Options{Namespace: "default"}
	for _, opt := range opts {
		opt(options)
	}

	c := &Client{
		k8sClient: k8sClient,
		scheme:    k8sClient.Scheme(),
	}

	c.DevTask = &DevTaskClient{client: k8sClient, namespace: options.Namespace}
	c.InferenceTask = &InferenceTaskClient{client: k8sClient, namespace: options.Namespace}
	c.TrainTask = &TrainTaskClient{client: k8sClient, namespace: options.Namespace}
	c.FineTuneTask = &FineTuneTaskClient{client: k8sClient, namespace: options.Namespace}
	c.EvalTask = &EvalTaskClient{client: k8sClient, namespace: options.Namespace}

	return c, nil
}

func (c *Client) KubernetesClient() client.Client {
	return c.k8sClient
}

func (c *Client) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *Client) SetNamespace(namespace string) {
	c.DevTask.namespace = namespace
	c.InferenceTask.namespace = namespace
	c.TrainTask.namespace = namespace
	c.FineTuneTask.namespace = namespace
	c.EvalTask.namespace = namespace
}

func (c *Client) ListAllTasks(ctx context.Context) (*TaskList, error) {
	var list TaskList

	devTasks, err := c.DevTask.List(ctx)
	if err != nil {
		return nil, err
	}
	list.DevTasks = devTasks

	inferenceTasks, err := c.InferenceTask.List(ctx)
	if err != nil {
		return nil, err
	}
	list.InferenceTasks = inferenceTasks

	trainTasks, err := c.TrainTask.List(ctx)
	if err != nil {
		return nil, err
	}
	list.TrainTasks = trainTasks

	finetuneTasks, err := c.FineTuneTask.List(ctx)
	if err != nil {
		return nil, err
	}
	list.FineTuneTasks = finetuneTasks

	evalTasks, err := c.EvalTask.List(ctx)
	if err != nil {
		return nil, err
	}
	list.EvalTasks = evalTasks

	return &list, nil
}

type TaskList struct {
	DevTasks       []*v1alpha1.DevTask
	InferenceTasks []*v1alpha1.InferenceTask
	TrainTasks     []*v1alpha1.TrainTask
	FineTuneTasks  []*v1alpha1.FineTuneTask
	EvalTasks      []*v1alpha1.EvalTask
}
