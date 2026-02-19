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

package evaltask

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kaiv1alpha1 "github.com/hwk114/kai-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	FinalizerName = "eval.kai.io/finalizer"
	RequeueDelay  = 30
)

type EvalTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kai.io,resources=evaltasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kai.io,resources=evaltasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kai.io,resources=evaltasks/finalizers,verbs=update

func (r *EvalTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	eval := &kaiv1alpha1.EvalTask{}
	err := r.Get(ctx, req.NamespacedName, eval)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get EvalTask")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling EvalTask", "name", eval.Name, "phase", eval.Status.Phase)

	if !eval.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, eval)
	}

	if !controllerutil.ContainsFinalizer(eval, FinalizerName) {
		controllerutil.AddFinalizer(eval, FinalizerName)
		if err := r.Update(ctx, eval); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	switch eval.Status.Phase {
	case "", kaiv1alpha1.TaskPhasePending:
		return r.handlePendingPhase(ctx, eval)
	case kaiv1alpha1.TaskPhaseRunning:
		return r.handleRunningPhase(ctx, eval)
	case kaiv1alpha1.TaskPhaseCompleted, kaiv1alpha1.TaskPhaseFailed:
		return r.handleFinishedPhase(ctx, eval)
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *EvalTaskReconciler) handlePendingPhase(ctx context.Context, eval *kaiv1alpha1.EvalTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handlePendingPhase", "name", eval.Name)

	framework := eval.GetFramework()
	if framework == "" {
		framework = "opencompass"
	}

	if eval.Labels == nil {
		eval.Labels = make(map[string]string)
	}
	eval.Labels["framework"] = framework

	if err := r.Update(ctx, eval); err != nil {
		logger.Error(err, "Failed to update task labels")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	eval.Status.Phase = kaiv1alpha1.TaskPhaseRunning
	now := metav1.Now()
	eval.Status.StartTime = &now
	if err := r.Status().Update(ctx, eval); err != nil {
		logger.Error(err, "Failed to update task status")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *EvalTaskReconciler) handleRunningPhase(ctx context.Context, eval *kaiv1alpha1.EvalTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handleRunningPhase", "name", eval.Name)

	framework := eval.GetFramework()
	if framework == "" {
		framework = "opencompass"
	}

	if err := r.ensureJob(ctx, eval, framework); err != nil {
		logger.Error(err, "Failed to create job")
		return ctrl.Result{}, err
	}

	if err := r.checkJobStatus(ctx, eval, framework); err != nil {
		logger.Error(err, "Failed to check job status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *EvalTaskReconciler) checkJobStatus(ctx context.Context, eval *kaiv1alpha1.EvalTask, framework string) error {
	logger := log.FromContext(ctx)

	jobName := fmt.Sprintf("eval-%s-%s", framework, eval.Name)
	job := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: eval.Namespace}, job); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	eval.Status.PodName = job.Name

	if job.Status.Active > 0 {
		eval.Status.Phase = kaiv1alpha1.TaskPhaseRunning
		eval.Status.Message = "Job is running"
		return r.Status().Update(ctx, eval)
	}

	if job.Status.Succeeded > 0 {
		eval.Status.Phase = kaiv1alpha1.TaskPhaseCompleted
		eval.Status.Message = "Evaluation completed successfully"
		logger.Info("Job completed successfully", "name", job.Name)
		return r.Status().Update(ctx, eval)
	}

	if job.Status.Failed > 0 {
		eval.Status.Phase = kaiv1alpha1.TaskPhaseFailed
		msg := "Job failed"
		if len(job.Status.Conditions) > 0 {
			msg = job.Status.Conditions[0].Message
		}
		eval.Status.Message = msg
		logger.Error(nil, "Job failed", "name", job.Name, "message", msg)
		return r.Status().Update(ctx, eval)
	}

	return nil
}

func (r *EvalTaskReconciler) handleFinishedPhase(ctx context.Context, eval *kaiv1alpha1.EvalTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if eval.Status.EndTime == nil {
		now := metav1.Now()
		eval.Status.EndTime = &now
		if err := r.Status().Update(ctx, eval); err != nil {
			logger.Error(err, "Failed to update end time")
			return ctrl.Result{}, err
		}
	}

	if err := r.cleanupResources(ctx, eval); err != nil {
		logger.Error(err, "Failed to cleanup resources")
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(eval, FinalizerName)
	if err := r.Update(ctx, eval); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EvalTaskReconciler) handleDeletion(ctx context.Context, eval *kaiv1alpha1.EvalTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(eval, FinalizerName) {
		if err := r.cleanupResources(ctx, eval); err != nil {
			logger.Error(err, "Failed to cleanup resources")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(eval, FinalizerName)
		if err := r.Update(ctx, eval); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *EvalTaskReconciler) cleanupResources(ctx context.Context, eval *kaiv1alpha1.EvalTask) error {
	logger := log.FromContext(ctx)

	framework := eval.GetFramework()
	if framework == "" {
		framework = "opencompass"
	}

	jobName := fmt.Sprintf("eval-%s-%s", framework, eval.Name)
	job := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: eval.Namespace}, job); err == nil {
		zero := int64(0)
		if err := r.Delete(ctx, job, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete job")
			return err
		}
	}

	return nil
}

func (r *EvalTaskReconciler) ensureJob(ctx context.Context, eval *kaiv1alpha1.EvalTask, framework string) error {
	logger := log.FromContext(ctx)

	jobName := fmt.Sprintf("eval-%s-%s", framework, eval.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: eval.Namespace}, job)
	if err == nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	spec := eval.Spec.OpenCompass
	if spec == nil {
		spec = &kaiv1alpha1.OpenCompassSpec{}
	}

	job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: eval.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kai",
				"app.kubernetes.io/managed-by": "kai-operator",
				"kai.io/eval":                  eval.Name,
				"kai.io/task-type":             "eval",
				"framework":                    framework,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptrTo(int32(300)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       "kai",
						"app.kubernetes.io/managed-by": "kai-operator",
						"kai.io/eval":                  eval.Name,
						"kai.io/task-type":             "eval",
						"framework":                    framework,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "eval",
							Image: "opencompass/eval:latest",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
								{Name: "output", MountPath: "/output"},
								{Name: "models", MountPath: "/models"},
							},
							Command: []string{"python"},
							Args:    buildEvalArgs(spec),
							Env: []corev1.EnvVar{
								{Name: "KAI_TASK_NAME", Value: eval.Name},
								{Name: "KAI_TASK_TYPE", Value: "eval"},
								{Name: "KAI_FRAMEWORK", Value: framework},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "output", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "models", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
				},
			},
		},
	}

	if spec.Resources != nil {
		if spec.Resources.CPU != "" {
			job.Spec.Template.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: resource.MustParse(spec.Resources.CPU),
			}
		}
		if spec.Resources.GPU != "" {
			job.Spec.Template.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceName("nvidia.com/gpu"): resource.MustParse(spec.Resources.GPU),
			}
		}
	}

	if err := ctrl.SetControllerReference(eval, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return err
	}

	if err := r.Create(ctx, job); err != nil {
		logger.Error(err, "Failed to create job")
		return err
	}

	logger.Info("Job created", "name", job.Name)
	return nil
}

func buildEvalArgs(spec *kaiv1alpha1.OpenCompassSpec) []string {
	var args []string

	args = append(args, "-m", "opencompass")

	if len(spec.Models) > 0 {
		model := spec.Models[0]
		args = append(args, "--models", model.Model)
		if model.ModelPath != "" {
			args = append(args, "--model-path", model.ModelPath)
		}
		if model.Template != "" {
			args = append(args, "--template", model.Template)
		}
	}

	if len(spec.Datasets) > 0 {
		args = append(args, "--datasets", spec.Datasets[0])
	}

	if spec.WorkDir != "" {
		args = append(args, "--work-dir", spec.WorkDir)
	}

	if spec.OutputDir != "" {
		args = append(args, "--output", spec.OutputDir)
	}

	if spec.Debug {
		args = append(args, "--debug")
	}

	return args
}

func ptrTo[T any](v T) *T {
	return &v
}

// SetupWithManager sets up the controller with the Manager.
func (r *EvalTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kaiv1alpha1.EvalTask{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Named("evaltask").
		Complete(r)
}
