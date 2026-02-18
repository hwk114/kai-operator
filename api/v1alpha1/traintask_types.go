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

type PyTorchSpec struct {
	BaseToolSpec       `json:",inline"`
	ScriptPath         string            `json:"scriptPath,omitempty"`
	Hyperparameters    map[string]string `json:"hyperparameters,omitempty"`
	CheckpointDir      string            `json:"checkpointDir,omitempty"`
	OutputDir          string            `json:"outputDir,omitempty"`
	MaxEpochs          int32             `json:"maxEpochs,omitempty"`
	BatchSize          int32             `json:"batchSize,omitempty"`
	LearningRate       string            `json:"learningRate,omitempty"`
	GradientAccumStep  int32             `json:"gradientAccumulationSteps,omitempty"`
	WarmupSteps        int32             `json:"warmupSteps,omitempty"`
	SaveInterval       int32             `json:"saveInterval,omitempty"`
	EvalInterval       int32             `json:"evalInterval,omitempty"`
	WorldSize          int32             `json:"worldSize,omitempty"`
	NprocPerNode       int32             `json:"nprocPerNode,omitempty"`
	DistributedBackend string            `json:"distributedBackend,omitempty"`
}

type TensorFlowSpec struct {
	BaseToolSpec         `json:",inline"`
	ScriptPath           string            `json:"scriptPath,omitempty"`
	Hyperparameters      map[string]string `json:"hyperparameters,omitempty"`
	CheckpointDir        string            `json:"checkpointDir,omitempty"`
	OutputDir            string            `json:"outputDir,omitempty"`
	MaxSteps             int32             `json:"maxSteps,omitempty"`
	BatchSize            int32             `json:"batchSize,omitempty"`
	LearningRate         string            `json:"learningRate,omitempty"`
	SaveInterval         int32             `json:"saveInterval,omitempty"`
	EvalInterval         int32             `json:"evalInterval,omitempty"`
	DistributionStrategy string            `json:"distributionStrategy,omitempty"`
}

type TrainTaskSpec struct {
	Resources   *ResourceSpec    `json:"resources,omitempty"`
	Volumes     []VolumeSpec     `json:"volumes,omitempty"`
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`
	RunAfter    []string         `json:"runAfter,omitempty"`
	TTLSeconds  *int32           `json:"ttlSecondsAfterFinished,omitempty"`

	PyTorch    *PyTorchSpec    `json:"pytorch,omitempty"`
	TensorFlow *TensorFlowSpec `json:"tensorflow,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=train,categories=all
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".metadata.labels.framework"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type TrainTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrainTaskSpec `json:"spec,omitempty"`
	Status TaskStatus    `json:"status,omitempty"`
}

func (r *TrainTask) GetFramework() string {
	if r.Spec.PyTorch != nil {
		return "pytorch"
	}
	if r.Spec.TensorFlow != nil {
		return "tensorflow"
	}
	return ""
}

// +kubebuilder:object:root=true
type TrainTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrainTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrainTask{}, &TrainTaskList{})
}
