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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	FinalizerName = "inference.kai.io/finalizer"
	RequeueDelay  = 30
)

type InferenceTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kai.io,resources=inferencetasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kai.io,resources=inferencetasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kai.io,resources=inferencetasks/finalizers,verbs=update

func (r *InferenceTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Force error to check if reconcile is called
	_ = fmt.Errorf("DEBUG: Inference reconcile called for %s/%s", req.Namespace, req.Name)
	log.Info("=== INFERENCE RECONCILE START ===", "reqName", req.Name, "reqNS", req.Namespace)

	inference := &kaiiov1alpha1.InferenceTask{}
	err := r.Get(ctx, req.NamespacedName, inference)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get InferenceTask")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling InferenceTask", "name", inference.Name, "phase", inference.Status.Phase, "deletionTimestamp", inference.DeletionTimestamp)

	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	if inference.Labels == nil {
		inference.Labels = make(map[string]string)
	}
	if inference.Labels["framework"] != framework {
		inference.Labels["framework"] = framework
		if err := r.Update(ctx, inference); err != nil {
			log.Error(err, "Failed to update framework label")
			return ctrl.Result{}, err
		}
		log.Info("Framework label updated, requeuing")
		return ctrl.Result{}, nil
	}

	if !inference.DeletionTimestamp.IsZero() {
		log.Info("InferenceTask is being deleted, handling deletion")
		return r.handleDeletion(ctx, inference)
	}

	if !controllerutil.ContainsFinalizer(inference, FinalizerName) {
		controllerutil.AddFinalizer(inference, FinalizerName)
		if err := r.Update(ctx, inference); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Finalizer added, waiting for next reconcile")
		return ctrl.Result{}, nil
	}

	switch inference.Status.Phase {
	case "", kaiiov1alpha1.TaskPhasePending:
		return r.handlePendingPhase(ctx, inference)
	case kaiiov1alpha1.TaskPhaseCreating:
		return r.handleCreatingPhase(ctx, inference)
	case kaiiov1alpha1.TaskPhaseRunning:
		return r.handleRunningPhase(ctx, inference)
	case kaiiov1alpha1.TaskPhaseCompleted, kaiiov1alpha1.TaskPhaseFailed:
		return r.handleFinishedPhase(ctx, inference)
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *InferenceTaskReconciler) handleDeletion(ctx context.Context, inference *kaiiov1alpha1.InferenceTask) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(inference, FinalizerName) {
		if err := r.cleanupResources(ctx, inference); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(inference, FinalizerName)
		if err := r.Update(ctx, inference); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *InferenceTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kaiiov1alpha1.InferenceTask{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&gatewayapiv1.HTTPRoute{}).
		Complete(r)
}
