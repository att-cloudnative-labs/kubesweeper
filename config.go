package main

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

// make this config object available outside
var ConfigObj = KleanerConfig{}

// This map would need to be updated to account for any other behavior we would want to enable
// The keys in this map have to map directly to the DeleteFuncString in the SweeperConfigDetails struct
var funcMap = map[string]DeleteFunc{
	"DeleteCrash":   DeleteCrash,
	"DeleteGeneric": DeleteGeneric,
}

/**
 * This is the parent config object
 */
type KleanerConfig struct {
	Reasons  []SweeperConfigDetails `yaml:"reasons"`
	DayLimit int                    `yaml:"dayLimit"`
}

/**
 * This is the object that holds the necessary information
 */
type SweeperConfigDetails struct {
	Reason           string `yaml:"reason"`
	RestartThreshold int    `yaml:"restartThreshold,omitempty"`
	DeleteFuncString string `yaml:"deleteFuncString"`
	DeleteFunction   DeleteFunc
}

func (dc *KleanerConfig) SetFunctions(fnMap map[string]DeleteFunc) {
	for i, reason := range dc.Reasons {
		// get the appropriate function out of the map
		val, ok := fnMap[reason.DeleteFuncString]
		if ok {
			// add the appropriate function call to the referenced object
			dc.Reasons[i].DeleteFunction = val
		}
	}
}

func init() {
	// we'll read config in from YAML
	viper.SetConfigType("yaml")

	// The config file name is config.yaml
	viper.SetConfigName("config")

	// Just in case we want to set another directory via an environment variable
	configDir := os.Getenv("GO_CONFIG_DIR")
	if len(configDir) > 0 {
		viper.AddConfigPath(configDir)
	}

	viper.AddConfigPath("./configs")
	// read the config file into memory
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	// unmarshal yaml file into ConfigObj member
	err = viper.UnmarshalKey("kleaner", &ConfigObj)
	viper.SetEnvPrefix("kleaner")
	viper.AutomaticEnv()

	if err != nil {
		log.Fatal(err.Error())
	}

	// this is necessary to assign the appropriate functions to the right objects
	ConfigObj.SetFunctions(funcMap)
}
