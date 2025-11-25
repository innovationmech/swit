# Copyright 2025 Swit. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# RBAC Policy for Full Security Stack Example
# This policy implements Role-Based Access Control for document management

package rbac

import rego.v1

# Default deny all access
default allow := false

# ===================================
# Role Definitions
# ===================================

# Admin role - full access to all resources
allow if {
    "admin" in input.user.roles
}

# Editor role - can create, read, update documents
allow if {
    "editor" in input.user.roles
    input.request.method in ["GET", "POST", "PUT"]
    startswith(input.request.path, "/api/v1/protected/documents")
}

# Viewer role - read-only access to documents
allow if {
    "viewer" in input.user.roles
    input.request.method == "GET"
    startswith(input.request.path, "/api/v1/protected/documents")
}

# ===================================
# Resource Owner Permissions
# ===================================

# Document owners can perform any operation on their own documents
allow if {
    input.resource.type == "document"
    input.resource.owner == input.user.username
}

# ===================================
# Profile Access
# ===================================

# All authenticated users can access their own profile
allow if {
    input.request.path == "/api/v1/protected/profile"
    input.request.method == "GET"
    count(input.user.roles) > 0
}

# ===================================
# Public Resource Access
# ===================================

# Health check endpoint is always accessible
allow if {
    input.request.path == "/api/v1/health"
}

# Public info endpoint is always accessible
allow if {
    startswith(input.request.path, "/api/v1/public/")
}

# Metrics endpoint is always accessible
allow if {
    input.request.path == "/metrics"
}

# ===================================
# Admin-Only Endpoints
# ===================================

# Admin dashboard requires admin role
allow if {
    "admin" in input.user.roles
    startswith(input.request.path, "/api/v1/admin/")
}

# ===================================
# Classification-Based Access
# ===================================

# Public documents are accessible to all authenticated users
allow if {
    input.resource.type == "document"
    input.resource.classification == "public"
    input.request.method == "GET"
    count(input.user.roles) > 0
}

# Internal documents require at least viewer role
allow if {
    input.resource.type == "document"
    input.resource.classification == "internal"
    input.request.method == "GET"
    has_role(["viewer", "editor", "admin"])
}

# Confidential documents require editor or admin role
allow if {
    input.resource.type == "document"
    input.resource.classification == "confidential"
    input.request.method == "GET"
    has_role(["editor", "admin"])
}

# Restricted documents require admin role
allow if {
    input.resource.type == "document"
    input.resource.classification == "restricted"
    input.request.method == "GET"
    "admin" in input.user.roles
}

# ===================================
# Helper Functions
# ===================================

# Check if user has any of the specified roles
has_role(required_roles) if {
    some role in input.user.roles
    role in required_roles
}

# ===================================
# Audit and Decision Reasons
# ===================================

# Decision reasons for audit logging
reason contains msg if {
    "admin" in input.user.roles
    msg := "Access granted: User has admin role"
}

reason contains msg if {
    "editor" in input.user.roles
    input.request.method in ["GET", "POST", "PUT"]
    msg := "Access granted: Editor can read, create, and update documents"
}

reason contains msg if {
    "viewer" in input.user.roles
    input.request.method == "GET"
    msg := "Access granted: Viewer can read documents"
}

reason contains msg if {
    input.resource.owner == input.user.username
    msg := "Access granted: User is resource owner"
}

reason contains msg if {
    not allow
    msg := "Access denied: User lacks required permissions"
}

# ===================================
# Metadata
# ===================================

# Policy metadata for introspection
metadata := {
    "name": "rbac",
    "version": "1.0.0",
    "description": "Role-Based Access Control for Full Security Stack Example",
    "roles": ["admin", "editor", "viewer"],
    "classifications": ["public", "internal", "confidential", "restricted"]
}

