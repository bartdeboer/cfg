package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string
var isInitialized bool

func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	initConfig()
	return viper.Unmarshal(rawVal, opts...)
}

func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	initConfig()
	rvp := reflect.ValueOf(rawVal)
	if k := rvp.Kind(); k != reflect.Ptr {
		panic("Value is not a pointer")
	}
	rv := rvp.Elem() // struct value from pointer
	if k := rv.Kind(); k != reflect.Struct {
		panic("Value is not a struct")
	}
	curVal := rv.Interface() // Get real value of Value
	if err := viper.UnmarshalKey(key, rawVal, opts...); err != nil {
		return err
	}
	if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
		return err
	}
	return nil
}

func BindPersistentFlags(c *cobra.Command, key string, rawVal interface{}) {
	BindCommandFlags(c, c.PersistentFlags(), key, rawVal)
}

func BindFlags(c *cobra.Command, key string, rawVal interface{}) {
	BindCommandFlags(c, c.Flags(), key, rawVal)
}

func BindCommandFlags(c *cobra.Command, flags *pflag.FlagSet, key string, rawVal interface{}) {
	createFlags(flags, rawVal)
	c.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		UnmarshalKey(key, rawVal)
		updateFlags(flags, rawVal)
	}
	helpFunc := c.HelpFunc()
	c.SetHelpFunc(func(cmd *cobra.Command, s []string) {
		UnmarshalKey(key, rawVal)
		updateFlags(flags, rawVal)
		helpFunc(cmd, s)
	})
}

func updateFlags(flags *pflag.FlagSet, rawVal interface{}) {
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

		exec, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		name := strings.TrimSuffix(filepath.Base(exec), (".exe"))

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName("." + name)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
