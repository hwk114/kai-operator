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

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	"github.com/hwk114/kai/internal/controller/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetServiceNameForTask(task *kaiiov1alpha1.DevTask) string {
	return fmt.Sprintf("%s-%s-%s", task.Name, GetFramework(task), task.Namespace)
}

func GetDeploymentName(task *kaiiov1alpha1.DevTask) string {
	return fmt.Sprintf("%s-%s-%s", task.Name, GetFramework(task), task.Namespace)
}

func GetRouteName(task *kaiiov1alpha1.DevTask) string {
	return fmt.Sprintf("%s-%s-%s", task.Name, GetFramework(task), task.Namespace)
}

func GetServicePortForTask(task *kaiiov1alpha1.DevTask) int32 {
	framework := GetFramework(task)
	toolSpec, _ := GetToolSpecAndImage(task)
	return GetServicePort(framework, toolSpec.Port)
}

func BuildDeployment(task *kaiiov1alpha1.DevTask, externalURL string) *appsv1.Deployment {
	framework := GetFramework(task)
	toolSpec, image := GetToolSpecAndImage(task)
	if toolSpec.Image != "" {
		image = toolSpec.Image
	}

	labels := map[string]string{"kai.io/devtask": task.Name}
	for k, v := range builder.CommonLabels {
		labels[k] = v
	}
	labels["framework"] = framework
	labels["kai.io/task-type"] = "development"

	container := BuildDevContainer(task, image, externalURL)
	container.Ports = []corev1.ContainerPort{{ContainerPort: GetServicePortForTask(task), Name: "app"}}

	deploy := builder.BuildDeployment(task.Name, task.Namespace, framework, "development", image, 1, container, labels)

	vols, mounts := builder.BuildVolumes(task.Spec.Volumes, task.Spec.DataSources)
	deploy.Spec.Template.Spec.Volumes = vols
	deploy.Spec.Template.Spec.Containers[0].VolumeMounts = mounts

	return deploy
}

func BuildDevContainer(task *kaiiov1alpha1.DevTask, image string, externalURL string) corev1.Container {
	framework := GetFramework(task)

	env := []corev1.EnvVar{
		{Name: "KAI_TASK_NAME", Value: task.Name},
		{Name: "KAI_TASK_TYPE", Value: "development"},
		{Name: "KAI_FRAMEWORK", Value: framework},
	}

	var cmd []string
	var args []string

	if task.Spec.CodeServer != nil {
		toolSpec := &task.Spec.CodeServer.BaseToolSpec
		cmd = toolSpec.Command
		args = append(args, toolSpec.Args...)
		args = append(args, "--host", "0.0.0.0")

		hasPassword := task.Spec.CodeServer.Password != nil && *task.Spec.CodeServer.Password != ""
		if hasPassword {
			args = append(args, "--password", *task.Spec.CodeServer.Password)
		} else {
			args = append(args, "--auth", "none")
		}
		if task.Spec.CodeServer.Workspace != "" {
			args = append(args, "--workspace", task.Spec.CodeServer.Workspace)
		}
		if task.Spec.CodeServer != nil {
			// With URL rewrite in HTTPRoute, path is rewritten to / before reaching backend
			args = append(args, "--abs-proxy-base-path", "/")
		}
		for _, e := range toolSpec.Env {
			env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}
	if task.Spec.Jupyter != nil {
		toolSpec := &task.Spec.Jupyter.BaseToolSpec
		cmd = toolSpec.Command
		if len(cmd) == 0 {
			cmd = []string{"jupyter", "lab"}
		}
		args = append(args, toolSpec.Args...)
		if task.Spec.Jupyter.NotebookDir != "" {
			args = append(args, "--NotebookApp.notebook_dir="+task.Spec.Jupyter.NotebookDir)
		}
		baseURL := fmt.Sprintf("/%s/%s", framework, task.Name)
		args = append(args, "--ServerApp.base_url="+baseURL)
		args = append(args, "--ServerApp.token=''")
		args = append(args, "--ServerApp.allow_root=True")
		for _, e := range toolSpec.Env {
			env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}
	if task.Spec.NoVNC != nil {
		toolSpec := &task.Spec.NoVNC.BaseToolSpec
		if task.Spec.NoVNC.Display != "" {
			env = append(env, corev1.EnvVar{Name: "VNC_DISPLAY", Value: task.Spec.NoVNC.Display})
		}
		if task.Spec.NoVNC.Password != "" {
			env = append(env, corev1.EnvVar{Name: "VNC_PASSWORD", Value: task.Spec.NoVNC.Password})
		}
		cmd = toolSpec.Command
		args = append(args, toolSpec.Args...)
	}

	return corev1.Container{Name: "devtask", Image: image, Command: cmd, Args: args, Env: env}
}

func CreateService(ctx context.Context, c client.Client, scheme *runtime.Scheme, task *kaiiov1alpha1.DevTask) error {
	framework := GetFramework(task)
	labels := map[string]string{"kai.io/devtask": task.Name}
	for k, v := range builder.CommonLabels {
		labels[k] = v
	}
	labels["framework"] = framework
	labels["kai.io/task-type"] = "development"

	svcName := GetServiceNameForTask(task)
	return builder.EnsureService(ctx, c, task, scheme, svcName, task.Namespace, GetServicePortForTask(task), labels)
}

func EnsureService(ctx context.Context, c client.Client, scheme *runtime.Scheme, task *kaiiov1alpha1.DevTask) error {
	return CreateService(ctx, c, scheme, task)
}

func CleanupService(ctx context.Context, c client.Client, task *kaiiov1alpha1.DevTask) error {
	svcName := GetServiceNameForTask(task)
	return builder.DeleteService(ctx, c, svcName, task.Namespace)
}
