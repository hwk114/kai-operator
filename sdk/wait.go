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
	"time"

	"github.com/hwk114/kai/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type WaitOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

var DefaultWaitOptions = WaitOptions{
	Interval: 2 * time.Second,
	Timeout:  10 * time.Minute,
}

type WaitOption func(*WaitOptions)

func WithInterval(interval time.Duration) WaitOption {
	return func(o *WaitOptions) { o.Interval = interval }
}

func WithTimeout(timeout time.Duration) WaitOption {
	return func(o *WaitOptions) { o.Timeout = timeout }
}

func parseWaitOptions(opts ...WaitOption) WaitOptions {
	options := DefaultWaitOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func waitForPhase(ctx context.Context, getFn func() (v1alpha1.TaskPhase, error), target v1alpha1.TaskPhase, opt WaitOptions) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	ticker := time.NewTicker(opt.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for phase %s", target)
		case <-ticker.C:
			phase, err := getFn()
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return err
			}
			if phase == target {
				return nil
			}
		}
	}
}

func waitForTerminal(ctx context.Context, getFn func() (v1alpha1.TaskPhase, error), opt WaitOptions) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	ticker := time.NewTicker(opt.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for terminal phase")
		case <-ticker.C:
			phase, err := getFn()
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return err
			}
			if phase == v1alpha1.TaskPhaseCompleted || phase == v1alpha1.TaskPhaseFailed {
				return nil
			}
		}
	}
}

func waitForDeletion(ctx context.Context, existsFn func() (bool, error), opt WaitOptions) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	ticker := time.NewTicker(opt.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for deletion")
		case <-ticker.C:
			exists, err := existsFn()
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
		}
	}
}

type TaskType string

const (
	TaskTypeDev       TaskType = "dev"
	TaskTypeInference TaskType = "inference"
	TaskTypeTrain     TaskType = "train"
	TaskTypeFineTune  TaskType = "finetune"
	TaskTypeEval      TaskType = "eval"
)

func (c *Client) WaitForTask(ctx context.Context, tt TaskType, name string, opts ...WaitOption) (interface{}, error) {
	switch tt {
	case TaskTypeDev:
		return c.DevTask.WaitForRunning(ctx, name, opts...)
	case TaskTypeInference:
		return c.InferenceTask.WaitForRunning(ctx, name, opts...)
	case TaskTypeTrain:
		return c.TrainTask.WaitForRunning(ctx, name, opts...)
	case TaskTypeFineTune:
		return c.FineTuneTask.WaitForRunning(ctx, name, opts...)
	case TaskTypeEval:
		return c.EvalTask.WaitForRunning(ctx, name, opts...)
	default:
		return nil, fmt.Errorf("unknown task type: %s", tt)
	}
}

func (c *Client) GetTask(ctx context.Context, tt TaskType, name string) (interface{}, error) {
	switch tt {
	case TaskTypeDev:
		return c.DevTask.Get(ctx, name)
	case TaskTypeInference:
		return c.InferenceTask.Get(ctx, name)
	case TaskTypeTrain:
		return c.TrainTask.Get(ctx, name)
	case TaskTypeFineTune:
		return c.FineTuneTask.Get(ctx, name)
	case TaskTypeEval:
		return c.EvalTask.Get(ctx, name)
	default:
		return nil, fmt.Errorf("unknown task type: %s", tt)
	}
}

func (c *Client) DeleteTask(ctx context.Context, tt TaskType, name string) error {
	switch tt {
	case TaskTypeDev:
		return c.DevTask.Delete(ctx, name)
	case TaskTypeInference:
		return c.InferenceTask.Delete(ctx, name)
	case TaskTypeTrain:
		return c.TrainTask.Delete(ctx, name)
	case TaskTypeFineTune:
		return c.FineTuneTask.Delete(ctx, name)
	case TaskTypeEval:
		return c.EvalTask.Delete(ctx, name)
	default:
		return fmt.Errorf("unknown task type: %s", tt)
	}
}
