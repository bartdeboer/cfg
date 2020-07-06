module github.com/bartdeboer/cfg

go 1.14

require (
	github.com/bartdeboer/cobrahooks v0.0.0-20200706095724-4485ab1a6802
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/imdario/mergo v0.3.9
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
)

replace github.com/bartdeboer/cobrahooks => ../cobrahooks/
