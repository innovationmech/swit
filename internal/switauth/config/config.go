package config

import (
	"github.com/spf13/viper"
	"sync"
)

var (
	JwtSecret = []byte("my-256-bit-secret")
	config    *AuthConfig
	once      sync.Once
)

type AuthConfig struct {
	BaseUrl  string
	Database struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		DBName   string `json:"dbname"`
	} `json:"database"`
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
}

func GetConfig() *AuthConfig {
	once.Do(func() {
		viper.SetConfigName("switauth")
		viper.AddConfigPath(".")
		err := viper.ReadInConfig()
		if err != nil {
			panic(err)
		}
		config = &AuthConfig{}
		err = viper.Unmarshal(config)
		if err != nil {
			panic(err)
		}
	})
	return config
}
