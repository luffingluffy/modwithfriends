package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// GetConfig ...
func GetConfig(envNames ...string) (map[string]string, error) {
	config := map[string]string{}

	for _, name := range envNames {
		val, ok := os.LookupEnv(name)
		if !ok {
			return nil, fmt.Errorf("\"%s\" environment variable is required but not set", name)
		}
		config[name] = val
	}

	return config, nil
}

// LoadEnvironmentVariables ...
func LoadEnvironmentVariables() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("Failed to load environment variables: %w", err)
	}
	return nil
}

// ToIntOrPanic ...
func ToIntOrPanic(str string) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		panic("Failed to parse \"" + str + "\" to int: " + err.Error())
	}
	return val
}
