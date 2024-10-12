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
	"sync"

	"github.com/spf13/viper"
)

var (
	JwtSecret = []byte("my-256-bit-secret")
	config    *AuthConfig
	once      sync.Once
)

type AuthConfig struct {
	Database struct {
		Username string `json:"username" yaml:"username"`
		Password string `json:"password" yaml:"password"`
		Host     string `json:"host" yaml:"host"`
		Port     string `json:"port" yaml:"port"`
		DBName   string `json:"dbname" yaml:"dbname"`
	} `json:"database"`
	Server struct {
		Port string `json:"port" yaml:"port"`
	} `json:"server" yaml:"server"`
	ServiceDiscovery struct {
		Address string `json:"address" yaml:"address"`
	} `json:"serviceDiscovery" yaml:"serviceDiscovery"`
}

func GetConfig() *AuthConfig {
	once.Do(func() {
		viper.SetConfigName("switauth")
		viper.SetConfigType("yaml")
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
