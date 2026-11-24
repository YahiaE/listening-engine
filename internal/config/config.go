package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func LoadEnv() {
	// Resolve .env relative to this source file so startup works independently
	// of the shell's current directory.
	_, b, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(b)

	projectRoot := filepath.Join(currentDir, "../../")
	envPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found. Using system environment variables: %v", err)

	}
}

func Get(key string) string {
	return os.Getenv(key)
}
