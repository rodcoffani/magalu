package core

import (
	"bytes"
	"context"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct{}

const (
	CONFIG_NAME = "config"
	CONFIG_TYPE = "yaml"
	CONFIG_PATH = "$HOME/.mgc"
)

var configKey contextKey = "magalu.cloud/core/Config"

func NewConfigContext(parent context.Context, config *Config) context.Context {
	return context.WithValue(parent, configKey, config)
}

func ConfigFromContext(ctx context.Context) *Config {
	c, _ := ctx.Value(configKey).(*Config)
	return c
}

func NewConfig() *Config {
	viper.SetConfigName(CONFIG_NAME)
	viper.SetConfigType(CONFIG_TYPE)
	viper.AddConfigPath(CONFIG_PATH)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return &Config{}
	}
	return &Config{}
}

func (c *Config) BindPFlag(key string, flag *pflag.Flag) error {
	return viper.BindPFlag(key, flag)
}

func (c *Config) IsSet(key string) bool {
	return viper.IsSet(key)
}

func (c *Config) Get(key string) any {
	return viper.Get(key)
}

func (c *Config) Set(key string, value interface{}) {
	viper.Set(key, value)
}

func (c *Config) Delete(key string) error {
	configMap := viper.AllSettings()

	delete(configMap, key)
	encodedConfig, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

	err = viper.ReadConfig(bytes.NewReader(encodedConfig))
	if err != nil {
		return err
	}

	if err = viper.WriteConfig(); err != nil {
		return err
	}

	return nil
}
