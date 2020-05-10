/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cfg

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var isInitialized bool

func init() {}

func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	fmt.Print("Unmarshal\n")
	initConfig()
	return viper.Unmarshal(rawVal, opts...)
}

func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	fmt.Print("UnmarshalKey\n")
	initConfig()
	return viper.UnmarshalKey(key, rawVal, opts...)
}

func Get(key string) interface{} {
	return viper.Get(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func ReadInConfig() {
	initConfig()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	fmt.Print("Read config\n")
	if isInitialized {
		return
	}
	isInitialized = true
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".mw" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".mw")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
