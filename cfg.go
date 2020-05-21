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

func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	initConfig()
	curVal := getPtrValue(rawVal)
	if err := viper.Unmarshal(rawVal, opts...); err != nil {
		return err
	}
	if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
		return err
	}
	return viper.Unmarshal(rawVal, opts...)
}

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

func BindPersistentFlags(c *cobra.Command, rawVal interface{}) {
	flags := c.PersistentFlags()
	createFlags(flags, rawVal)
	SetCommandHooks(c, func() {
		Unmarshal(rawVal)
		setFlagDefaults(flags, rawVal)
	})
}

func BindFlags(c *cobra.Command, rawVal interface{}) {
	flags := c.Flags()
	createFlags(flags, rawVal)
	SetCommandHooks(c, func() {
		Unmarshal(rawVal)
		setFlagDefaults(flags, rawVal)
	})
}

func BindPersistentFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	flags := c.PersistentFlags()
	createFlags(flags, rawVal)
	SetCommandHooks(c, func() {
		UnmarshalKey(key, rawVal)
		setFlagDefaults(flags, rawVal)
	})
}

func BindFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	flags := c.Flags()
	createFlags(flags, rawVal)
	SetCommandHooks(c, func() {
		UnmarshalKey(key, rawVal)
		setFlagDefaults(flags, rawVal)
	})
}

func SetCommandHooks(c *cobra.Command, hooks ...func()) {
	c.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		for _, hook := range hooks {
			hook()
		}
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		for _, hook := range hooks {
			hook()
		}
		helpFunc(cmd, args)
	})
}

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
