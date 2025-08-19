package events

import (
	"github.com/shuldan/framework/pkg/contracts"
	"reflect"
)

func New() contracts.Bus {
	return &bus{
		listeners: make(map[reflect.Type][]*listenerAdapter),
	}
}

func NewModule() contracts.AppModule {
	return &module{}
}
