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

package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// extractPortFromAddress extracts the port number from an address string
// Supports formats like ":8080", "localhost:8080", "127.0.0.1:8080", "[::1]:8080"
func extractPortFromAddress(address string) (int, error) {
	if address == "" {
		return 0, fmt.Errorf("address is empty")
	}

	// Handle IPv6 addresses
	if strings.Contains(address, "]") {
		// IPv6 format like [::1]:8080
		parts := strings.Split(address, "]:")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid IPv6 address format: %s", address)
		}
		portStr := parts[1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return 0, fmt.Errorf("invalid port in IPv6 address %s: %w", address, err)
		}
		return port, nil
	}

	// Handle IPv4 addresses and hostnames
	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		// If SplitHostPort fails, try to handle cases like ":8080"
		if strings.HasPrefix(address, ":") {
			portStr = address[1:]
		} else {
			return 0, fmt.Errorf("failed to parse address %s: %w", address, err)
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port %s: %w", portStr, err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d is out of valid range (1-65535)", port)
	}

	return port, nil
}

// validateAddress validates that an address string is properly formatted
func validateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Try to extract port to validate format
	_, err := extractPortFromAddress(address)
	return err
}

// normalizeAddress normalizes an address string to a consistent format
func normalizeAddress(address string) string {
	if address == "" {
		return ""
	}

	// If address starts with ":" and doesn't contain host, prepend localhost
	if strings.HasPrefix(address, ":") && !strings.Contains(address, "]") {
		return "localhost" + address
	}

	return address
}

// getHostFromAddress extracts the host part from an address string
func getHostFromAddress(address string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("address is empty")
	}

	// Handle IPv6 addresses
	if strings.Contains(address, "]") {
		// IPv6 format like [::1]:8080
		parts := strings.Split(address, "]:")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid IPv6 address format: %s", address)
		}
		// Remove the opening bracket
		host := strings.TrimPrefix(parts[0], "[")
		return host, nil
	}

	// Handle IPv4 addresses and hostnames
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// If SplitHostPort fails, try to handle cases like ":8080"
		if strings.HasPrefix(address, ":") {
			return "localhost", nil
		}
		return "", fmt.Errorf("failed to parse address %s: %w", address, err)
	}

	if host == "" {
		return "localhost", nil
	}

	return host, nil
}
