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

	"github.com/bartdeboer/cobrahooks"
	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Get(key string) interface{} {
	loadConfig()
	return viper.Get(key)
}

func GetInt(key string) int {
	loadConfig()
	return viper.GetInt(key)
}

func GetString(key string) string {
	loadConfig()
	return viper.GetString(key)
}

func Set(key string, value interface{}) {
	viper.Set(key, value)
}

func ReadInConfig() {
	loadConfig()
}

// Unmashal unmarshals the config into a Struct overriding with any flags that are set
func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	loadConfig()
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
	loadConfig()
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

type BindOptions struct {
	noViper bool
	key     string
}

func NoViper(o *BindOptions) { o.noViper = true }

func Key(key string) func(*BindOptions) {
	return func(o *BindOptions) {
		o.key = key
	}
}

// BindCobraFlags binds a Struct with a viper config when running a Cobra command.
// Generates Cobra flags for the Struct so they can be overriden.
func BindFlags(c *cobra.Command, rawVal interface{}, options ...func(*BindOptions)) {
	var opts BindOptions
	for _, option := range options {
		option(&opts)
	}
	createFlags(c.Flags(), rawVal)
	cobrahooks.OnPreRun(c, func(cmd *cobra.Command, args []string) error {
		fmt.Println("RUN Flags:", c.Use)
		if !opts.noViper {
			if opts.key != "" {
				UnmarshalKey(opts.key, rawVal)
			} else {
				Unmarshal(rawVal)
			}
		}
		setFlagDefaults(c.Flags(), rawVal)
		return nil
	}, cobrahooks.RunOnHelp)
}

// BindCobraFlagsKey binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates Cobra flags for the struct so they can be overriden
func BindFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	BindFlags(c, rawVal, Key(key))
}

// BindCobraPersistentFlags persistently binds a Struct with a viper config when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden.
// Runs the parent persistent hooks as well.
func BindPersistentFlags(c *cobra.Command, rawVal interface{}, options ...func(*BindOptions)) {
	var opts BindOptions
	for _, option := range options {
		option(&opts)
	}
	createFlags(c.PersistentFlags(), rawVal)
	cobrahooks.OnPersistentPreRun(c, func(cmd *cobra.Command, args []string) error {
		fmt.Println("RUN PersistentFlags:", c.Use)
		if !opts.noViper {
			if opts.key != "" {
				UnmarshalKey(opts.key, rawVal)
			} else {
				Unmarshal(rawVal)
			}
		}
		setFlagDefaults(c.PersistentFlags(), rawVal)
		return nil
	}, cobrahooks.RunOnHelp)
}

// BindCobraFlagsKeyKey persistently binds a Struct with a viper config at a specific key when running a Cobra command.
// Generates persistent flags for the struct so they can be overriden
func BindPersistentFlagsKey(key string, c *cobra.Command, rawVal interface{}) {
	BindPersistentFlags(c, rawVal, Key(key))
}

type BindCollectionOptions struct {
	selectField     string
	selectValue     string
	collectionField string
	collection      *[]map[string]interface{}
	bindTo          interface{}
	idField         string
}

func IdField(name string) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.idField = name
	}
}

func SelectField(name string) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.selectField = name
	}
}

func SelectValue(value string) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.selectValue = value
	}
}

func CollectionField(name string) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.collectionField = name
	}
}

func Collection(collection *[]map[string]interface{}) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.collection = collection
	}
}

func BindTo(rawVal interface{}) func(*BindCollectionOptions) {
	return func(o *BindCollectionOptions) {
		o.bindTo = rawVal
	}
}

func BindCollectionItemFields(colField string, selectField string, c *cobra.Command, rawVal interface{}) {
	BindCollectionItem(c, rawVal, CollectionField(colField), SelectField(selectField))
}

func BindCollectionItem(c *cobra.Command, rawVal interface{}, options ...func(*BindCollectionOptions)) {
	var opts BindCollectionOptions
	for _, option := range options {
		option(&opts)
	}
	var idField = opts.idField
	if idField == "" {
		idField = "name"
	}
	var selectField = opts.selectField
	var collField = opts.collectionField
	createFlags(c.PersistentFlags(), rawVal)
	cobrahooks.OnPersistentPreRun(c, func(cmd *cobra.Command, args []string) error {
		fmt.Println("RUN PersistentFlagsCollection:", c.Use)
		selectValue := GetString(selectField)
		var coll []map[string]interface{}
		if opts.collection != nil {
			coll = *opts.collection
		}
		if coll == nil {
			fmt.Println("UNMARSHALL COLLECTION:", c.Use)
			UnmarshalKey(collField, &coll)
		}
		for i := 0; i < len(coll); i++ {
			if val, ok := coll[i][idField]; ok {
				if val.(string) == selectValue {
					curVal := getPtrValue(rawVal)
					if err := mapstructure.Decode(coll[i], rawVal); err != nil {
						return err
					}
					if err := mergo.MergeWithOverwrite(rawVal, curVal); err != nil {
						return err
					}
					setFlagDefaults(c.PersistentFlags(), rawVal)
					return nil
				}
			}
		}
		return nil
	}, cobrahooks.RunOnHelp)
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

	curDir, err := os.Getwd()
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
	viper.AddConfigPath(curDir)
	// viper.SetConfigName("." + name)
	viper.SetConfigName(name)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

var once sync.Once

// initConfig reads in config file and ENV variables if set.
func loadConfig() {
	once.Do(ConfigLoader)
}

func Write() error {
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	fmt.Println("Writing config:", viper.ConfigFileUsed())
	return nil
}
