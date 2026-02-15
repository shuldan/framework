package framework

import (
	"github.com/shuldan/config"

	"github.com/shuldan/framework/logger"
)

type KernelOption func(*kernelOptions)

type kernelOptions struct {
	configFiles   []string
	envPrefix     string
	profileEnvVar string
	logger        *logger.Logger
	config        *config.Config
}

func defaultKernelOptions() *kernelOptions {
	return &kernelOptions{
		configFiles: []string{"config.yaml"},
	}
}

func WithConfigFile(paths ...string) KernelOption {
	return func(o *kernelOptions) {
		o.configFiles = paths
	}
}

func WithEnvPrefix(prefix string) KernelOption {
	return func(o *kernelOptions) {
		o.envPrefix = prefix
	}
}

func WithProfileEnv(envVar string) KernelOption {
	return func(o *kernelOptions) {
		o.profileEnvVar = envVar
	}
}

func WithLogger(l *logger.Logger) KernelOption {
	return func(o *kernelOptions) {
		if l != nil {
			o.logger = l
		}
	}
}

func WithConfig(cfg *config.Config) KernelOption {
	return func(o *kernelOptions) {
		o.config = cfg
	}
}
