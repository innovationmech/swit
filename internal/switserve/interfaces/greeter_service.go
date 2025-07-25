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

package interfaces

import "context"

// GreeterService defines the interface for greeting operations.
// This interface provides methods for generating personalized greetings
// in different languages and formats.
//
// Future implementations should support:
// - Multiple language support
// - Customizable greeting templates
// - Time-based greetings (morning, afternoon, evening)
// - Cultural greeting variations
// - Greeting personalization based on user preferences
type GreeterService interface {
	// GenerateGreeting creates a personalized greeting message.
	// It takes a name and optional language parameter to generate
	// an appropriate greeting. If language is empty, defaults to English.
	// Returns the generated greeting message or an error if generation fails.
	GenerateGreeting(ctx context.Context, name, language string) (string, error)
}
