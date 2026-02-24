package settings

import (
	"fmt"
	"os"
	"strconv"

	"blog-api/pkg/sentinel"
)

/*
Package settings provides tools to define configuration structs whose values are loaded from environment.

Usage:
Implement Setup on a concrete config to declare env-backed fields

	type MyCustomConfig struct {
		MyOptinalIntSetting       int
		MyOptionalBoolSetting     bool
		MyCompulsoryStringSetting string
	}

	func (c *MyCustomConfig) Setup() []settings.EnvLoadable {
		return []settings.EnvLoadable{
			settings.Item[int]{Name: "MY_OPTIONAL_INT_SETTING", Default: 42, Field: &c.MyOptinalIntSetting},
			settings.Item[bool]{Name: "MY_OPTIONAL_BOOL_SETTING", Default: true, Field: &c.MyOptionalBoolSetting},
			settings.Item[string]{Name: "MY_COMPULSORY_STRING_SETTING", Default: settings.NoDefault, Field: &c.MyCompulsoryStringSetting},
		}
	}

Call LoadConfig on its instance:
	settings.LoadConfig(&JWTManagerConfig{})
*/

// SupportedConfigType limits Item to types supported by getEnv.
type SupportedConfigType interface {
	~string | ~int | ~bool
}

// EnvLoadable interface represents a single env-backed field.
type EnvLoadable interface {
	load()
}

// EnvConfigurable interface represents a config struct that declares a method returning the Items.
type EnvConfigurable interface {
	Setup() []EnvLoadable
}

// Item represents a single env variable and its target field.
type Item[T SupportedConfigType] struct {
	Name    string
	Default any // T || NoDefault
	Field   *T
}

func (c Item[T]) load() {
	*c.Field = getEnv[T](c.Name, c.Default)
}

// A function thal loads up the values from env for a single config
func LoadConfig(cfg EnvConfigurable) {
	for _, item := range cfg.Setup() {
		item.load()
	}
}

// A sentinel value to mark a mandatory setting
var NoDefault = sentinel.New("NoDefault")

func getEnv[T SupportedConfigType](key string, defaultValue any) T {
	if value := os.Getenv(key); value != "" {
		switch any(*new(T)).(type) {
		case string:
			return any(value).(T)
		case int:
			v, err := strconv.Atoi(value)
			if err != nil {
				if !NoDefault.Is(defaultValue) {
					return defaultValue.(T)
				}
				panic(err)
			}
			return any(v).(T)
		case bool:
			v, err := strconv.ParseBool(value)
			if err != nil {
				if !NoDefault.Is(defaultValue) {
					return defaultValue.(T)
				}
				panic(err)
			}
			return any(v).(T)
		}
	}
	if !NoDefault.Is(defaultValue) {
		return defaultValue.(T)
	}
	panic(fmt.Errorf("%q not set", key))
}
