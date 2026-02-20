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
	kaiiov1alpha1 "github.com/hwk114/kai-operator/api/v1alpha1"
	"github.com/hwk114/kai-operator/internal/config"
)

func GetFramework(task *kaiiov1alpha1.DevTask) string {
	if task.Spec.CodeServer != nil {
		return FrameworkCodeServer
	}
	if task.Spec.Jupyter != nil {
		return FrameworkJupyter
	}
	if task.Spec.NoVNC != nil {
		return FrameworkNoVNC
	}
	return ""
}

func GetToolSpecAndImage(task *kaiiov1alpha1.DevTask) (*kaiiov1alpha1.BaseToolSpec, string) {
	framework := GetFramework(task)

	if task.Spec.CodeServer != nil {
		return &task.Spec.CodeServer.BaseToolSpec, GetImageByFramework(framework, task.Spec.CodeServer.Version)
	}
	if task.Spec.Jupyter != nil {
		return &task.Spec.Jupyter.BaseToolSpec, GetImageByFramework(framework, task.Spec.Jupyter.Version)
	}
	if task.Spec.NoVNC != nil {
		return &task.Spec.NoVNC.BaseToolSpec, GetImageByFramework(framework, task.Spec.NoVNC.Version)
	}

	return &kaiiov1alpha1.BaseToolSpec{}, "alpine:latest"
}

func GetImageByFramework(framework, version string) string {
	return config.GetFrameworkImage(framework)
}

func GetServicePort(framework string, configuredPort int32) int32 {
	if configuredPort > 0 {
		return configuredPort
	}
	if port, ok := DefaultPorts[framework]; ok {
		return port
	}
	return 0
}
