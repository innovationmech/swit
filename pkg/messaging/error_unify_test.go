package messaging

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
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
