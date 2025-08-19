package application

import "github.com/shuldan/framework/pkg/errors"

var newAppCode = errors.WithPrefix("APP")
var newRegistryCode = errors.WithPrefix("APP_REGISTRY")
var newContainerCode = errors.WithPrefix("APP_CONTAINER")

var (
	ErrModuleRegister = newAppCode().New("failed to register module {{.module}}")
	ErrModuleStart    = newAppCode().New("failed to start module {{.module}}")
	ErrAppRun         = newAppCode().New("application run failed with reason: {{.reason}}")
	ErrAppStop        = newAppCode().New("application stop failed with reason: {{.reason}}")

	ErrModuleStop = newRegistryCode().New("failed to stop module {{.module}}")

	ErrCircularDep       = newContainerCode().New("circular dependency detected for key {{.name}}")
	ErrValueNotFound     = newContainerCode().New("value not found for key {{.name}}")
	ErrDuplicateInstance = newContainerCode().New("instance already exists for key {{.name}}")
	ErrDuplicateFactory  = newContainerCode().New("factory already registered for key {{.name}}")
)
