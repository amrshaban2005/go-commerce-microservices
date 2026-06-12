package configloader

import "github.com/joho/godotenv"

func LoadDotEnv(filenames ...string) error {
	return godotenv.Load(filenames...)
}
