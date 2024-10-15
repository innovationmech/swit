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

package switblog

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/pkg/discovery"
	"github.com/innovationmech/swit/internal/switblog/config"
)

type Server struct {
	router *gin.Engine
	sd     *discovery.ServiceDiscovery
}

func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
	}
	// Register Gin routes
	s.registerGinRoutes()

	// Initialize service discovery
	sd, err := discovery.NewServiceDiscovery(config.GetConfig().ServiceDiscovery.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service discovery: %v", err)
	}
	s.sd = sd

	// Register service to service discovery
	err = s.registerService()
	if err != nil {
		return nil, fmt.Errorf("failed to register service: %v", err)
	}

	return s, nil
}

func (s *Server) Run() error {
	return s.router.Run(fmt.Sprintf(":%s", config.GetConfig().Server.Port))
}

func (s *Server) registerService() error {
	return s.sd.RegisterService("switblog", config.GetConfig().Server.Port, 5)
}
