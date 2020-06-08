// Copyright 2009 Bart de Boer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Get(key string) interface{} {
	initConfig()
	return viper.Get(key)
}

func GetInt(key string) int {
	initConfig()
	return viper.GetInt(key)
}

func GetString(key string) string {
	initConfig()
	return viper.GetString(key)
}

func ReadInConfig() {
	initConfig()
}

// Unmashal unmarshals the config into a Struct overriding with any flags that are set
func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	fmt.Printf("Unmarshal\n")
	initConfig()
	curVal := getPtrValue(rawVal)
	if err := viper.Unmarshal(rawVal, opts...); err != nil {
		return err
	}
	if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
		return err
	}
	return nil
}

// Unmashal takes a single key and unmarshals it into a Struct overriding with any flags that are set
func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	fmt.Printf("UnmarshalKey for %s\n", key)
	initConfig()
	curVal := getPtrValue(rawVal)
	if err := viper.UnmarshalKey(key, rawVal, opts...); err != nil {
		return err
	}
	if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
		return err
	}
	return nil
}

// getPtrValue Gets the real struct value of a pointer
func getPtrValue(i interface{}) interface{} {
	rvp := reflect.ValueOf(i)
	if k := rvp.Kind(); k != reflect.Ptr {
		panic("Value is not a pointer")
	}
	rv := rvp.Elem() // struct value from pointer
	if k := rv.Kind(); k != reflect.Struct {
		panic("Value is not a struct")
	}
	return rv.Interface() // Get real value of Value
}

// BindCobraFlags binds a Struct with a viper config when running a Cobra command.
// Generates Cobra flags for the Struct so they can be overriden.
func BindCobraFlags(c *cobra.Command, rawVal interface{}) {
	fmt.Printf("BindFlags for %s\n", c.Use)
	flags := c.Flags()
	createFlags(flags, rawVal)
	c.PreRun = func(cmd *cobra.Command, args []string) {
		Unmarshal(rawVal)
		setFlagDefaults(flags, rawVal)
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PersistentPreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

// BindCobraFlagsKey binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates Cobra flags for the struct so they can be overriden
func BindCobraFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	fmt.Printf("BindFlagsKey for %s\n", c.Use)
	flags := c.Flags()
	createFlags(flags, rawVal)
	c.PreRun = func(cmd *cobra.Command, args []string) {
		UnmarshalKey(key, rawVal)
		setFlagDefaults(flags, rawVal)
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PersistentPreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

// BindCobraPersistentFlags persistently binds a Struct with a viper config when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden.
// Runs the parent persistent hooks as well.
func BindCobraPersistentFlags(c *cobra.Command, rawVal interface{}) {
	fmt.Printf("BindCobraPersistentFlags for %s\n", c.Use)
	flags := c.PersistentFlags()
	createFlags(flags, rawVal)
	c.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// first run all the parents
		for p := cmd.Parent(); p != nil; p = p.Parent() {
			if p.PersistentPreRun != nil {
				p.PersistentPreRun(p, args)
				break
			}
		}
		fmt.Printf("PersistentPreRun for %s\n", cmd.Use)
		Unmarshal(rawVal)
		setFlagDefaults(flags, rawVal)
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PersistentPreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

// BindCobraFlagsKeyKey persistently binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden
func BindCobraPersistentFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	fmt.Printf("BindPersistentFlagsKey for %s\n", c.Use)
	flags := c.PersistentFlags()
	createFlags(flags, rawVal)
	c.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		for p := c.Parent(); p != nil; p = p.Parent() {
			if p.PersistentPreRun != nil {
				p.PersistentPreRun(cmd, args)
				break
			}
		}
		UnmarshalKey(key, rawVal)
		setFlagDefaults(flags, rawVal)
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PersistentPreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

// setFlagDefaults takes the values of a Struct and sets them as flag defaults
func setFlagDefaults(flags *pflag.FlagSet, rawVal interface{}) {
	rvp := reflect.ValueOf(rawVal) // pointer struct value
	rtp := reflect.TypeOf(rawVal)  // pointer struct type
	if k := rvp.Kind(); k != reflect.Ptr {
		panic("Value is not a pointer")
	}
	rv := rvp.Elem() // struct value from pointer
	rt := rtp.Elem() // struct type from pointer
	if k := rv.Kind(); k != reflect.Struct {
		panic("Value is not a struct")
	}
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i) // value
		ft := rt.Field(i) // struct field type
		flag := flags.Lookup(strcase.ToKebab(ft.Name))
		if flag != nil {
			flag.DefValue = fmt.Sprintf("%v", fv.Interface())
		}
	}
}

// Generates cobra flags based on a Struct
func createFlags(flags *pflag.FlagSet, rawVal interface{}) {
	// https://blog.golang.org/laws-of-reflection
	rvp := reflect.ValueOf(rawVal) // pointer struct value
	rtp := reflect.TypeOf(rawVal)  // pointer struct type
	if k := rvp.Kind(); k != reflect.Ptr {
		panic("Value is not a pointer")
	}
	rv := rvp.Elem() // struct value from pointer
	rt := rtp.Elem() // struct type from pointer
	if k := rv.Kind(); k != reflect.Struct {
		panic("Value is not a struct")
	}
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i) // value
		ft := rt.Field(i) // struct field type
		flagName := strcase.ToKebab(ft.Name)
		switch fv.Kind() {
		case reflect.Bool:
			flags.BoolVarP(
				fv.Addr().Interface().(*bool),
				flagName, "",
				fv.Interface().(bool),
				ft.Tag.Get("usage"))
			break
		case reflect.String:
			flags.StringVarP(
				fv.Addr().Interface().(*string),
				flagName, "",
				fv.Interface().(string),
				ft.Tag.Get("usage"))
			break
		case reflect.Float64:
			flags.Float64VarP(
				fv.Addr().Interface().(*float64),
				flagName, "",
				fv.Interface().(float64),
				ft.Tag.Get("usage"))
			break
		case reflect.Int:
			flags.IntVarP(
				fv.Addr().Interface().(*int),
				flagName, "",
				fv.Interface().(int),
				ft.Tag.Get("usage"))
			break
		}
	}
}

var once sync.Once

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	once.Do(func() {
		fmt.Printf("initConfig\n")
		// if cfgFile != "" {
		// 	// Use config file from the flag.
		// 	viper.SetConfigFile(cfgFile)
		// 	return
		// }

		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		exec, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		name := strings.TrimSuffix(filepath.Base(exec), (".exe"))

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName("." + name)
		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	})
}

func WriteConfig() error {
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	fmt.Println("Writing config:", viper.ConfigFileUsed())
	return nil
}
