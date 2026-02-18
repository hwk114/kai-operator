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

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HuggingFaceSpec struct {
	ModelID   string `json:"modelId,omitempty"`
	Token     string `json:"token,omitempty"`
	MirrorURL string `json:"mirrorURL,omitempty"`
}

type VLLMSpec struct {
	BaseToolSpec         `json:",inline"`
	ModelPath            string           `json:"modelPath,omitempty"`
	ModelName            string           `json:"modelName,omitempty"`
	TensorParallelSize   int32            `json:"tensorParallelSize,omitempty"`
	GPUMemoryUtilization string           `json:"gpuMemoryUtilization,omitempty"`
	MaxModelLen          int32            `json:"maxModelLen,omitempty"`
	Port                 int32            `json:"port,omitempty"`
	HealthCheck          *HealthCheckSpec `json:"healthCheck,omitempty"`
	AutoScaling          *AutoScalingSpec `json:"autoScaling,omitempty"`
}

type LlamaCppSpec struct {
	BaseToolSpec `json:",inline"`
	ModelPath    string           `json:"modelPath,omitempty"`
	ContextSize  int32            `json:"contextSize,omitempty"`
	Threads      int32            `json:"threads,omitempty"`
	GPULayers    int32            `json:"gpuLayers,omitempty"`
	Port         int32            `json:"port,omitempty"`
	HealthCheck  *HealthCheckSpec `json:"healthCheck,omitempty"`
	AutoScaling  *AutoScalingSpec `json:"autoScaling,omitempty"`
}

type TGISpec struct {
	BaseToolSpec   `json:",inline"`
	ModelPath      string           `json:"modelPath,omitempty"`
	ModelID        string           `json:"modelId,omitempty"`
	MaxBatchSize   int32            `json:"maxBatchSize,omitempty"`
	MaxInputLen    int32            `json:"maxInputLen,omitempty"`
	MaxTotalTokens int32            `json:"maxTotalTokens,omitempty"`
	Port           int32            `json:"port,omitempty"`
	HealthCheck    *HealthCheckSpec `json:"healthCheck,omitempty"`
	AutoScaling    *AutoScalingSpec `json:"autoScaling,omitempty"`
}

type InferenceTaskSpec struct {
	Resources   *ResourceSpec    `json:"resources,omitempty"`
	Volumes     []VolumeSpec     `json:"volumes,omitempty"`
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
	RunAfter    []string         `json:"runAfter,omitempty"`
	TTLSeconds  *int32           `json:"ttlSecondsAfterFinished,omitempty"`

	Model    string `json:"model,omitempty"`
	Engine   string `json:"engine,omitempty"`
	HFMirror string `json:"hfMirror,omitempty"`

	VLLM     *VLLMSpec     `json:"vllm,omitempty"`
	LlamaCpp *LlamaCppSpec `json:"llama-cpp,omitempty"`
	TGI      *TGISpec      `json:"tgi,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=inference,categories=all
// +kubebuilder:group=kai.io
// +kubebuilder:version:name=v1alpha1
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".metadata.labels.framework"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="External",type="string",JSONPath=".status.externalURL"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type InferenceTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InferenceTaskSpec `json:"spec,omitempty"`
	Status TaskStatus        `json:"status,omitempty"`
}

func (r *InferenceTask) GetFramework() string {
	if r.Spec.VLLM != nil {
		return "vllm"
	}
	if r.Spec.LlamaCpp != nil {
		return "llama-cpp"
	}
	if r.Spec.TGI != nil {
		return "tgi"
	}
	return ""
}

func (r *InferenceTask) NeedsExternalRouting() bool {
	if r.Spec.VLLM != nil {
		return r.Spec.VLLM.Routing == nil || r.Spec.VLLM.Routing.Enabled
	}
	if r.Spec.LlamaCpp != nil {
		return r.Spec.LlamaCpp.Routing == nil || r.Spec.LlamaCpp.Routing.Enabled
	}
	if r.Spec.TGI != nil {
		return r.Spec.TGI.Routing == nil || r.Spec.TGI.Routing.Enabled
	}
	return false
}

func (r *InferenceTask) GetRoutingConfig() *RoutingSpec {
	if r.Spec.VLLM != nil && r.Spec.VLLM.Routing != nil {
		return r.Spec.VLLM.Routing
	}
	if r.Spec.LlamaCpp != nil && r.Spec.LlamaCpp.Routing != nil {
		return r.Spec.LlamaCpp.Routing
	}
	if r.Spec.TGI != nil && r.Spec.TGI.Routing != nil {
		return r.Spec.TGI.Routing
	}
	return nil
}

func (r *InferenceTask) GetGatewayNamespace() string {
	cfg := r.GetRoutingConfig()
	if cfg != nil && cfg.GatewayNamespace != "" {
		return cfg.GatewayNamespace
	}
	return "default"
}

func (r *InferenceTask) GetGatewayName() string {
	cfg := r.GetRoutingConfig()
	if cfg != nil && cfg.GatewayName != "" {
		return cfg.GatewayName
	}
	return "traefik"
}

func (r *InferenceTask) GetRoutePath() string {
	cfg := r.GetRoutingConfig()
	if cfg != nil && cfg.PathPrefix != "" {
		return cfg.PathPrefix
	}
	return fmt.Sprintf("/%s/%s", r.GetFramework(), r.Name)
}

func (r *InferenceTask) GetExternalURL() string {
	if !r.NeedsExternalRouting() {
		return ""
	}
	cfg := r.GetRoutingConfig()
	gatewayName := "traefik-gateway"
	gatewayNamespace := "default"
	if cfg != nil && cfg.GatewayName != "" {
		gatewayName = cfg.GatewayName
	}
	if cfg != nil && cfg.GatewayNamespace != "" {
		gatewayNamespace = cfg.GatewayNamespace
	}
	return fmt.Sprintf("http://%s.%s.svc%s", gatewayName, gatewayNamespace, r.GetRoutePath())
}

// +kubebuilder:object:root=true
type InferenceTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InferenceTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InferenceTask{}, &InferenceTaskList{})
}
