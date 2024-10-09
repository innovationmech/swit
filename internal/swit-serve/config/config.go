// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
// 
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// 
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
// 
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
// 

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
