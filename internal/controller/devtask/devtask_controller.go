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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kaiiov1alpha1 "github.com/hwk114/kai-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type DevTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kai.io,resources=devtasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kai.io,resources=devtasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kai.io,resources=devtasks/finalizers,verbs=update

func (r *DevTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	task := &kaiiov1alpha1.DevTask{}
	err := r.Get(ctx, req.NamespacedName, task)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get DevTask")
		return ctrl.Result{}, err
	}

	if !task.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, task)
	}

	if !controllerutil.ContainsFinalizer(task, FinalizerName) {
		controllerutil.AddFinalizer(task, FinalizerName)
		if err := r.Update(ctx, task); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch task.Status.Phase {
	case "", kaiiov1alpha1.TaskPhasePending:
		return r.handlePendingPhase(ctx, task)
	case kaiiov1alpha1.TaskPhaseCreating:
		return r.handleCreatingPhase(ctx, task)
	case kaiiov1alpha1.TaskPhaseRunning:
		return r.handleRunningPhase(ctx, task)
	case kaiiov1alpha1.TaskPhaseCompleted, kaiiov1alpha1.TaskPhaseFailed:
		return r.handleFinishedPhase(ctx, task)
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *DevTaskReconciler) handleDeletion(ctx context.Context, task *kaiiov1alpha1.DevTask) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(task, FinalizerName) {
		if err := r.cleanupResources(ctx, task); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(task, FinalizerName)
		if err := r.Update(ctx, task); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *DevTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kaiiov1alpha1.DevTask{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&gatewayapiv1.HTTPRoute{}).
		Complete(r)
}
