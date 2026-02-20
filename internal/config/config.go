package config

import (
	"os"
	"strconv"
	"sync"
)

type FrameworkConfig struct {
	DefaultImage string
	DefaultPort  int32
}

type Config struct {
	Frameworks map[string]FrameworkConfig
}

var (
	cfg  *Config
	once sync.Once
)

func getDefaultConfig() *Config {
	return &Config{
		Frameworks: map[string]FrameworkConfig{
			"codeserver": {
				DefaultImage: "ghcr.io/coder/code-server:latest",
				DefaultPort:  8443,
			},
			"jupyter": {
				DefaultImage: "jupyter/base-notebook:latest",
				DefaultPort:  8888,
			},
			"novnc": {
				DefaultImage: "novnc/no-vnc:latest",
				DefaultPort:  6081,
			},
			"vllm": {
				DefaultImage: "vllm/vllm-openai:nightly-aarch64",
				DefaultPort:  8000,
			},
			"tgi": {
				DefaultImage: "ghcr.io/huggingface/text-generation-inference:latest",
				DefaultPort:  8080,
			},
			"llama.cpp": {
				DefaultImage: "ghcr.io/ggml-org/llama.cpp:server",
				DefaultPort:  8080,
			},
		},
	}
}

func getEnvConfig() *Config {
	cfg := getDefaultConfig()

	for framework := range cfg.Frameworks {
		imageEnv := "KAI_DEFAULT_" + toUpperSnakeCase(framework) + "_IMAGE"
		if img := os.Getenv(imageEnv); img != "" {
			f := cfg.Frameworks[framework]
			f.DefaultImage = img
			cfg.Frameworks[framework] = f
		}

		portEnv := "KAI_DEFAULT_" + toUpperSnakeCase(framework) + "_PORT"
		if port := os.Getenv(portEnv); port != "" {
			if p, err := strconv.Atoi(port); err == nil {
				f := cfg.Frameworks[framework]
				f.DefaultPort = int32(p)
				cfg.Frameworks[framework] = f
			}
		}
	}

	return cfg
}

func toUpperSnakeCase(s string) string {
	result := make([]byte, 0, len(s)*2)
	for i, b := range []byte(s) {
		if b == '-' || b == '.' {
			result = append(result, '_')
		} else if b >= 'A' && b <= 'Z' && i > 0 {
			result = append(result, '_', b)
		} else if b >= 'a' && b <= 'z' {
			result = append(result, b-('a'-'A'))
		}
	}
	return string(result)
}

func GetConfig() *Config {
	once.Do(func() {
		cfg = getEnvConfig()
	})
	return cfg
}

func GetFrameworkImage(framework string) string {
	if f, ok := GetConfig().Frameworks[framework]; ok {
		return f.DefaultImage
	}
	return ""
}

func GetFrameworkPort(framework string) int32 {
	if f, ok := GetConfig().Frameworks[framework]; ok {
		return f.DefaultPort
	}
	return 0
}
