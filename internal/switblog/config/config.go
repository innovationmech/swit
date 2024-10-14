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

	"github.com/spf13/viper"
)

type SwitBlogConfig struct {
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
	Store struct {
		DB struct {
			Host     string `json:"host"`
			Port     string `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			DBName   string `json:"dbname"`
		} `json:"db"`
	} `json:"store"`
	ServiceDiscovery struct {
		Address string `json:"address"`
	} `json:"serviceDiscovery"`
	Grpc struct {
		Port string `json:"port"`
	} `json:"grpc"`
}

var (
	cfg *SwitBlogConfig
)

func GetConfig() *SwitBlogConfig {
	viper.SetConfigName("switblog")
	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config file: %v", err))
	}

	cfg = &SwitBlogConfig{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %v", err))
	}

	return cfg
}
