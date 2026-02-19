# KAI Operator

KAI Operator is a Kubernetes Operator for managing AI/ML workloads, including development environments, model inference, training, fine-tuning, and evaluation tasks.

## Features

- **DevTask**: Development environment management (CodeServer, Jupyter, noVNC)
- **InferenceTask**: Model inference services (vllm, llama.cpp)
- **TrainTask**: Model training tasks (PyTorch, TensorFlow)
- **FineTuneTask**: Model fine-tuning tasks (LoRA, LlamaFactory)
- **EvalTask**: Model evaluation tasks (OpenCompass)

## Quick Start

### Prerequisites

- Go 1.24.6+
- Docker 17.03+
- kubectl v1.11.3+
- Kubernetes v1.11.3+ cluster

### Deploy to Cluster

**Build and push image:**

```sh
make docker-build docker-push IMG=<some-registry>/kai-operator:tag
```

**Install CRDs:**

```sh
make install
```

**Deploy Manager:**

```sh
make deploy IMG=<some-registry>/kai-operator:tag
```

### Create Sample Resources

```sh
# Development environment
kubectl apply -f config/samples/dev/codeserver-minimal.yaml

# Inference service
kubectl apply -f config/samples/inference/vllm-minimal.yaml

# Fine-tuning task
kubectl apply -f config/samples/finetune/llamafactory-minimal.yaml

```

### Uninstall

```sh
# Delete resources
kubectl delete -k config/samples/

# Delete CRDs
make uninstall

# Undeploy controller
make undeploy
```

## Development

### Local Run

```sh
make run
```

### Generate Code

```sh
make generate
make manifests
```

## Project Structure

```
kai-operator/
├── api/v1alpha1/          # CRD type definitions
├── internal/controller/   # Controller implementations
├── config/
│   ├── crd/              # CRD YAML files
│   ├── samples/          # Sample resources
│   ├── rbac/             # RBAC configuration
│   └── manager/          # Manager configuration
├── Dockerfile            # Build image
├── Makefile              # Build commands
└── PROJECT               # Kubebuilder project configuration
```

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
