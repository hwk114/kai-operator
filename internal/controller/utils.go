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

package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kaiv1alpha1 "github.com/hwk114/kai/api/v1alpha1"
)

const DefaultRequeueDelay = 30

func EnsureFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.AddFinalizer(obj, finalizer)
		if err := c.Update(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	controllerutil.RemoveFinalizer(obj, finalizer)
	return c.Update(ctx, obj)
}

func HandleDeletion(ctx context.Context, c client.Client, obj client.Object, finalizer string, cleanupFn func() error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		if err := cleanupFn(); err != nil {
			logger.Error(err, "Failed to cleanup resources")
			return ctrl.Result{}, err
		}
		if err := RemoveFinalizer(ctx, c, obj, finalizer); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func UpdateStatusPhase(ctx context.Context, c client.Client, task client.Object, phase kaiv1alpha1.TaskPhase, message string) error {
	switch t := task.(type) {
	case *kaiv1alpha1.DevTask:
		t.Status.Phase = phase
		t.Status.Message = message
	case *kaiv1alpha1.InferenceTask:
		t.Status.Phase = phase
		t.Status.Message = message
	case *kaiv1alpha1.TrainTask:
		t.Status.Phase = phase
		t.Status.Message = message
	case *kaiv1alpha1.FineTuneTask:
		t.Status.Phase = phase
		t.Status.Message = message
	case *kaiv1alpha1.EvalTask:
		t.Status.Phase = phase
		t.Status.Message = message
	}
	return c.Status().Update(ctx, task)
}

func SetStartTime(ctx context.Context, c client.Client, task client.Object) error {
	now := metav1.Now()
	switch t := task.(type) {
	case *kaiv1alpha1.DevTask:
		t.Status.StartTime = &now
	case *kaiv1alpha1.InferenceTask:
		t.Status.StartTime = &now
	case *kaiv1alpha1.TrainTask:
		t.Status.StartTime = &now
	case *kaiv1alpha1.FineTuneTask:
		t.Status.StartTime = &now
	case *kaiv1alpha1.EvalTask:
		t.Status.StartTime = &now
	}
	return c.Status().Update(ctx, task)
}

func SetEndTime(ctx context.Context, c client.Client, task client.Object) error {
	now := metav1.Now()
	switch t := task.(type) {
	case *kaiv1alpha1.DevTask:
		t.Status.EndTime = &now
	case *kaiv1alpha1.InferenceTask:
		t.Status.EndTime = &now
	case *kaiv1alpha1.TrainTask:
		t.Status.EndTime = &now
	case *kaiv1alpha1.FineTuneTask:
		t.Status.EndTime = &now
	case *kaiv1alpha1.EvalTask:
		t.Status.EndTime = &now
	}
	return c.Status().Update(ctx, task)
}

func ScheduleTaskDeletion(ctx context.Context, c client.Client, task client.Object, ttl *int32) {
	if ttl == nil {
		return
	}
	name := task.GetName()
	ns := task.GetNamespace()
	go func() {
		time.Sleep(time.Duration(*ttl) * time.Second)
		if err := c.Delete(context.Background(), &kaiv1alpha1.DevTask{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}); err != nil {
			log.Log.Error(err, "Failed to delete task", "name", name)
		}
	}()
}

func GetTaskPhase(task client.Object) kaiv1alpha1.TaskPhase {
	switch t := task.(type) {
	case *kaiv1alpha1.DevTask:
		return t.Status.Phase
	case *kaiv1alpha1.InferenceTask:
		return t.Status.Phase
	case *kaiv1alpha1.TrainTask:
		return t.Status.Phase
	case *kaiv1alpha1.FineTuneTask:
		return t.Status.Phase
	case *kaiv1alpha1.EvalTask:
		return t.Status.Phase
	}
	return ""
}

func FormatResourceName(prefix, framework, name string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, framework, name)
}

func GetNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: namespace}
}

func SetControllerRef(owner, obj client.Object, scheme *runtime.Scheme) error {
	return controllerutil.SetControllerReference(owner, obj, scheme)
}
