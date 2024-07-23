package types

import (
	"fmt"

	"sigs.k8s.io/yaml"
)

// DefaultParams returns default distribution parameters
func DefaultParams() Params {
	return Params{
		EnableDao: true,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ValidateBasic performs basic validation on distribution parameters.
func (p Params) ValidateBasic() error {
	return validateBool(p.EnableDao)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
