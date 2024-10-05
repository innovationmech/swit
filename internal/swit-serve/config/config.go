package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
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

var (
	cfg    *Config
	once   sync.Once
	Logger *zap.Logger
)

func GetConfig() *Config {
	once.Do(func() {
		viper.SetConfigName("swit")
		viper.AddConfigPath(".")
		err := viper.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("FATAL ERROR CONFIG FILE: %s", err))
		}

		cfg = &Config{}
		err = viper.Unmarshal(cfg)
		if err != nil {
			panic(fmt.Errorf("UNABLE TO DECODE INTO STRUCT, %v", err))
		}
	})
	return cfg
}

func InitLogger() {
	var err error
	Logger, err = zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("FATAL ERROR LOGGER: %s", err))
	}
}
