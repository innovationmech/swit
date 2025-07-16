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

package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Service implements the greeter business logic
type Service struct {
	// Add dependencies here if needed (e.g., repositories, external services)
}

// NewService creates a new greeter service implementation
func NewService() *Service {
	return &Service{}
}

// GenerateGreeting generates a greeting message based on name and language
func (s *Service) GenerateGreeting(ctx context.Context, name, language string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	// Normalize inputs
	name = strings.TrimSpace(name)
	language = strings.ToLower(strings.TrimSpace(language))

	logger.Logger.Debug("Generating greeting",
		zap.String("name", name),
		zap.String("language", language),
	)

	// Generate greeting based on language
	var greeting string
	switch language {
	case "chinese", "zh", "中文":
		greeting = fmt.Sprintf("你好, %s！", name)
	case "spanish", "es", "español":
		greeting = fmt.Sprintf("¡Hola, %s!", name)
	case "french", "fr", "français":
		greeting = fmt.Sprintf("Bonjour, %s!", name)
	case "german", "de", "deutsch":
		greeting = fmt.Sprintf("Hallo, %s!", name)
	case "japanese", "ja", "日本語":
		greeting = fmt.Sprintf("こんにちは, %s!", name)
	case "korean", "ko", "한국어":
		greeting = fmt.Sprintf("안녕하세요, %s!", name)
	default:
		// Default to English
		greeting = fmt.Sprintf("Hello, %s!", name)
	}

	logger.Logger.Debug("Generated greeting",
		zap.String("greeting", greeting),
		zap.String("language", language),
	)

	return greeting, nil
}
