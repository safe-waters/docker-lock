module github.com/safe-waters/docker-lock

go 1.14

require (
	github.com/docker/cli v20.10.0-beta1.0.20201029214301-1d20b15adc38+incompatible
	github.com/docker/docker-credential-helpers v0.6.3
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	// using buildkit from master rather than a tagged release because
	// https://github.com/moby/buildkit/pull/1425
	// is not fixed in a tag release, but is not a problem
	// from master
	github.com/moby/buildkit v0.7.1-0.20201106222540-703a774918a8
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/sys v0.0.0-20201109165425-215b40eba54c // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
)
