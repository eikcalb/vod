package vod

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
	AWS           struct {
		AWS_ACCESS_KEY_ID     string
		AWS_SECRET_ACCESS_KEY string
		AWS_SESSION_TOKEN     string
	}
}

var (
	// Config contains the application configuration
	Config *Configuration
)

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

func init() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Cannot continue with application", err)
	}
	path, err := filepath.Abs(dir + "/config.json")
	if err != nil {
		log.Fatal("Cannot continue with application", err)
	}
	Config = LoadConfig(path)
}
