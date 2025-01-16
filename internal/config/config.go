package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Db_url            string `json:"db_url"`
	Current_user_name string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

func Read() (Config, error) {
	file, err := os.Open(getConfigFilePath())
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	cfg := Config{}
	err = decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c *Config) SetUser(name string) error {
	c.Current_user_name = name
	return write(*c)
}

func write(c Config) error {
	file, err := os.Create(getConfigFilePath())
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() string {
	home, _ := os.UserHomeDir()

	fullPath := filepath.Join(home, configFileName)
	return fullPath
}
