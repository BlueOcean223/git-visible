package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const DefaultMonths = 6

type Config struct {
	Email  string
	Months int
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "git-visible"), nil
}

func File() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func Load() (Config, error) {
	configFile, err := File()
	if err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	v.SetDefault("months", DefaultMonths)
	v.SetDefault("email", "")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	return Config{
		Email:  v.GetString("email"),
		Months: v.GetInt("months"),
	}, nil
}

func Save(config Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	configFile, err := File()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("email", config.Email)
	v.Set("months", config.Months)

	return v.WriteConfigAs(configFile)
}
