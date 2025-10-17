package app

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

type testStringType string
type testStructType struct {
	Value string
}
type testInterfaceType interface {
	GetValue() string
}
type testImplementation struct {
	Value string
}

func (t *testImplementation) GetValue() string {
	return t.Value
}

func TestContainer_ResolveInstance(t *testing.T) {
	c := NewContainer()

	testStr := testStringType("hello")
	strType := reflect.TypeOf((*testStringType)(nil)).Elem()
	if err := c.Instance(strType, testStr); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	val, err := c.Resolve(strType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(testStringType) != testStr {
		t.Errorf("Expected 'hello', got %v", val)
	}
}

func TestContainer_ResolveFactory(t *testing.T) {
	c := NewContainer()

	strType := reflect.TypeOf((*string)(nil)).Elem()
	if err := c.Factory(strType, func(c contracts.DIContainer) (interface{}, error) {
		result := "Hello from factory!"
		return result, nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}

	val, err := c.Resolve(strType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(string) != "Hello from factory!" {
		t.Errorf("Expected factory result, got %v", val)
	}
}

func TestContainer_StructInstance(t *testing.T) {
	c := NewContainer()

	testStruct := &testStructType{Value: "test"}
	structType := reflect.TypeOf((*testStructType)(nil))
	if err := c.Instance(structType, testStruct); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	val, err := c.Resolve(structType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(*testStructType).Value != "test" {
		t.Errorf("Expected 'test', got %v", val)
	}
}

func TestContainer_InterfaceFactory(t *testing.T) {
	c := NewContainer()

	interfaceType := reflect.TypeOf((*testInterfaceType)(nil)).Elem()
	impl := &testImplementation{Value: "implementation"}

	if err := c.Instance(interfaceType, impl); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	val, err := c.Resolve(interfaceType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(testInterfaceType).GetValue() != "implementation" {
		t.Errorf("Expected 'implementation', got %v", val.(testInterfaceType).GetValue())
	}
}

func TestContainer_FactoryWithDependencies(t *testing.T) {
	c := NewContainer()

	depType := reflect.TypeOf((*string)(nil)).Elem()
	if err := c.Instance(depType, "dependency"); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	resultType := reflect.TypeOf((*testStructType)(nil))
	if err := c.Factory(resultType, func(c contracts.DIContainer) (interface{}, error) {
		dep, err := c.Resolve(depType)
		if err != nil {
			return nil, err
		}
		return &testStructType{Value: dep.(string)}, nil
	}); err != nil {
		t.Fatalf("Factory failed: %v", err)
	}

	val, err := c.Resolve(resultType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(*testStructType).Value != "dependency" {
		t.Errorf("Expected 'dependency', got %v", val.(*testStructType).Value)
	}
}

func TestContainer_CircularDependency(t *testing.T) {
	c := NewContainer()

	typeAType := reflect.TypeOf((*string)(nil)).Elem()
	typeBType := reflect.TypeOf((*int)(nil)).Elem()

	if err := c.Factory(typeAType, func(c contracts.DIContainer) (interface{}, error) {
		_, err := c.Resolve(typeBType)
		if err != nil {
			return nil, err
		}
		result := "a"
		return result, nil
	}); err != nil {
		t.Fatalf("Factory failed: %v", err)
	}

	if err := c.Factory(typeBType, func(c contracts.DIContainer) (interface{}, error) {
		_, err := c.Resolve(typeAType)
		if err != nil {
			return nil, err
		}
		result := 1
		return result, nil
	}); err != nil {
		t.Fatalf("Factory failed: %v", err)
	}

	_, err := c.Resolve(typeAType)
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}

	if !errors.Is(err, ErrCircularDep) {
		t.Errorf("Expected ErrCircularDep, got %v", err)
	}
}

func TestContainer_Has(t *testing.T) {
	c := NewContainer()

	strType := reflect.TypeOf((*string)(nil)).Elem()
	intType := reflect.TypeOf((*int)(nil)).Elem()
	structType := reflect.TypeOf((*testStructType)(nil))
	interfaceType := reflect.TypeOf((*testInterfaceType)(nil)).Elem()

	if c.Has(strType) {
		t.Error("Expected false for nonexistent key")
	}

	if err := c.Instance(strType, "value"); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}
	if !c.Has(strType) {
		t.Error("Expected true for existing instance")
	}

	if err := c.Factory(intType, func(contracts.DIContainer) (interface{}, error) {
		return 42, nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}
	if !c.Has(intType) {
		t.Error("Expected true for existing factory")
	}

	if err := c.Instance(structType, &testStructType{Value: "struct"}); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}
	if !c.Has(structType) {
		t.Error("Expected true for existing struct instance")
	}

	if err := c.Instance(interfaceType, &testImplementation{Value: "interface"}); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}
	if !c.Has(interfaceType) {
		t.Error("Expected true for existing interface instance")
	}
}

func TestContainer_DuplicateInstance(t *testing.T) {
	c := NewContainer()

	strType := reflect.TypeOf((*string)(nil)).Elem()

	err := c.Instance(strType, "value1")
	if err != nil {
		t.Fatalf("First Instance() failed: %v", err)
	}

	err = c.Instance(strType, "value2")
	if err == nil {
		t.Error("Expected error for duplicate instance")
	}

	if !errors.Is(err, ErrDuplicateInstance) {
		t.Errorf("Expected ErrDuplicateInstance, got %v", err)
	}

	val, resolveErr := c.Resolve(strType)
	if resolveErr != nil {
		t.Fatalf("Resolve failed: %v", resolveErr)
	}

	if val.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}
}

func TestContainer_DuplicateFactory(t *testing.T) {
	c := NewContainer()

	intType := reflect.TypeOf((*int)(nil)).Elem()

	err := c.Factory(intType, func(contracts.DIContainer) (interface{}, error) {
		return 1, nil
	})
	if err != nil {
		t.Fatalf("First Factory() failed: %v", err)
	}

	err = c.Factory(intType, func(contracts.DIContainer) (interface{}, error) {
		return 2, nil
	})
	if err == nil {
		t.Error("Expected error for duplicate factory")
	}

	if !errors.Is(err, ErrDuplicateFactory) {
		t.Errorf("Expected ErrDuplicateFactory, got %v", err)
	}

	val, resolveErr := c.Resolve(intType)
	if resolveErr != nil {
		t.Fatalf("Resolve failed: %v", resolveErr)
	}

	if val.(int) != 1 {
		t.Errorf("Expected 1, got %v", val)
	}
}

func TestContainer_ResolveNonExistent(t *testing.T) {
	c := NewContainer()

	strType := reflect.TypeOf((*string)(nil)).Elem()

	_, err := c.Resolve(strType)
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}

	if !errors.Is(err, ErrValueNotFound) {
		t.Errorf("Expected ErrValueNotFound, got %v", err)
	}
}

func TestContainer_FactoryErrorPropagation(t *testing.T) {
	c := NewContainer()

	strType := reflect.TypeOf((*string)(nil)).Elem()

	testErr := errors.New("factory error")
	if err := c.Factory(strType, func(contracts.DIContainer) (interface{}, error) {
		return nil, testErr
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}

	_, err := c.Resolve(strType)
	if err == nil {
		t.Fatal("Expected error from factory")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("Expected original error, got %v", err)
	}
}

func TestContainer_ConcurrentAccess(t *testing.T) {
	c := NewContainer()

	intType := reflect.TypeOf((*int)(nil)).Elem()

	if err := c.Factory(intType, func(contracts.DIContainer) (interface{}, error) {
		return 1, nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Resolve(intType)
			if err != nil {
				t.Errorf("Concurrent Resolve failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestContainer_InterfaceImplementation(t *testing.T) {
	c := NewContainer()

	interfaceType := reflect.TypeOf((*testInterfaceType)(nil)).Elem()
	implementation := &testImplementation{Value: "test impl"}

	if err := c.Instance(interfaceType, implementation); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	resolved, err := c.Resolve(interfaceType)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolvedInterface, ok := resolved.(testInterfaceType); ok {
		if resolvedInterface.GetValue() != "test impl" {
			t.Errorf("Expected 'test impl', got %v", resolvedInterface.GetValue())
		}
	} else {
		t.Errorf("Resolved value is not of expected interface type")
	}
}

func TestContainer_StructPointerVsValue(t *testing.T) {
	c := NewContainer()

	structPtrType := reflect.TypeOf((*testStructType)(nil))
	structValueType := reflect.TypeOf(testStructType{})

	structInstance := &testStructType{Value: "pointer"}
	valueInstance := testStructType{Value: "value"}

	if err := c.Instance(structPtrType, structInstance); err != nil {
		t.Fatalf("Instance failed for pointer: %v", err)
	}

	if err := c.Instance(structValueType, valueInstance); err != nil {
		t.Fatalf("Instance failed for value: %v", err)
	}

	ptrResult, err := c.Resolve(structPtrType)
	if err != nil {
		t.Fatalf("Resolve pointer failed: %v", err)
	}
	if ptrResult.(*testStructType).Value != "pointer" {
		t.Errorf("Expected 'pointer', got %v", ptrResult.(*testStructType).Value)
	}

	valResult, err := c.Resolve(structValueType)
	if err != nil {
		t.Fatalf("Resolve value failed: %v", err)
	}
	if valResult.(testStructType).Value != "value" {
		t.Errorf("Expected 'value', got %v", valResult.(testStructType).Value)
	}
}
