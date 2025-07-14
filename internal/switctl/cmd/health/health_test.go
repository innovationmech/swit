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

package health

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewHealthCmd(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "should create health command successfully",
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.NotNil(t, cmd)
				assert.Equal(t, "health", cmd.Use)
				assert.Equal(t, "Check the health status of the SWIT server", cmd.Short)
			},
		},
		{
			name: "should have correct command structure",
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.Empty(t, cmd.Commands()) // No subcommands
				assert.NotNil(t, cmd.RunE)      // Has execution function
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewHealthCmd()
			tt.validate(t, cmd)
		})
	}
}

func TestHealthCommandProperties(t *testing.T) {
	cmd := NewHealthCmd()

	// Test all command properties
	assert.Equal(t, "health", cmd.Use)
	assert.Equal(t, "Check the health status of the SWIT server", cmd.Short)
	assert.Equal(t, "", cmd.Long)
	assert.Empty(t, cmd.Commands())
	assert.NotNil(t, cmd.RunE)
}
