package config

import (
	"encoding/json"
	"fmt"
)

const TrapKey = "trap"

func init() {
	allSections[TrapKey] = section{
		key:         TrapKey,
		mapToStruct: trapMapToStruct,
		validate:    validateTrap,
		defaultValue: func() any {
			return DefaultTrap()
		},
		pointerValue: func() any {
			return &Trap{}
		},
	}
}

type Trap struct {
	Config string `mapstructure:"config"`
}

func DefaultTrap() Trap {
	return Trap{
		Config: "{}",
	}
}

func trapMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Trap
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}

func validateTrap(t any) error {
	trap, err := ConvertToStruct[Trap](t)
	if err != nil {
		return err
	}

	if trap.Config != "" && !json.Valid([]byte(trap.Config)) {
		return fmt.Errorf("trap config is not valid JSON")
	}

	return nil
}
