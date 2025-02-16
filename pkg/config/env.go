package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFile loads environment variables from a file
func loadEnvFile(filename string) (bool, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}

	foundKey := false
	// Create a map to store the environment variables
	env := make(map[string]string)

	// Split the file into lines
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Find the equals sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) > 1 && (value[0] == '"' || value[0] == '\'') &&
			(value[len(value)-1] == '"' || value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}

		if key == "ANTHROPIC_API_KEY" && value != "" {
			foundKey = true
		}

		env[key] = value
	}

	// Set environment variables
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			return false, fmt.Errorf("error setting environment variable %s: %v", key, err)
		}
	}

	return foundKey, nil
}

// GetAnthropicAPIKey attempts to get the Anthropic API key from either .env file or environment variables
func GetAnthropicAPIKey() (string, error) {
	// First try environment variable
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return key, nil
	}

	// If not in environment, try .env file
	foundKey, err := loadEnvFile(".env")
	if err == nil && foundKey {
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key, nil
		}
	}

	// If still not found, try .env.local
	foundKey, err = loadEnvFile(".env.local")
	if err == nil && foundKey {
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key, nil
		}
	}

	// Get the current working directory for the error message
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "current directory"
	}

	return "", fmt.Errorf("ANTHROPIC_API_KEY not found in environment variables or .env files.\n"+
		"Please either:\n"+
		"1. Create a .env file in %s with ANTHROPIC_API_KEY='your-api-key', or\n"+
		"2. Set it in your environment with: export ANTHROPIC_API_KEY='your-api-key'",
		filepath.Clean(cwd))
}
