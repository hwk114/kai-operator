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

package devtask

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kaiiov1alpha1 "github.com/hwk114/kai-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *DevTaskReconciler) handlePendingPhase(ctx context.Context, task *kaiiov1alpha1.DevTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task); err != nil {
		logger.Error(err, "Failed to get latest DevTask")
		return ctrl.Result{}, err
	}

	if task.Status.Phase != "" && task.Status.Phase != kaiiov1alpha1.TaskPhasePending {
		return ctrl.Result{}, nil
	}

	if err := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task); err != nil {
		logger.Error(err, "Failed to get latest DevTask before update")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, err
	}

	framework := GetFramework(task)
	if task.Labels == nil {
		task.Labels = make(map[string]string)
	}
	task.Labels["framework"] = framework

	if err := r.Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update task labels")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	if err := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task); err != nil {
		logger.Error(err, "Failed to get latest DevTask after label update")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, err
	}

	task.Status.Phase = kaiiov1alpha1.TaskPhaseCreating
	now := metav1.Now()
	task.Status.StartTime = &now
	if err := r.Status().Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update task status")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *DevTaskReconciler) handleCreatingPhase(ctx context.Context, task *kaiiov1alpha1.DevTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	externalURL := ""
	if NeedsExternalRouting(task) {
		var err error
		externalURL, err = GetExternalURLWithGatewayIP(ctx, r.Client, task)
		if err != nil {
			logger.Error(err, "Failed to get external URL, using fallback")
			externalURL = task.GetExternalURL()
		}
	}

	deploy := BuildDeployment(task, externalURL)
	if deploy == nil {
		logger.Error(nil, "Failed to build deployment")
		return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed, "Failed to build deployment")
	}

	if err := ctrl.SetControllerReference(task, deploy, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference for deployment")
		return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed, err.Error())
	}

	existingDeploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, existingDeploy)
	if err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, deploy); err != nil {
			logger.Error(err, "Failed to create deployment")
			return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed, err.Error())
		}
		logger.Info("Deployment created", "deployment", deploy.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get existing deployment")
		return ctrl.Result{}, err
	}

	podName := fmt.Sprintf("devtask-%s-%s", GetFramework(task), task.Name)
	task.Status.PodName = podName

	if err := CreateService(ctx, r.Client, r.Scheme, task); err != nil {
		logger.Error(err, "Failed to create service")
		return ctrl.Result{}, err
	}

	if NeedsExternalRouting(task) {
		serviceName := GetServiceNameForTask(task)
		servicePort := GetServicePortForTask(task)
		if err := CreateHTTPRoute(ctx, r.Client, r.Scheme, task, serviceName, servicePort); err != nil {
			logger.Error(err, "Failed to create HTTPRoute, continuing anyway")
		}
		externalURL, err := GetExternalURLWithGatewayIP(ctx, r.Client, task)
		if err != nil {
			logger.Error(err, "Failed to get external URL, using fallback")
			task.Status.ExternalURL = task.GetExternalURL()
		} else {
			task.Status.ExternalURL = externalURL
		}
	}

	task.Status.Phase = kaiiov1alpha1.TaskPhaseRunning
	if err := r.Status().Update(ctx, task); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DevTaskReconciler) handleRunningPhase(ctx context.Context, task *kaiiov1alpha1.DevTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	deployName := GetDeploymentName(task)
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: task.Namespace}, deploy); err != nil {
		if errors.IsNotFound(err) {
			externalURL := ""
			if NeedsExternalRouting(task) {
				var err error
				externalURL, err = GetExternalURLWithGatewayIP(ctx, r.Client, task)
				if err != nil {
					externalURL = task.GetExternalURL()
				}
			}
			newDeploy := BuildDeployment(task, externalURL)
			if newDeploy == nil {
				return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed, "Failed to rebuild deployment")
			}
			if err := ctrl.SetControllerReference(task, newDeploy, r.Scheme); err != nil {
				logger.Error(err, "Failed to set controller reference for deployment")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, newDeploy); err != nil {
				logger.Error(err, "Failed to recreate deployment")
				return ctrl.Result{}, err
			}
			logger.Info("Deployment recreated", "deployment", newDeploy.Name)
			task.Status.Phase = kaiiov1alpha1.TaskPhaseCreating
			if err := r.Status().Update(ctx, task); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get deployment")
		return ctrl.Result{}, err
	}

	podList := &corev1.PodList{}
	labels := map[string]string{
		"framework":        GetFramework(task),
		"kai.io/task-type": "development",
	}
	if err := r.List(ctx, podList, client.InNamespace(task.Namespace), client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	for _, pod := range podList.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				if reason == "ImagePullBackOff" || reason == "ErrImagePull" ||
					reason == "CrashLoopBackOff" || reason == "Unknown" {
					return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed,
						fmt.Sprintf("Container failed: %s", reason))
				}
			}
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			task.Status.Message = "Pod pending"
			if err := r.Status().Update(ctx, task); err != nil {
				return ctrl.Result{}, err
			}
		case corev1.PodRunning:
			task.Status.Message = "Pod running"
			if err := r.Status().Update(ctx, task); err != nil {
				return ctrl.Result{}, err
			}
		case corev1.PodSucceeded:
			return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseCompleted, "Task completed")
		case corev1.PodFailed, corev1.PodUnknown:
			return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseFailed,
				fmt.Sprintf("Pod failed: %s", pod.Status.Reason))
		}
	}

	if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == 0 {
		return r.updateStatusAndReturn(ctx, task, kaiiov1alpha1.TaskPhaseCompleted, "Task completed")
	}

	if err := EnsureService(ctx, r.Client, r.Scheme, task); err != nil {
		logger.Error(err, "Failed to ensure service exists")
	}

	if NeedsExternalRouting(task) {
		if err := EnsureHTTPRoute(ctx, r.Client, r.Scheme, task); err != nil {
			logger.Error(err, "Failed to ensure HTTPRoute exists")
		}
		externalURL, err := GetExternalURLWithGatewayIP(ctx, r.Client, task)
		if err != nil {
			logger.Error(err, "Failed to get external URL, using fallback")
			task.Status.ExternalURL = task.GetExternalURL()
		} else {
			task.Status.ExternalURL = externalURL
		}
		if err := r.Status().Update(ctx, task); err != nil {
			logger.Error(err, "Failed to update external URL")
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *DevTaskReconciler) handleFinishedPhase(ctx context.Context, task *kaiiov1alpha1.DevTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if task.Status.EndTime == nil {
		now := metav1.Now()
		task.Status.EndTime = &now
		if err := r.Status().Update(ctx, task); err != nil {
			logger.Error(err, "Failed to update end time")
			return ctrl.Result{}, err
		}
	}

	// Ensure resources exist even in finished state
	if err := EnsureService(ctx, r.Client, r.Scheme, task); err != nil {
		logger.Error(err, "Failed to ensure service exists")
	}

	if NeedsExternalRouting(task) {
		if err := EnsureHTTPRoute(ctx, r.Client, r.Scheme, task); err != nil {
			logger.Error(err, "Failed to ensure HTTPRoute exists")
		}
	}

	if task.Spec.TTLSeconds != nil && task.Status.Phase == kaiiov1alpha1.TaskPhaseCompleted {
		go func() {
			time.Sleep(time.Duration(*task.Spec.TTLSeconds) * time.Second)
			if err := r.Delete(context.Background(), task); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete completed task")
			}
		}()
	}

	return ctrl.Result{}, nil
}

func (r *DevTaskReconciler) cleanupResources(ctx context.Context, task *kaiiov1alpha1.DevTask) error {
	logger := log.FromContext(ctx)

	deployName := GetDeploymentName(task)
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: task.Namespace}, deploy); err == nil {
		zero := int64(0)
		if err := r.Delete(ctx, deploy, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete deployment")
			return err
		}
	}

	if err := CleanupService(ctx, r.Client, task); err != nil {
		logger.Error(err, "Failed to cleanup service")
		return err
	}

	if err := CleanupHTTPRoute(ctx, r.Client, task); err != nil {
		logger.Error(err, "Failed to cleanup HTTPRoute")
		return err
	}

	return nil
}

func (r *DevTaskReconciler) updateStatusAndReturn(ctx context.Context, task *kaiiov1alpha1.DevTask, phase kaiiov1alpha1.TaskPhase, message string) (ctrl.Result, error) {
	task.Status.Phase = phase
	task.Status.Message = message
	if err := r.Status().Update(ctx, task); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
