package common

import (
	"maps"
	"strings"
)

type (
	VariableMap   map[string]string
	VariableSlice []string
)

// Converts a VariableMap to a VariableSlice
func VariablesMapToSlice(variables VariableMap) VariableSlice {
	var ret VariableSlice = make(VariableSlice, 0)
	for key, val := range variables {
		ret = append(ret, string(key+"="+val))
	}
	return ret
}

// Converts a VariableSlice to a VariableMap
func VariablesSliceToMap(variables VariableSlice) VariableMap {
	var ret VariableMap = make(VariableMap)
	for _, val := range variables {
		keyValSplit := strings.Split(val, "=")
		ret[strings.TrimSpace(keyValSplit[0])] = strings.TrimSpace(keyValSplit[1])
	}

	return ret
}

func MergeVariables(vars ...VariableMap) VariableMap {
	var ret VariableMap = make(VariableMap)
	for _, v := range vars {
		maps.Copy(ret, v)
	}
	return ret
}
