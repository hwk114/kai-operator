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

const (
	FinalizerName = "devtask.kai.io/finalizer"
	RequeueDelay  = 30
)

const (
	FrameworkCodeServer = "codeserver"
	FrameworkJupyter    = "jupyter"
	FrameworkNoVNC      = "novnc"
	FrameworkVLLM       = "vllm"
	FrameworkLlamaCpp   = "llama-cpp"
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
	DefaultPort    = int32(8080)
	DefaultGPUUtil = "0.9"
)

var ToolImages = map[string]map[string]string{
	FrameworkCodeServer: {
		"default": "codercom/code-server:latest",
	},
	FrameworkJupyter: {
		"default":    "jupyter/base-notebook:python-3.11",
		"minimal":    "jupyter/base-notebook:python-3.11",
		"tensorflow": "jupyter/tensorflow-notebook:latest",
	},
	FrameworkNoVNC: {
		"default": "novnc:20260214-dev",
	},
	FrameworkVLLM: {
		"default": "vllm/vllm-openai:nightly-aarch64",
	},
	FrameworkLlamaCpp: {
		"default": "ghcr.io/ggerganov/llama.cpp:server",
	},
	FrameworkTGI: {
		"default": "ghcr.io/huggingface/text-generation-inference:latest",
	},
}

var DefaultPorts = map[string]int32{
	FrameworkCodeServer: 8080,
	FrameworkJupyter:    8888,
	FrameworkNoVNC:      6081,
	FrameworkVLLM:       8000,
	FrameworkTGI:        3000,
}
