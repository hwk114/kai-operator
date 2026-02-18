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

package finetunetask

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

	kaiv1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	FinalizerName = "finetune.kai.io/finalizer"
	RequeueDelay  = 30
)

type FineTuneTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kai.io,resources=finetunetasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kai.io,resources=finetunetasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kai.io,resources=finetunetasks/finalizers,verbs=update

func (r *FineTuneTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("=== Reconcile called ===", "req", req)

	finetune := &kaiv1alpha1.FineTuneTask{}
	err := r.Get(ctx, req.NamespacedName, finetune)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("FineTuneTask not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get FineTuneTask")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling FineTuneTask", "name", finetune.Name, "phase", finetune.Status.Phase, "uid", finetune.UID)

	if !finetune.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, finetune)
	}

	if !controllerutil.ContainsFinalizer(finetune, FinalizerName) {
		controllerutil.AddFinalizer(finetune, FinalizerName)
		if err := r.Update(ctx, finetune); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	switch finetune.Status.Phase {
	case "", kaiv1alpha1.TaskPhasePending:
		return r.handlePendingPhase(ctx, finetune)
	case kaiv1alpha1.TaskPhaseRunning:
		return r.handleRunningPhase(ctx, finetune)
	case kaiv1alpha1.TaskPhaseCompleted, kaiv1alpha1.TaskPhaseFailed:
		return r.handleFinishedPhase(ctx, finetune)
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *FineTuneTaskReconciler) handlePendingPhase(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handlePendingPhase", "name", finetune.Name)

	framework := finetune.GetFramework()
	if framework == "" {
		framework = "llamafactory"
	}

	if finetune.Labels == nil {
		finetune.Labels = make(map[string]string)
	}
	finetune.Labels["framework"] = framework

	if err := r.Update(ctx, finetune); err != nil {
		logger.Error(err, "Failed to update task labels")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	finetune.Status.Phase = kaiv1alpha1.TaskPhaseRunning
	now := metav1.Now()
	finetune.Status.StartTime = &now
	if err := r.Status().Update(ctx, finetune); err != nil {
		logger.Error(err, "Failed to update task status")
		return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *FineTuneTaskReconciler) handleRunningPhase(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handleRunningPhase", "name", finetune.Name)

	framework := finetune.GetFramework()
	if framework == "" {
		framework = "llamafactory"
	}

	if err := r.ensureJob(ctx, finetune, framework); err != nil {
		logger.Error(err, "Failed to create job")
		return ctrl.Result{}, err
	}

	if err := r.ensureWebUI(ctx, finetune, framework); err != nil {
		logger.Error(err, "Failed to create web UI")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Duration(RequeueDelay) * time.Second}, nil
}

func (r *FineTuneTaskReconciler) handleFinishedPhase(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if finetune.Status.EndTime == nil {
		now := metav1.Now()
		finetune.Status.EndTime = &now
		if err := r.Status().Update(ctx, finetune); err != nil {
			logger.Error(err, "Failed to update end time")
			return ctrl.Result{}, err
		}
	}

	if err := r.cleanupResources(ctx, finetune); err != nil {
		logger.Error(err, "Failed to cleanup resources")
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(finetune, FinalizerName)
	if err := r.Update(ctx, finetune); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FineTuneTaskReconciler) handleDeletion(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(finetune, FinalizerName) {
		if err := r.cleanupResources(ctx, finetune); err != nil {
			logger.Error(err, "Failed to cleanup resources")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(finetune, FinalizerName)
		if err := r.Update(ctx, finetune); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *FineTuneTaskReconciler) cleanupResources(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask) error {
	logger := log.FromContext(ctx)

	framework := finetune.GetFramework()
	if framework == "" {
		framework = "llamafactory"
	}

	jobName := fmt.Sprintf("finetune-%s-%s", framework, finetune.Name)
	job := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: finetune.Namespace}, job); err == nil {
		zero := int64(0)
		if err := r.Delete(ctx, job, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete job")
			return err
		}
	}

	svcName := fmt.Sprintf("finetune-%s-%s", framework, finetune.Name)
	svc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{Name: svcName, Namespace: finetune.Namespace}, svc); err == nil {
		if err := r.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete service")
			return err
		}
	}

	deployName := fmt.Sprintf("finetune-%s-%s-ui", framework, finetune.Name)
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: deployName, Namespace: finetune.Namespace}, deploy); err == nil {
		if err := r.Delete(ctx, deploy); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete deployment")
			return err
		}
	}

	httproute := &gatewayapiv1.HTTPRoute{}
	if err := r.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("finetune-%s-%s", framework, finetune.Name), Namespace: finetune.Namespace}, httproute); err == nil {
		if err := r.Delete(ctx, httproute); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete HTTPRoute")
			return err
		}
	}

	return nil
}

func (r *FineTuneTaskReconciler) ensureJob(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask, framework string) error {
	logger := log.FromContext(ctx)

	jobName := fmt.Sprintf("finetune-%s-%s", framework, finetune.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: finetune.Namespace}, job)
	if err == nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	spec := finetune.Spec.LlamaFactory
	if spec == nil {
		spec = &kaiv1alpha1.LlamaFactorySpec{}
	}

	job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: finetune.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kai",
				"app.kubernetes.io/managed-by": "kai-operator",
				"kai.io/finetune":              finetune.Name,
				"kai.io/task-type":             "finetune",
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
						"kai.io/finetune":              finetune.Name,
						"kai.io/task-type":             "finetune",
						"framework":                    framework,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "train",
							Image: "hiyouga/llamafactory:latest",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
								{Name: "output", MountPath: "/output"},
								{Name: "models", MountPath: "/models"},
							},
							Command: []string{"llamafactory-cli"},
							Args:    buildTrainArgs(finetune),
							Env: []corev1.EnvVar{
								{Name: "KAI_TASK_NAME", Value: finetune.Name},
								{Name: "KAI_TASK_TYPE", Value: "finetune"},
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

	if finetune.Spec.Resources != nil {
		if finetune.Spec.Resources.CPU != "" {
			job.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(finetune.Spec.Resources.CPU),
			}
		}
		if finetune.Spec.Resources.GPU != "" {
			job.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceName("nvidia.com/gpu"): resource.MustParse(finetune.Spec.Resources.GPU),
			}
		}
	}

	if err := ctrl.SetControllerReference(finetune, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference for job")
		return err
	}

	if err := r.Create(ctx, job); err != nil {
		logger.Error(err, "Failed to create job")
		return err
	}

	logger.Info("Job created", "job", jobName)
	return nil
}

func buildTrainArgs(finetune *kaiv1alpha1.FineTuneTask) []string {
	spec := finetune.Spec.LlamaFactory
	if spec == nil {
		spec = &kaiv1alpha1.LlamaFactorySpec{}
	}

	args := []string{
		"train",
		"--stage", "sft",
		"--do_train", "true",
		"--finetuning_type", "lora",
	}

	if spec.ModelPath != "" {
		args = append(args, "--model_name_or_path", spec.ModelPath)
	}
	if spec.TrainData != "" {
		args = append(args, "--dataset", spec.TrainData)
	}
	if spec.ValidData != "" {
		args = append(args, "--dataset_format", "alpaca")
	}
	if spec.OutputDir != "" {
		args = append(args, "--output_dir", spec.OutputDir)
	} else {
		args = append(args, "--output_dir", "/output")
	}

	if spec.BatchSize > 0 {
		args = append(args, "--per_device_train_batch_size", fmt.Sprintf("%d", spec.BatchSize))
	}
	if spec.LearningRate != "" {
		args = append(args, "--learning_rate", spec.LearningRate)
	} else {
		args = append(args, "--learning_rate", "5e-5")
	}
	if spec.NumEpochs > 0 {
		args = append(args, "--num_train_epochs", fmt.Sprintf("%d", spec.NumEpochs))
	} else {
		args = append(args, "--num_train_epochs", "3")
	}

	args = append(args, "--loraplus_lr_ratio", "1.0")
	args = append(args, "--fp16", "true")

	return args
}

func (r *FineTuneTaskReconciler) ensureWebUI(ctx context.Context, finetune *kaiv1alpha1.FineTuneTask, framework string) error {
	logger := log.FromContext(ctx)

	svcName := fmt.Sprintf("finetune-%s-%s", framework, finetune.Name)
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: svcName, Namespace: finetune.Namespace}, svc)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	svc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: finetune.Namespace,
			Labels: map[string]string{
				"kai.io/finetune": finetune.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 8000},
			},
			Selector: map[string]string{
				"kai.io/finetune": finetune.Name,
			},
		},
	}

	if err := ctrl.SetControllerReference(finetune, svc, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, svc); err != nil {
		logger.Error(err, "Failed to create service")
		return err
	}

	deployName := fmt.Sprintf("finetune-%s-%s-ui", framework, finetune.Name)
	deploy := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: deployName, Namespace: finetune.Namespace}, deploy)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	deploy = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: finetune.Namespace,
			Labels: map[string]string{
				"kai.io/finetune": finetune.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrTo(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kai.io/finetune": finetune.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kai.io/finetune": finetune.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "webui",
							Image:   "hiyouga/llamafactory:latest",
							Command: []string{"python", "-m", "llamafactory.api.webui"},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 8000},
							},
							Env: []corev1.EnvVar{
								{Name: "GRADIO_SERVER_NAME", Value: "0.0.0.0"},
								{Name: "GRADIO_SERVER_PORT", Value: "8000"},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(finetune, deploy, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, deploy); err != nil {
		logger.Error(err, "Failed to create web UI deployment")
		return err
	}

	logger.Info("Web UI created", "deployment", deployName)
	return nil
}

func ptrTo[T any](v T) *T {
	return &v
}

func (r *FineTuneTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kaiv1alpha1.FineTuneTask{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&gatewayapiv1.HTTPRoute{}).
		Complete(r)
}
