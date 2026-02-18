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

package inference

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

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *InferenceTaskReconciler) handlePendingPhase(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := r.Get(ctx, types.NamespacedName{Name: inference.Name, Namespace: inference.Namespace}, inference); err != nil {
		logger.Error(err, "Failed to get latest InferenceTask")
		return ctrl.Result{}, err
	}

	if inference.Status.Phase != "" && inference.Status.Phase != kaiiov1alpha1.TaskPhasePending {
		return ctrl.Result{}, nil
	}

	inference.Status.Phase = kaiiov1alpha1.TaskPhaseCreating
	now := metav1.Now()
	inference.Status.StartTime = &now
	if err := r.Status().Update(ctx, inference); err != nil {
		logger.Error(err, "Failed to update task status")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *InferenceTaskReconciler) handleCreatingPhase(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	deploy := BuildInferenceDeployment(inference)
	if deploy == nil {
		logger.Error(nil, "Failed to build deployment")
		return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed, "Failed to build deployment")
	}

	if err := ctrl.SetControllerReference(inference, deploy, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference for deployment")
		return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed, err.Error())
	}

	existingDeploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, existingDeploy)
	if err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, deploy); err != nil {
			logger.Error(err, "Failed to create deployment")
			return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed, err.Error())
		}
		logger.Info("Deployment created", "deployment", deploy.Name)
	} else if err != nil {
		logger.Error(err, "Failed to get existing deployment")
		return ctrl.Result{}, err
	}

	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	podName := fmt.Sprintf("inference-%s-%s", framework, inference.Name)
	inference.Status.PodName = podName

	if err := r.ensureService(ctx, inference); err != nil {
		logger.Error(err, "Failed to create service")
		return ctrl.Result{}, err
	}

	if inference.NeedsExternalRouting() {
		serviceName := getServiceName(inference)
		servicePort := GetInferencePort(inference)
		if err := r.ensureHTTPRoute(ctx, inference, serviceName, servicePort); err != nil {
			logger.Error(err, "Failed to create HTTPRoute, continuing anyway")
		}
		if inference.Status.ExternalURL == "" {
			inference.Status.ExternalURL = getExternalURL(ctx, r.Client, inference)
		}
	}

	inference.Status.Phase = kaiiov1alpha1.TaskPhaseRunning
	if err := r.Status().Update(ctx, inference); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InferenceTaskReconciler) handleRunningPhase(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	deployName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: inference.Namespace}, deploy); err != nil {
		if errors.IsNotFound(err) {
			newDeploy := BuildInferenceDeployment(inference)
			if newDeploy == nil {
				return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed, "Failed to rebuild deployment")
			}
			if err := ctrl.SetControllerReference(inference, newDeploy, r.Scheme); err != nil {
				logger.Error(err, "Failed to set controller reference for deployment")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, newDeploy); err != nil {
				logger.Error(err, "Failed to recreate deployment")
				return ctrl.Result{}, err
			}
			logger.Info("Deployment recreated", "deployment", newDeploy.Name)
			inference.Status.Phase = kaiiov1alpha1.TaskPhaseCreating
			if err := r.Status().Update(ctx, inference); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get deployment")
		return ctrl.Result{}, err
	}

	podList := &corev1.PodList{}
	labels := map[string]string{
		"kai.io/inference": inference.Name,
		"kai.io/task-type": "inference",
	}
	if err := r.List(ctx, podList, client.InNamespace(inference.Namespace), client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	for _, pod := range podList.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				if reason == "ImagePullBackOff" || reason == "ErrImagePull" ||
					reason == "CrashLoopBackOff" || reason == "Unknown" {
					return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed,
						fmt.Sprintf("Container failed: %s", reason))
				}
			}
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			inference.Status.Message = "Pod pending"
			if err := r.Status().Update(ctx, inference); err != nil {
				return ctrl.Result{}, err
			}
		case corev1.PodRunning:
			inference.Status.Message = "Pod running"
			if err := r.Status().Update(ctx, inference); err != nil {
				return ctrl.Result{}, err
			}
		case corev1.PodSucceeded:
			return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseCompleted, "Task completed")
		case corev1.PodFailed, corev1.PodUnknown:
			return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseFailed,
				fmt.Sprintf("Pod failed: %s", pod.Status.Reason))
		}
	}

	if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == 0 {
		return r.updateStatusAndReturn(ctx, inference, kaiiov1alpha1.TaskPhaseCompleted, "Task completed")
	}

	if err := r.ensureService(ctx, inference); err != nil {
		logger.Error(err, "Failed to ensure service exists")
	}

	if inference.NeedsExternalRouting() && inference.DeletionTimestamp.IsZero() {
		logger.Info("NeedsExternalRouting is true, ensuring HTTPRoute and externalURL")
		if err := r.ensureHTTPRoute(ctx, inference, getServiceName(inference), GetInferencePort(inference)); err != nil {
			logger.Error(err, "Failed to ensure HTTPRoute exists")
		}
		externalURL := getExternalURL(ctx, r.Client, inference)
		logger.Info("External URL", "url", externalURL)
		if externalURL != "" {
			inference.Status.ExternalURL = externalURL
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *InferenceTaskReconciler) handleFinishedPhase(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if inference.Status.EndTime == nil {
		now := metav1.Now()
		inference.Status.EndTime = &now
		if err := r.Status().Update(ctx, inference); err != nil {
			logger.Error(err, "Failed to update end time")
			return ctrl.Result{}, err
		}
	}

	if inference.Spec.TTLSeconds != nil && inference.Status.Phase == kaiiov1alpha1.TaskPhaseCompleted {
		go func() {
			time.Sleep(time.Duration(*inference.Spec.TTLSeconds) * time.Second)
			if err := r.Delete(context.Background(), inference); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete completed task")
			}
		}()
	}

	return ctrl.Result{}, nil
}

func (r *InferenceTaskReconciler) cleanupResources(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) error {
	logger := log.FromContext(ctx)
	logger.Info("Cleaning up resources for inference", "name", inference.Name)

	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	deployName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: inference.Namespace}, deploy); err == nil {
		zero := int64(0)
		if err := r.Delete(ctx, deploy, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete deployment")
			return err
		}
	}

	if err := r.cleanupService(ctx, inference); err != nil {
		logger.Error(err, "Failed to cleanup service")
		return err
	}

	if err := r.cleanupHTTPRoute(ctx, inference); err != nil {
		logger.Error(err, "Failed to cleanup HTTPRoute")
		return err
	}

	return nil
}

func (r *InferenceTaskReconciler) updateStatusAndReturn(ctx context.Context, inference *kaiiov1alpha1.InferenceTask, phase kaiiov1alpha1.TaskPhase, message string) (ctrl.Result, error) {
	inference.Status.Phase = phase
	inference.Status.Message = message
	if err := r.Status().Update(ctx, inference); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *InferenceTaskReconciler) ensureService(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) error {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	labels := map[string]string{"kai.io/inference": inference.Name}
	labels["kai.io/task-type"] = "inference"
	labels["framework"] = framework

	svcName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)
	return ensureService(ctx, r.Client, r.Scheme, inference, svcName, inference.Namespace, GetInferencePort(inference), labels)
}

func (r *InferenceTaskReconciler) cleanupService(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) error {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	svcName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)
	return deleteService(ctx, r.Client, svcName, inference.Namespace)
}

func (r *InferenceTaskReconciler) ensureHTTPRoute(ctx context.Context, inference *kaiiov1alpha1.InferenceTask, serviceName string, servicePort int32) error {
	return ensureHTTPRoute(ctx, r.Client, r.Scheme, inference, serviceName, servicePort)
}

func (r *InferenceTaskReconciler) cleanupHTTPRoute(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) error {
	return cleanupHTTPRoute(ctx, r.Client, inference)
}
