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

type LoRASpec struct {
	BaseToolSpec  `json:",inline"`
	ModelPath     string   `json:"modelPath,omitempty"`
	TrainData     string   `json:"trainData,omitempty"`
	ValidData     string   `json:"validData,omitempty"`
	OutputDir     string   `json:"outputDir,omitempty"`
	LoraRank      int32    `json:"loraRank,omitempty"`
	LoraAlpha     int32    `json:"loraAlpha,omitempty"`
	LoraDropout   string   `json:"loraDropout,omitempty"`
	TargetModules []string `json:"targetModules,omitempty"`
	BatchSize     int32    `json:"batchSize,omitempty"`
	LearningRate  string   `json:"learningRate,omitempty"`
	Epochs        int32    `json:"epochs,omitempty"`
	CutoffLen     int32    `json:"cutoffLen,omitempty"`
	WarmupSteps   int32    `json:"warmupSteps,omitempty"`
	SaveSteps     int32    `json:"saveSteps,omitempty"`
	EvalSteps     int32    `json:"evalSteps,omitempty"`
}

type LlamaFactorySpec struct {
	BaseToolSpec   `json:",inline"`
	ModelPath      string   `json:"modelPath,omitempty"`
	Stage          string   `json:"stage,omitempty"`
	TrainData      string   `json:"trainData,omitempty"`
	ValidData      string   `json:"validData,omitempty"`
	OutputDir      string   `json:"outputDir,omitempty"`
	FinetuningType string   `json:"finetuningType,omitempty"`
	LoraRank       int32    `json:"loraRank,omitempty"`
	LoraAlpha      int32    `json:"loraAlpha,omitempty"`
	LoraDropout    string   `json:"loraDropout,omitempty"`
	TargetModules  []string `json:"targetModules,omitempty"`
	BatchSize      int32    `json:"batchSize,omitempty"`
	LearningRate   string   `json:"learningRate,omitempty"`
	NumEpochs      int32    `json:"numEpochs,omitempty"`
	MaxGradNorm    string   `json:"maxGradNorm,omitempty"`
	WarmupRatio    string   `json:"warmupRatio,omitempty"`
	SaveSteps      int32    `json:"saveSteps,omitempty"`
	EvalSteps      int32    `json:"evalSteps,omitempty"`
}

type FineTuneTaskSpec struct {
	Resources   *ResourceSpec    `json:"resources,omitempty"`
	Volumes     []VolumeSpec     `json:"volumes,omitempty"`
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
	RunAfter    []string         `json:"runAfter,omitempty"`
	TTLSeconds  *int32           `json:"ttlSecondsAfterFinished,omitempty"`

	LoRA         *LoRASpec         `json:"lora,omitempty"`
	LlamaFactory *LlamaFactorySpec `json:"llamafactory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=finetune,categories=all
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".metadata.labels.framework"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type FineTuneTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FineTuneTaskSpec `json:"spec,omitempty"`
	Status TaskStatus       `json:"status,omitempty"`
}

func (r *FineTuneTask) GetFramework() string {
	if r.Spec.LoRA != nil {
		return "lora"
	}
	if r.Spec.LlamaFactory != nil {
		return "llamafactory"
	}
	return ""
}

// +kubebuilder:object:root=true
type FineTuneTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FineTuneTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FineTuneTask{}, &FineTuneTaskList{})
}
