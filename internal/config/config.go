package config

import "github.com/joho/godotenv"

func loadEnv() {
	godotenv.Load()
}
