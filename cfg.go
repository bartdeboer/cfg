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
	"github.com/mitchellh/mapstructure"
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
	if k := rv.Kind(); k != reflect.Struct && k != reflect.Slice {
		panic("Value is not a struct or slice")
	}
	return rv.Interface() // Get real value of Value
}

func bindCFlags(c *cobra.Command, f *pflag.FlagSet, s interface{}, h func()) {
	createFlags(f, s)
	c.PreRun = func(cmd *cobra.Command, args []string) {
		h()
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

func bindCFlagsPersistent(c *cobra.Command, f *pflag.FlagSet, s interface{}, h func()) {
	createFlags(f, s)
	c.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// first run all the parents
		for p := cmd.Parent(); p != nil; p = p.Parent() {
			if p.PersistentPreRun != nil {
				p.PersistentPreRun(p, args)
				break
			}
		}
		h()
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		c.PersistentPreRun(cmd, args)
		helpFunc(cmd, args)
	})
}

// BindCobraFlags binds a Struct with a viper config when running a Cobra command.
// Generates Cobra flags for the Struct so they can be overriden.
func BindFlags(c *cobra.Command, rawVal interface{}) {
	f := c.Flags()
	bindCFlags(c, f, rawVal, func() {
		Unmarshal(rawVal)
		setFlagDefaults(f, rawVal)
	})
}

// BindCobraFlagsKey binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates Cobra flags for the struct so they can be overriden
func BindFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	f := c.Flags()
	bindCFlags(c, f, rawVal, func() {
		UnmarshalKey(key, rawVal)
		setFlagDefaults(f, rawVal)
	})
}

// BindCobraPersistentFlags persistently binds a Struct with a viper config when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden.
// Runs the parent persistent hooks as well.
func BindPersistentFlags(c *cobra.Command, rawVal interface{}) {
	f := c.PersistentFlags()
	bindCFlagsPersistent(c, f, rawVal, func() {
		Unmarshal(rawVal)
		setFlagDefaults(f, rawVal)
	})
}

// BindCobraFlagsKeyKey persistently binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden
func BindPersistentFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	f := c.PersistentFlags()
	bindCFlagsPersistent(c, f, rawVal, func() {
		UnmarshalKey(key, rawVal)
		setFlagDefaults(f, rawVal)
	})
}

// BindCobraFlagsKeyKey persistently binds a Struct with a viper config at a specific array with dynamic key when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden
func BindPersistentFlagsCollection(colKey string, keyKey string, c *cobra.Command, rawVal interface{}) {
	f := c.PersistentFlags()
	bindCFlagsPersistent(c, f, rawVal, func() {
		var coll []map[string]interface{}
		key := viper.GetString(keyKey)
		UnmarshalKey(colKey, &coll)
		for i := 0; i < len(coll); i++ {
			if val, ok := coll[i]["name"]; ok {
				if val.(string) == key {
					curVal := getPtrValue(rawVal)
					if err := mapstructure.Decode(coll[i], rawVal); err != nil {
						panic(err)
					}
					if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
						panic(err)
					}
					return
				}
			}
		}
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

var ConfigLoader = func() {
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
}

var once sync.Once

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	once.Do(ConfigLoader)
}

func Set(key string, value interface{}) {
	viper.Set(key, value)
}

func Write() error {
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	fmt.Println("Writing config:", viper.ConfigFileUsed())
	return nil
}
