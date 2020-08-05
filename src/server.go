package vod

import (
	"encoding/json"
	"log"
	"os"
)

// Configuration describes the configuration settings acceptable by the application.
// This will be used to parse the config.json file of the application.
type Configuration struct {
	AppName     string `json:"appName"`
	Description string `json:"description"`
	Listen      struct {
		Port int
		Host string
	}
	ServerMode    string `json:"serverMode"`
	MaxUploadSize int64  `json:"maxUploadSize"`
}

// LoadConfig loads configuration file for use within the application.
func LoadConfig(path string) *Configuration {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Printf("Log file at %s does not exist: %s", path, err.Error())
		panic("Cannot continue without config file")
	} else if err != nil {
		log.Fatal(err)
		panic("Cannot continue without config file")
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	Config := new(Configuration)
	err = decoder.Decode(&Config)
	if err != nil {
		log.Fatal(err)
		panic("Cannot continue without config file")
	}
	return Config
}
