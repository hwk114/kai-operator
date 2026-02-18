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

type TaskPhase string

const (
	TaskPhasePending     TaskPhase = "pending"
	TaskPhaseCreating    TaskPhase = "creating"
	TaskPhaseRunning     TaskPhase = "running"
	TaskPhaseCompleted   TaskPhase = "completed"
	TaskPhaseFailed      TaskPhase = "failed"
	TaskPhaseTerminating TaskPhase = "terminating"
)

type RoutingSpec struct {
	Enabled          bool     `json:"enabled,omitempty"`
	GatewayName      string   `json:"gatewayName,omitempty"`
	GatewayNamespace string   `json:"gatewayNamespace,omitempty"`
	PathPrefix       string   `json:"pathPrefix,omitempty"`
	ExternalHost     string   `json:"externalHost,omitempty"`
	ExternalPort     int32    `json:"externalPort,omitempty"`
	GatewayPort      int32    `json:"gatewayPort,omitempty"`
	Hostnames        []string `json:"hostnames,omitempty"`
}

type ResourceSpec struct {
	CPU          string `json:"cpu,omitempty"`
	Memory       string `json:"memory,omitempty"`
	GPU          string `json:"gpu,omitempty"`
	Storage      string `json:"storage,omitempty"`
	InstanceType string `json:"instanceType,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type VolumeSpec struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Capacity     string `json:"capacity,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
	MountPath    string `json:"mountPath,omitempty"`
}

type DataSourceSpec struct {
	Type      string `json:"type"`
	URI       string `json:"uri"`
	MountPath string `json:"mountPath"`
}

type HealthCheckSpec struct {
	Path         string `json:"path,omitempty"`
	Port         int32  `json:"port,omitempty"`
	InitialDelay int32  `json:"initialDelaySeconds,omitempty"`
	Period       int32  `json:"periodSeconds,omitempty"`
	Timeout      int32  `json:"timeoutSeconds,omitempty"`
}

type AutoScalingSpec struct {
	MinReplicas   int32 `json:"minReplicas,omitempty"`
	MaxReplicas   int32 `json:"maxReplicas,omitempty"`
	TargetCPUUtil int32 `json:"targetCPUUtilizationPercentage,omitempty"`
	TargetGPUUtil int32 `json:"targetGPUUtilizationPercentage,omitempty"`
}

type BaseToolSpec struct {
	Image        string        `json:"image,omitempty"`
	Version      string        `json:"version,omitempty"`
	Env          []EnvVar      `json:"env,omitempty"`
	Volumes      []VolumeMount `json:"volumes,omitempty"`
	Port         int32         `json:"port,omitempty"`
	Command      []string      `json:"command,omitempty"`
	Args         []string      `json:"args,omitempty"`
	Resources    *ResourceSpec `json:"resources,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
	Routing      *RoutingSpec  `json:"routing,omitempty"`
}

type CodeServerSpec struct {
	BaseToolSpec `json:",inline"`
	Password     *string `json:"password,omitempty"`
	Workspace    string  `json:"workspace,omitempty"`
}

type JupyterSpec struct {
	BaseToolSpec `json:",inline"`
	NotebookDir  string `json:"notebookDir,omitempty"`
}

type NoVNCSpec struct {
	BaseToolSpec `json:",inline"`
	Display      string `json:"display,omitempty"`
	Password     string `json:"password,omitempty"`
}

type DevTaskSpec struct {
	Resources   *ResourceSpec    `json:"resources,omitempty"`
	Volumes     []VolumeSpec     `json:"volumes,omitempty"`
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
	RunAfter    []string         `json:"runAfter,omitempty"`
	TTLSeconds  *int32           `json:"ttlSecondsAfterFinished,omitempty"`

	CodeServer *CodeServerSpec `json:"codeserver,omitempty"`
	Jupyter    *JupyterSpec    `json:"jupyter,omitempty"`
	NoVNC      *NoVNCSpec      `json:"novnc,omitempty"`
	VLLM       *VLLMSpec       `json:"vllm,omitempty"`
}

type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type TaskStatus struct {
	Phase       TaskPhase         `json:"phase"`
	Message     string            `json:"message,omitempty"`
	PodName     string            `json:"podName,omitempty"`
	ServiceURL  string            `json:"serviceURL,omitempty"`
	ExternalURL string            `json:"externalURL,omitempty"`
	Endpoints   []string          `json:"endpoints,omitempty"`
	Metrics     map[string]string `json:"metrics,omitempty"`
	Conditions  []Condition       `json:"conditions,omitempty"`
	StartTime   *metav1.Time      `json:"startTime,omitempty"`
	EndTime     *metav1.Time      `json:"endTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dev,categories=all
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".metadata.labels.framework"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="External",type="string",JSONPath=".status.externalURL"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type DevTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevTaskSpec `json:"spec,omitempty"`
	Status TaskStatus  `json:"status,omitempty"`
}

func (r *DevTask) GetFramework() string {
	if r.Spec.CodeServer != nil {
		return "codeserver"
	}
	if r.Spec.Jupyter != nil {
		return "jupyter"
	}
	if r.Spec.NoVNC != nil {
		return "novnc"
	}
	return ""
}

func (r *DevTask) NeedsExternalRouting() bool {
	if r.Spec.CodeServer != nil {
		return r.Spec.CodeServer.Routing == nil || r.Spec.CodeServer.Routing.Enabled
	}
	if r.Spec.Jupyter != nil {
		return r.Spec.Jupyter.Routing == nil || r.Spec.Jupyter.Routing.Enabled
	}
	if r.Spec.NoVNC != nil {
		return r.Spec.NoVNC.Routing == nil || r.Spec.NoVNC.Routing.Enabled
	}
	return false
}

func (r *DevTask) GetRoutingConfig() *RoutingSpec {
	if r.Spec.CodeServer != nil && r.Spec.CodeServer.Routing != nil {
		return r.Spec.CodeServer.Routing
	}
	if r.Spec.Jupyter != nil && r.Spec.Jupyter.Routing != nil {
		return r.Spec.Jupyter.Routing
	}
	if r.Spec.NoVNC != nil && r.Spec.NoVNC.Routing != nil {
		return r.Spec.NoVNC.Routing
	}
	if r.Spec.CodeServer != nil || r.Spec.Jupyter != nil || r.Spec.NoVNC != nil {
		return &RoutingSpec{Enabled: true}
	}
	return nil
}

func (r *DevTask) GetGatewayNamespace() string {
	cfg := r.GetRoutingConfig()
	if cfg != nil && cfg.GatewayNamespace != "" {
		return cfg.GatewayNamespace
	}
	return "default"
}

func (r *DevTask) GetRoutePath() string {
	cfg := r.GetRoutingConfig()
	if cfg != nil && cfg.PathPrefix != "" {
		return cfg.PathPrefix
	}
	return fmt.Sprintf("/%s/%s", r.GetFramework(), r.Name)
}

func (r *DevTask) GetExternalURL() string {
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
type DevTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevTask{}, &DevTaskList{})
}
