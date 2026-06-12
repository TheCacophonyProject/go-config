package config

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const TrapKey = "trap"

func init() {
	allSections[TrapKey] = section{
		key:         TrapKey,
		mapToStruct: trapMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() any {
			return DefaultTrap()
		},
		pointerValue: func() any {
			return &Trap{}
		},
	}
}

type Trap struct {
	Config map[string]any `mapstructure:"config"`
}

func DefaultTrap() Trap {
	return Trap{
		Config: map[string]any{},
	}
}

func trapMapToStruct(m map[string]any) (any, error) {
	var s Trap
	if err := decodeStructFromMap(&s, m, jsonStringToMap); err != nil {
		return nil, err
	}
	return s, nil
}

func jsonStringToMap(f reflect.Type, t reflect.Type, data any) (any, error) {
	if f.Kind() != reflect.String || t != reflect.TypeFor[map[string]any]() {
		return data, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data.(string)), &m); err != nil {
		return nil, fmt.Errorf("failed to parse JSON string as map: %v", err)
	}
	return m, nil
}
