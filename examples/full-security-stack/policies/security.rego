# Copyright 2025 Swit. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Security Policy for Full Security Stack Example
# This policy implements additional security checks and constraints

package security

import rego.v1

# ===================================
# Rate Limiting Rules
# ===================================

# Check if request rate is within limits
rate_limit_exceeded if {
    input.request.rate_limit
    input.request.rate_limit.current > input.request.rate_limit.max
}

# ===================================
# Token Validation Rules
# ===================================

# Check if token is valid
token_valid if {
    input.token
    input.token.exp > time.now_ns() / 1000000000
    input.token.iat < time.now_ns() / 1000000000
}

# Check if token is about to expire (within 5 minutes)
token_expiring_soon if {
    input.token
    input.token.exp - (time.now_ns() / 1000000000) < 300
}

# ===================================
# Certificate Validation Rules
# ===================================

# Check if client certificate is valid
certificate_valid if {
    input.certificate
    input.certificate.not_before < time.now_ns() / 1000000000
    input.certificate.not_after > time.now_ns() / 1000000000
}

# Check if certificate is about to expire (within 30 days)
certificate_expiring_soon if {
    input.certificate
    input.certificate.not_after - (time.now_ns() / 1000000000) < 2592000
}

# ===================================
# Suspicious Activity Detection
# ===================================

# Detect suspicious user agent patterns
suspicious_user_agent if {
    input.request.user_agent
    contains(lower(input.request.user_agent), "curl")
    not input.request.authenticated
}

suspicious_user_agent if {
    input.request.user_agent
    contains(lower(input.request.user_agent), "wget")
    not input.request.authenticated
}

suspicious_user_agent if {
    input.request.user_agent
    contains(lower(input.request.user_agent), "scanner")
}

# Detect potential SQL injection attempts
sql_injection_attempt if {
    input.request.query_params
    some param in input.request.query_params
    contains(lower(param), "select")
    contains(lower(param), "from")
}

sql_injection_attempt if {
    input.request.query_params
    some param in input.request.query_params
    contains(lower(param), "union")
}

sql_injection_attempt if {
    input.request.query_params
    some param in input.request.query_params
    contains(lower(param), "drop")
    contains(lower(param), "table")
}

# Detect potential XSS attempts
xss_attempt if {
    input.request.query_params
    some param in input.request.query_params
    contains(lower(param), "<script")
}

xss_attempt if {
    input.request.query_params
    some param in input.request.query_params
    contains(lower(param), "javascript:")
}

# ===================================
# Security Alerts
# ===================================

# Generate security alerts
alerts contains alert if {
    rate_limit_exceeded
    alert := {
        "type": "rate_limit_exceeded",
        "severity": "high",
        "message": "Rate limit exceeded for this client"
    }
}

alerts contains alert if {
    token_expiring_soon
    alert := {
        "type": "token_expiring",
        "severity": "low",
        "message": "Token is expiring soon"
    }
}

alerts contains alert if {
    certificate_expiring_soon
    alert := {
        "type": "certificate_expiring",
        "severity": "medium",
        "message": "Client certificate is expiring soon"
    }
}

alerts contains alert if {
    suspicious_user_agent
    alert := {
        "type": "suspicious_user_agent",
        "severity": "medium",
        "message": "Suspicious user agent detected"
    }
}

alerts contains alert if {
    sql_injection_attempt
    alert := {
        "type": "sql_injection_attempt",
        "severity": "critical",
        "message": "Potential SQL injection attempt detected"
    }
}

alerts contains alert if {
    xss_attempt
    alert := {
        "type": "xss_attempt",
        "severity": "critical",
        "message": "Potential XSS attempt detected"
    }
}

# ===================================
# Security Score
# ===================================

# Calculate security score (0-100)
security_score := score if {
    base_score := 100
    
    # Deduct points for various security issues
    rate_limit_penalty := 20 if rate_limit_exceeded else := 0
    token_penalty := 5 if token_expiring_soon else := 0
    cert_penalty := 10 if certificate_expiring_soon else := 0
    ua_penalty := 15 if suspicious_user_agent else := 0
    sqli_penalty := 50 if sql_injection_attempt else := 0
    xss_penalty := 50 if xss_attempt else := 0
    
    total_penalty := rate_limit_penalty + token_penalty + cert_penalty + ua_penalty + sqli_penalty + xss_penalty
    score := max([0, base_score - total_penalty])
}

# ===================================
# Security Decision
# ===================================

# Block request if critical security issues detected
block_request if {
    sql_injection_attempt
}

block_request if {
    xss_attempt
}

block_request if {
    rate_limit_exceeded
}

# Allow request if no critical security issues
allow if {
    not block_request
}

# Default deny
default allow := false

# ===================================
# Metadata
# ===================================

metadata := {
    "name": "security",
    "version": "1.0.0",
    "description": "Security checks and constraints for Full Security Stack Example",
    "checks": [
        "rate_limiting",
        "token_validation",
        "certificate_validation",
        "suspicious_activity_detection",
        "sql_injection_detection",
        "xss_detection"
    ]
}

