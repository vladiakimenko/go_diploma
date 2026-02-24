package sentinel

import "fmt"

/*
Package sentinel provides a type for marking signal values or states.

Usage:
	signal := sentinel.New("MySignal")

	if signal.Is(other) {
		fmt.Println("The value of other is signal")
	}
*/

type Sentinel struct {
	name string
}

// Init
func New(name string) Sentinel {
	return Sentinel{name: name}
}

// Readonly name
func (s Sentinel) Name() string {
	return s.name
}

// Printing
func (s Sentinel) String() string {
	return s.name
}

// Error logging
func (s Sentinel) Error(context string) error {
	return fmt.Errorf("%s: sentinel triggered (%s)", context, s.name)
}

// Value comparison
func (s Sentinel) Is(value any) bool {
	if v, ok := value.(Sentinel); ok {
		return v == s
	}
	return false
}
