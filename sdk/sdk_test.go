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

package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hwk114/kai-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("SDK Client", func() {
	fmt.Println("=== SDK Client Tests ===")
	var (
		cl        *Client
		k8sClient client.Client
		scheme    *runtime.Scheme
		ctx       context.Context
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		var err error
		cl, err = NewClientFromK8s(k8sClient, WithNamespace("default"))
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
	})

	Describe("DevTask CRUD", func() {
		var devTask *v1alpha1.DevTask

		BeforeEach(func() {
			devTask = &v1alpha1.DevTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dev",
					Namespace: "default",
				},
				Spec: v1alpha1.DevTaskSpec{
					CodeServer: &v1alpha1.CodeServerSpec{
						BaseToolSpec: v1alpha1.BaseToolSpec{
							Image: "codercom/code-server:latest",
						},
					},
				},
			}
		})

		It("should create DevTask", func() {
			fmt.Println("  [TEST] Creating DevTask...")
			err := cl.DevTask.Create(ctx, devTask)
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.DevTask.Get(ctx, "test-dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Name).To(Equal("test-dev"))
			Expect(created.Spec.CodeServer.Image).To(Equal("codercom/code-server:latest"))
			fmt.Println("  [PASS] DevTask created successfully")
		})

		It("should list DevTasks", func() {
			fmt.Println("  [TEST] Listing DevTasks...")
			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())

			list, err := cl.DevTask.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list)).To(Equal(1))
			Expect(list[0].Name).To(Equal("test-dev"))
			fmt.Println("  [PASS] DevTasks listed successfully")
		})

		It("should update DevTask", func() {
			fmt.Println("  [TEST] Updating DevTask...")
			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())

			devTask.Spec.CodeServer.Image = "codercom/code-server:v2"
			Expect(cl.DevTask.Update(ctx, devTask)).To(Succeed())

			updated, err := cl.DevTask.Get(ctx, "test-dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Spec.CodeServer.Image).To(Equal("codercom/code-server:v2"))
			fmt.Println("  [PASS] DevTask updated successfully")
		})

		It("should delete DevTask", func() {
			fmt.Println("  [TEST] Deleting DevTask...")
			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())
			Expect(cl.DevTask.Delete(ctx, "test-dev")).To(Succeed())

			exists, err := cl.DevTask.Exists(ctx, "test-dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
			fmt.Println("  [PASS] DevTask deleted successfully")
		})

		It("should check existence", func() {
			fmt.Println("  [TEST] Checking DevTask existence...")
			exists, err := cl.DevTask.Exists(ctx, "test-dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())

			exists, err = cl.DevTask.Exists(ctx, "test-dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			fmt.Println("  [PASS] DevTask existence check passed")
		})
	})

	Describe("InferenceTask CRUD", func() {
		var inferenceTask *v1alpha1.InferenceTask

		BeforeEach(func() {
			fmt.Println("\n=== InferenceTask CRUD Tests ===")
			inferenceTask = &v1alpha1.InferenceTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-inference",
					Namespace: "default",
				},
				Spec: v1alpha1.InferenceTaskSpec{
					VLLM: &v1alpha1.VLLMSpec{
						BaseToolSpec: v1alpha1.BaseToolSpec{
							Image: "vllm/vllm:latest",
						},
						ModelPath: "/models/llama",
						ModelName: "llama",
					},
				},
			}
		})

		It("should create InferenceTask", func() {
			err := cl.InferenceTask.Create(ctx, inferenceTask)
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.InferenceTask.Get(ctx, "test-inference")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.VLLM.ModelName).To(Equal("llama"))
		})

		It("should get endpoint", func() {
			Expect(cl.InferenceTask.Create(ctx, inferenceTask)).To(Succeed())

			endpoint, err := cl.InferenceTask.GetEndpoint(ctx, "test-inference")
			Expect(err).NotTo(HaveOccurred())
			Expect(endpoint).To(BeEmpty())
		})

		It("should delete InferenceTask", func() {
			Expect(cl.InferenceTask.Create(ctx, inferenceTask)).To(Succeed())
			Expect(cl.InferenceTask.Delete(ctx, "test-inference")).To(Succeed())

			exists, err := cl.InferenceTask.Exists(ctx, "test-inference")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("TrainTask CRUD", func() {
		var trainTask *v1alpha1.TrainTask

		BeforeEach(func() {
			fmt.Println("\n=== TrainTask CRUD Tests ===")
			time.Sleep(10 * time.Millisecond)
			trainTask = &v1alpha1.TrainTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-train",
					Namespace: "default",
				},
				Spec: v1alpha1.TrainTaskSpec{
					PyTorch: &v1alpha1.PyTorchSpec{
						BaseToolSpec: v1alpha1.BaseToolSpec{
							Image: "pytorch/pytorch:latest",
						},
						ScriptPath: "/train.py",
					},
				},
			}
		})

		It("should create TrainTask", func() {
			err := cl.TrainTask.Create(ctx, trainTask)
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.TrainTask.Get(ctx, "test-train")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.PyTorch.ScriptPath).To(Equal("/train.py"))
		})

		It("should get phase", func() {
			Expect(cl.TrainTask.Create(ctx, trainTask)).To(Succeed())

			phase, err := cl.TrainTask.GetPhase(ctx, "test-train")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(BeEmpty())
		})

		It("should check is running", func() {
			Expect(cl.TrainTask.Create(ctx, trainTask)).To(Succeed())

			isRunning, err := cl.TrainTask.IsRunning(ctx, "test-train")
			Expect(err).NotTo(HaveOccurred())
			Expect(isRunning).To(BeFalse())
		})
	})

	Describe("FineTuneTask CRUD", func() {
		var fineTuneTask *v1alpha1.FineTuneTask

		BeforeEach(func() {
			fineTuneTask = &v1alpha1.FineTuneTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-finetune",
					Namespace: "default",
				},
				Spec: v1alpha1.FineTuneTaskSpec{
					LoRA: &v1alpha1.LoRASpec{
						BaseToolSpec: v1alpha1.BaseToolSpec{
							Image: "ghcr.io/huggingface/peft:latest",
						},
						ModelPath: "/models/llama",
					},
				},
			}
		})

		It("should create FineTuneTask", func() {
			err := cl.FineTuneTask.Create(ctx, fineTuneTask)
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.FineTuneTask.Get(ctx, "test-finetune")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.LoRA.ModelPath).To(Equal("/models/llama"))
		})
	})

	Describe("EvalTask CRUD", func() {
		var evalTask *v1alpha1.EvalTask

		BeforeEach(func() {
			evalTask = &v1alpha1.EvalTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eval",
					Namespace: "default",
				},
				Spec: v1alpha1.EvalTaskSpec{
					OpenCompass: &v1alpha1.OpenCompassSpec{
						BaseToolSpec: v1alpha1.BaseToolSpec{
							Image: "opencompass/opencompass:latest",
						},
					},
				},
			}
		})

		It("should create EvalTask", func() {
			err := cl.EvalTask.Create(ctx, evalTask)
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.EvalTask.Get(ctx, "test-eval")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.OpenCompass.Image).To(Equal("opencompass/opencompass:latest"))
		})
	})

	Describe("WaitOptions", func() {
		It("should parse default options", func() {
			opts := parseWaitOptions()
			Expect(opts.Interval).To(Equal(2 * time.Second))
			Expect(opts.Timeout).To(Equal(10 * time.Minute))
		})

		It("should parse custom options", func() {
			opts := parseWaitOptions(
				WithInterval(5*time.Second),
				WithTimeout(5*time.Minute),
			)
			Expect(opts.Interval).To(Equal(5 * time.Second))
			Expect(opts.Timeout).To(Equal(5 * time.Minute))
		})
	})

	Describe("Builder", func() {
		It("should create DevTask with builder", func() {
			task := NewDevTask("builder-dev", "default").
				WithCodeServer("codercom/code-server:latest").
				WithResources("2", "4Gi", "1").
				WithLabels(map[string]string{"app": "test"}).
				Build()

			Expect(task.Name).To(Equal("builder-dev"))
			Expect(task.Namespace).To(Equal("default"))
			Expect(task.Spec.CodeServer.Image).To(Equal("codercom/code-server:latest"))
			Expect(task.Spec.Resources.CPU).To(Equal("2"))
			Expect(task.Labels["app"]).To(Equal("test"))
		})

		It("should create InferenceTask with builder", func() {
			task := NewInferenceTask("builder-inference", "default").
				WithVLLM("llama-7b").
				WithResources("4", "16Gi", "2").
				WithGPUCount(2).
				Build()

			Expect(task.Name).To(Equal("builder-inference"))
			Expect(task.Spec.Engine).To(Equal("vllm"))
			Expect(task.Spec.VLLM.TensorParallelSize).To(Equal(int32(2)))
		})

		It("should create TrainTask with builder", func() {
			task := NewTrainTask("builder-train", "default").
				WithPyTorch("/train.py").
				WithBatchSize(32).
				WithMaxEpochs(10).
				Build()

			Expect(task.Name).To(Equal("builder-train"))
			Expect(task.Spec.PyTorch.ScriptPath).To(Equal("/train.py"))
			Expect(task.Spec.PyTorch.BatchSize).To(Equal(int32(32)))
		})

		It("should create FineTuneTask with builder", func() {
			task := NewFineTuneTask("builder-finetune", "default").
				WithLoRA("/models/llama").
				WithTrainData("/data/train.json").
				WithEpochs(3).
				Build()

			Expect(task.Name).To(Equal("builder-finetune"))
			Expect(task.Spec.LoRA.TrainData).To(Equal("/data/train.json"))
			Expect(task.Spec.LoRA.Epochs).To(Equal(int32(3)))
		})

		It("should create EvalTask with builder", func() {
			task := NewEvalTask("builder-eval", "default").
				WithOpenCompass().
				WithModel("llama-7b", "/models/llama").
				WithDatasets("mmlu", "arc").
				Build()

			Expect(task.Name).To(Equal("builder-eval"))
			Expect(len(task.Spec.OpenCompass.Datasets)).To(Equal(2))
		})
	})

	Describe("CreateOrPatch", func() {
		It("should create new resource if not exists", func() {
			task := NewDevTask("patch-test", "default").
				WithCodeServer("codercom/code-server:latest").
				Build()

			err := cl.DevTask.CreateOrPatch(ctx, task, func(t *v1alpha1.DevTask) {
				t.Spec.CodeServer.Image = "codercom/code-server:v2"
			})
			Expect(err).NotTo(HaveOccurred())

			created, err := cl.DevTask.Get(ctx, "patch-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.CodeServer.Image).To(Equal("codercom/code-server:v2"))
		})
	})

	Describe("GetTask/DeleteTask", func() {
		It("should get task by type", func() {
			devTask := NewDevTask("type-test", "default").
				WithCodeServer("codercom/code-server:latest").
				Build()
			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())

			task, err := cl.GetTask(ctx, TaskTypeDev, "type-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(task.(*v1alpha1.DevTask).Name).To(Equal("type-test"))
		})

		It("should delete task by type", func() {
			devTask := NewDevTask("delete-test", "default").
				WithCodeServer("codercom/code-server:latest").
				Build()
			Expect(cl.DevTask.Create(ctx, devTask)).To(Succeed())

			err := cl.DeleteTask(ctx, TaskTypeDev, "delete-test")
			Expect(err).NotTo(HaveOccurred())

			exists, _ := cl.DevTask.Exists(ctx, "delete-test")
			Expect(exists).To(BeFalse())
		})
	})
})

func TestSDK(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SDK Suite")
}
