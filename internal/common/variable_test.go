package common_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/ConnorShore/micro-ci/internal/common"
)

func TestVariableMapToSlice(t *testing.T) {
	expected := common.VariableSlice([]string{"key1=val1", "key2=val2", "key3=val3", "key4=val4"})
	input := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	})

	actual := common.VariablesMapToSlice(input)
	if eq := slices.Equal(actual, expected); !eq {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestVariablesSliceToMap(t *testing.T) {
	expected := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	})
	input := common.VariableSlice([]string{"key1=val1", "key2=val2", "key3=val3", "key4=val4"})

	actual := common.VariablesSliceToMap(input)
	if eq := maps.Equal(actual, expected); !eq {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestMergeVariablesWithEmptyMap(t *testing.T) {
	expected := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	})
	input1 := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	})
	input2 := make(common.VariableMap)

	actual := common.MergeVariables(input1, input2)

	if eq := maps.Equal(actual, expected); !eq {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestMergeVariables2Maps(t *testing.T) {
	expected := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	})
	input1 := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
	})
	input2 := common.VariableMap(map[string]string{
		"key3": "val3",
		"key4": "val4",
	})

	actual := common.MergeVariables(input1, input2)

	if eq := maps.Equal(actual, expected); !eq {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}

func TestMergeVariables3Maps(t *testing.T) {
	expected := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
		"key5": "val5",
	})
	input1 := common.VariableMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
	})
	input2 := common.VariableMap(map[string]string{
		"key3": "val3",
		"key4": "val4",
	})
	input3 := common.VariableMap(map[string]string{
		"key5": "val5",
	})

	actual := common.MergeVariables(input1, input2, input3)

	if eq := maps.Equal(actual, expected); !eq {
		t.Errorf("Expected %v but got %v\n", expected, actual)
	}
}
