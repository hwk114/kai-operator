# KAI Operator

KAI Operator 是一个 Kubernetes Operator，用于管理 AI/ML 工作负载，包括开发环境、模型推理、训练、微调和评估任务。

## 功能特性

- **DevTask**: 开发环境管理 (CodeServer, Jupyter, noVNC)
- **InferenceTask**: 模型推理服务 (vllm, llama.cpp)
- **TrainTask**: 模型训练任务 (PyTorch, TensorFlow)
- **FineTuneTask**: 模型微调任务 (LoRA, LlamaFactory)
- **EvalTask**: 模型评估任务 (OpenCompass)

## 快速开始

### 前置条件

- Go 1.24.6+
- Docker 17.03+
- kubectl v1.11.3+
- Kubernetes v1.11.3+ 集群

### 部署到集群

**构建并推送镜像:**

```sh
make docker-build docker-push IMG=<some-registry>/kai-operator:tag
```

**安装 CRDs:**

```sh
make install
```

**部署 Manager:**

```sh
make deploy IMG=<some-registry>/kai-operator:tag
```

### 创建示例资源

```sh
# 开发环境
kubectl apply -f config/samples/dev/codeserver-minimal.yaml

# 推理服务
kubectl apply -f config/samples/inference/vllm-minimal.yaml

# 微调任务
kubectl apply -f config/samples/finetune/llamafactory-minimal.yaml

```

### 卸载

```sh
# 删除资源
kubectl delete -k config/samples/

# 删除 CRDs
make uninstall

# 卸载控制器
make undeploy
```

## 开发指南

### 本地运行

```sh
make run
```

### 生成代码

```sh
make generate
make manifests
```

## 项目结构

```
kai-operator/
├── api/v1alpha1/          # CRD 类型定义
├── internal/controller/   # 控制器实现
├── config/
│   ├── crd/              # CRD YAML 文件
│   ├── samples/          # 示例资源
│   ├── rbac/             # RBAC 配置
│   └── manager/          # Manager 配置
├── Dockerfile            # 构建镜像
├── Makefile              # 构建命令
└── PROJECT               # Kubebuilder 项目配置
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
