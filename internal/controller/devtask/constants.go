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
	"github.com/hwk114/kai-operator/internal/config"
)

const (
	FinalizerName = "devtask.kai.io/finalizer"
	RequeueDelay  = 30
)

const (
	FrameworkCodeServer = "codeserver"
	FrameworkJupyter    = "jupyter"
	FrameworkNoVNC      = "novnc"
	FrameworkVLLM       = "vllm"
	FrameworkLlamaCpp   = "llama.cpp"
	FrameworkTGI        = "tgi"
)

const (
	DefaultGatewayName      = "traefik-gateway"
	DefaultGatewayNamespace = "default"
)

const (
	DefaultCPU     = "500m"
	DefaultMemory  = "512Mi"
	DefaultGPU     = "0"
	DefaultGPUUtil = "0.9"
)

var DefaultPorts = map[string]int32{
	FrameworkCodeServer: config.GetFrameworkPort("codeserver"),
	FrameworkJupyter:    config.GetFrameworkPort("jupyter"),
	FrameworkNoVNC:      config.GetFrameworkPort("novnc"),
	FrameworkVLLM:       config.GetFrameworkPort("vllm"),
	FrameworkTGI:        config.GetFrameworkPort("tgi"),
	FrameworkLlamaCpp:   config.GetFrameworkPort("llama.cpp"),
}
