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

package main

import (
	"os"

	"github.com/innovationmech/swit/internal/component-base/cli"
	"github.com/innovationmech/swit/internal/pkg/logger"
	"github.com/innovationmech/swit/internal/swit-serve/cmd"
	"go.uber.org/zap"
)

// main is the entry point of the application.
func main() {
	command := cmd.NewRootServeCmdCommand()
	if err := cli.Run(command); err != nil {
		logger.Logger.Error("Error occurred while running command", zap.Error(err))
		os.Exit(1)
	}
	if err := logger.Logger.Sync(); err != nil {
		logger.Logger.Error("Error occurred while syncing logs", zap.Error(err))
	}
}
