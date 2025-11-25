# Copyright 2025 Swit. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# ABAC Policy for Full Security Stack Example
# This policy implements Attribute-Based Access Control with multiple conditions

package abac

import rego.v1

# Default deny all access
default allow := false

# ===================================
# Main Decision Rules
# ===================================

# Rule 1: Admin bypass - admins have full access
allow if {
    "admin" in input.user.roles
}

# Rule 2: Time and IP-based access control
allow if {
    # User must have valid roles
    count(input.user.roles) > 0
    # Must be during business hours
    is_business_hours
    # IP address must be in allowed range
    is_allowed_ip
    # Operation must be allowed for the user's role
    is_allowed_action
}

# Rule 3: Resource owner access
allow if {
    input.resource.owner == input.user.username
    is_allowed_action
}

# ===================================
# Time-Based Access Control
# ===================================

# Check if current time is within business hours (9:00 - 18:00)
is_business_hours if {
    not input.request.time
}

is_business_hours if {
    input.request.time
    hour := time.clock([input.request.time, "UTC"])[0]
    hour >= 9
    hour < 18
}

# Weekend override for admin users
is_business_hours if {
    "admin" in input.user.roles
}

# ===================================
# IP-Based Access Control
# ===================================

# No IP restriction if not provided
is_allowed_ip if {
    not input.request.client_ip
}

# Allow localhost
is_allowed_ip if {
    input.request.client_ip
    startswith(input.request.client_ip, "127.0.")
}

# Allow private network (192.168.x.x)
is_allowed_ip if {
    input.request.client_ip
    startswith(input.request.client_ip, "192.168.")
}

# Allow private network (10.x.x.x)
is_allowed_ip if {
    input.request.client_ip
    startswith(input.request.client_ip, "10.")
}

# Allow Docker network (172.x.x.x)
is_allowed_ip if {
    input.request.client_ip
    startswith(input.request.client_ip, "172.")
}

# Allow IPv6 localhost
is_allowed_ip if {
    input.request.client_ip
    input.request.client_ip == "::1"
}

# ===================================
# Action-Based Access Control
# ===================================

# Check if the action is allowed
is_allowed_action if {
    input.request.method in ["GET", "POST", "PUT", "DELETE"]
}

# ===================================
# Role and Classification Rules
# ===================================

# Editors can modify documents during business hours
allow if {
    "editor" in input.user.roles
    input.request.method in ["POST", "PUT"]
    input.resource.type == "document"
    is_business_hours
    is_allowed_ip
    input.resource.classification in ["public", "internal"]
}

# Viewers can read documents anytime (not restricted by time)
allow if {
    "viewer" in input.user.roles
    input.request.method == "GET"
    input.resource.type == "document"
    is_allowed_ip
    input.resource.classification in ["public", "internal"]
}

# Editors can read confidential documents
allow if {
    "editor" in input.user.roles
    input.request.method == "GET"
    input.resource.type == "document"
    is_allowed_ip
    input.resource.classification == "confidential"
}

# ===================================
# Profile and Public Access
# ===================================

# All authenticated users can access their profile
allow if {
    input.request.path == "/api/v1/protected/profile"
    input.request.method == "GET"
    count(input.user.roles) > 0
}

# Public endpoints are always accessible
allow if {
    startswith(input.request.path, "/api/v1/public/")
}

# Health and metrics endpoints are always accessible
allow if {
    input.request.path in ["/api/v1/health", "/metrics", "/health", "/ready"]
}

# ===================================
# Department-Based Access
# ===================================

# Users can access documents from their department
allow if {
    input.user.department
    input.resource.department
    input.user.department == input.resource.department
    input.request.method == "GET"
}

# ===================================
# Sensitivity Level Rules
# ===================================

# Define sensitivity levels (higher number = more sensitive)
sensitivity_level(classification) := 1 if {
    classification == "public"
}

sensitivity_level(classification) := 2 if {
    classification == "internal"
}

sensitivity_level(classification) := 3 if {
    classification == "confidential"
}

sensitivity_level(classification) := 4 if {
    classification == "restricted"
}

# User clearance based on role
user_clearance(roles) := 4 if {
    "admin" in roles
}

user_clearance(roles) := 3 if {
    not "admin" in roles
    "editor" in roles
}

user_clearance(roles) := 2 if {
    not "admin" in roles
    not "editor" in roles
    "viewer" in roles
}

user_clearance(roles) := 1 if {
    count(roles) == 0
}

# Allow access if user clearance >= document sensitivity
allow if {
    input.resource.type == "document"
    input.resource.classification
    input.request.method == "GET"
    user_clearance(input.user.roles) >= sensitivity_level(input.resource.classification)
    is_allowed_ip
}

# ===================================
# Audit and Decision Reasons
# ===================================

reason contains msg if {
    "admin" in input.user.roles
    msg := "Access granted: User has admin role"
}

reason contains msg if {
    input.resource.owner == input.user.username
    msg := "Access granted: User is resource owner"
}

reason contains msg if {
    not is_business_hours
    msg := "Access denied: Outside business hours"
}

reason contains msg if {
    not is_allowed_ip
    msg := sprintf("Access denied: IP address %s not in allowed range", [input.request.client_ip])
}

reason contains msg if {
    input.resource.classification
    user_clearance(input.user.roles) < sensitivity_level(input.resource.classification)
    msg := sprintf("Access denied: Insufficient clearance for %s documents", [input.resource.classification])
}

reason contains msg if {
    not allow
    msg := "Access denied: Policy conditions not met"
}

# ===================================
# Metadata
# ===================================

# Policy metadata for introspection
metadata := {
    "name": "abac",
    "version": "1.0.0",
    "description": "Attribute-Based Access Control for Full Security Stack Example",
    "attributes": {
        "user": ["roles", "username", "department"],
        "resource": ["type", "owner", "classification", "department"],
        "environment": ["time", "client_ip"]
    },
    "classifications": ["public", "internal", "confidential", "restricted"]
}

