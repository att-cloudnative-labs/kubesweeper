package main

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
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
	Reasons  			[]SweeperConfigDetails `mapstructure:"reasons"`
	DayLimit 			int                    `mapstructure:"day_limit"`
	DeleteIngresses		bool				   `mapstructure:"delete_ingresses"`
	DeleteServices      bool				   `mapstructure:"delete_services"`
	DeleteHpas			bool				   `mapstructure:"delete_hpas"`
	ExcludedNamespaces  []string			   `mapstructure:"excluded_namespaces"`
}

/**
 * This is the object that holds the necessary information
 */
type SweeperConfigDetails struct {
	Reason           string `mapstructure:"reason"`
	RestartThreshold int    `mapstructure:"restart_threshold,omitempty"`
	DeleteFuncString string `mapstructure:"delete_func_string"`
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
	v := viper.New()
	
	// we'll read config in from YAML
	v.SetConfigType("yaml")

	// The config file name is config.yaml
	v.SetConfigName("config")

	// Just in case we want to set another directory via an environment variable
	configDir := os.Getenv("GO_CONFIG_DIR")
	if len(configDir) > 0 {
		v.AddConfigPath(configDir)
	}

	v.AddConfigPath("./configs")
	// read the config file into memory

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := v.ReadInConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	// unmarshal yaml file into ConfigObj member
	err = v.Unmarshal(&ConfigObj)

	if err != nil {
		log.Fatal(err.Error())
	}

	// this is necessary to assign the appropriate functions to the right objects
	ConfigObj.SetFunctions(funcMap)
}
