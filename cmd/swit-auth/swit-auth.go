// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
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

// SWIT Auth API
//	@title			SWIT Auth API
//	@version		1.0
//	@description	This is the SWIT authentication service API documentation.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@host		localhost:8090
//	@BasePath	/
//	@schemes	http https

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.

package main

import (
	"os"

	"github.com/innovationmech/swit/internal/base/cli"
	"github.com/innovationmech/swit/internal/switauth/cmd"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Version information set by ldflags during build
var (
	version   = "dev"     // Set by -X main.version
	buildTime = "unknown" // Set by -X main.buildTime
	gitCommit = "unknown" // Set by -X main.gitCommit
)

func run() int {
	// Initialize logger
	logger.InitLogger()
	defer func() {
		if logger.Logger != nil {
			if err := logger.Logger.Sync(); err != nil {
				// Ignore errors on stderr sync - this is expected on some platforms
			}
		}
	}()

	command := cmd.NewSwitAuthCmd()
	if err := cli.Run(command); err != nil {
		logger.Logger.Error("Error occurred while running command", zap.Error(err))
		// Ensure logger sync happens before exit
		if logger.Logger != nil {
			if syncErr := logger.Logger.Sync(); syncErr != nil {
				// Ignore errors on stderr sync - this is expected on some platforms
			}
		}
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
