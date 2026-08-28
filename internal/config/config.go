package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"github.com/joho/godotenv"
)

func LoadEnv() {
	_, b, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(b)

	projectRoot := filepath.Join(currentDir, "../../")
	envPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found at %s. Using system env variables")

	}
}

func Get(key string) string {
	return os.Getenv(key)
}