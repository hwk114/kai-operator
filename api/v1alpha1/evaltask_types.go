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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ModelEvalConfig struct {
	Model     string `json:"model"`
	ModelPath string `json:"modelPath,omitempty"`
	Template  string `json:"template,omitempty"`
	BatchSize int32  `json:"batchSize,omitempty"`
	MaxSeqLen int32  `json:"maxSeqLen,omitempty"`
}

type OpenCompassSpec struct {
	BaseToolSpec `json:",inline"`
	Models       []ModelEvalConfig `json:"models,omitempty"`
	Datasets     []string          `json:"datasets,omitempty"`
	WorkDir      string            `json:"workDir,omitempty"`
	OutputDir    string            `json:"outputDir,omitempty"`
	Debug        bool              `json:"debug,omitempty"`
}

type LmEvalSpec struct {
	BaseToolSpec `json:",inline"`
	ModelPath    string   `json:"modelPath,omitempty"`
	Tasks        []string `json:"tasks,omitempty"`
	BatchSize    int32    `json:"batchSize,omitempty"`
	NumFewShot   int32    `json:"numFewShot,omitempty"`
	OutputPath   string   `json:"outputPath,omitempty"`
}

type EvalTaskSpec struct {
	Resources   *ResourceSpec    `json:"resources,omitempty"`
	Volumes     []VolumeSpec     `json:"volumes,omitempty"`
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
	RunAfter    []string         `json:"runAfter,omitempty"`
	TTLSeconds  *int32           `json:"ttlSecondsAfterFinished,omitempty"`

	OpenCompass *OpenCompassSpec `json:"opencompass,omitempty"`
	LmEval      *LmEvalSpec      `json:"lm-eval,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=eval,categories=all
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".metadata.labels.framework"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type EvalTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EvalTaskSpec `json:"spec,omitempty"`
	Status TaskStatus   `json:"status,omitempty"`
}

func (r *EvalTask) GetFramework() string {
	if r.Spec.OpenCompass != nil {
		return "opencompass"
	}
	if r.Spec.LmEval != nil {
		return "lm-eval"
	}
	return ""
}

// +kubebuilder:object:root=true
type EvalTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EvalTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EvalTask{}, &EvalTaskList{})
}
