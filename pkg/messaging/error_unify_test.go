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

package messaging

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToBaseMessagingError_ConvertsLegacy(t *testing.T) {
	le := NewConfigError("invalid config", nil)
	be := ToBaseMessagingError(le)
	if assert.NotNil(t, be) {
		assert.Equal(t, ErrorTypeConfiguration, be.Type)
		assert.Equal(t, ErrCodeInvalidConfig, be.Code)
		assert.False(t, be.Retryable)
	}
}

func TestClassifyError_UsesConvertedLegacy(t *testing.T) {
	classifier := NewErrorClassifier()
	le := NewConfigError("invalid", nil)
	result := classifier.ClassifyError(le)
	assert.Equal(t, ClassificationConfig, result.Classification)
	assert.Equal(t, RetryStrategyNone, result.RetryStrategy)
}

func TestNormalizeError_WrapsGeneric(t *testing.T) {
	gen := errors.New("some io failure")
	be := NormalizeError(gen, ErrorTypeInternal, ErrCodeInternal, "wrap")
	if assert.NotNil(t, be) {
		assert.Equal(t, ErrorTypeInternal, be.Type)
		assert.Equal(t, ErrCodeInternal, be.Code)
	}
}
