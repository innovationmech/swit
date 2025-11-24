# OPA Policy Writing and Integration Guide

This guide covers how to write and integrate policies using Open Policy Agent (OPA), including Rego language basics, policy writing best practices, RBAC and ABAC examples, testing, and performance optimization.

## Table of Contents

- [Overview](#overview)
- [Rego Language Basics](#rego-language-basics)
- [Policy Writing Best Practices](#policy-writing-best-practices)
- [RBAC Policy Examples](#rbac-policy-examples)
- [ABAC Policy Examples](#abac-policy-examples)
- [Policy Testing](#policy-testing)
- [Performance Optimization](#performance-optimization)
- [FAQ](#faq)

## Overview

Open Policy Agent (OPA) is an open-source general-purpose policy engine that uses the declarative Rego language to define policies. OPA enables unified policy enforcement across your entire stack, supporting use cases like API authorization, data filtering, and Kubernetes admission control.

### Core Concepts

#### 1. Policy as Code

OPA allows you to define policies as code, providing:

- **Version Control**: Policies can be versioned like code
- **Testing**: Write unit and integration tests for policies
- **Reusability**: Policies can be modularized and reused
- **Audit Trail**: Complete audit trail for all policy changes

#### 2. Decoupling Decision from Enforcement

OPA separates policy decisions from policy enforcement:

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│ Application │───────>│     OPA     │───────>│  Decision   │
│ (Enforcement)│ Query │  (Engine)   │ Result │ (allow/deny)│
└─────────────┘        └─────────────┘        └─────────────┘
                              │
                              │ Load
                              ▼
                        ┌─────────────┐
                        │ Rego Policy │
                        └─────────────┘
```

#### 3. Data-Driven Decisions

OPA makes decisions based on three inputs:

- **Input**: Request context (user, resource, action, etc.)
- **Data**: External data (user lists, resource attributes, etc.)
- **Policy**: Rego policy rules

### Why Use OPA?

| Feature | Traditional Approach | OPA Approach |
|---------|---------------------|--------------|
| Policy Location | Scattered in code | Centralized |
| Policy Changes | Requires redeployment | Dynamic loading |
| Policy Testing | Hard to test | Independent testing |
| Policy Reuse | Duplicated code | Modular reuse |
| Audit Trail | Difficult to track | Complete audit |

## Rego Language Basics

Rego is OPA's declarative query language designed specifically for writing policies.

### Basic Syntax

#### 1. Rules

Rules are the building blocks of Rego:

```rego
package example

import rego.v1

# Simple rule returning boolean
allow if {
    input.user == "alice"
}

# Rule with conditions
allow if {
    input.user == "bob"
    input.action == "read"
}

# Multiple rules (OR relationship)
allow if {
    input.user == "admin"
}

# Rule defining a value
user_name := input.user

# Generating sets
admins contains user if {
    some user in data.users
    user.role == "admin"
}
```

#### 2. Data Types

Rego supports the following data types:

```rego
# Strings
name := "alice"

# Numbers
count := 42
price := 19.99

# Booleans
is_admin := true

# Arrays
users := ["alice", "bob", "charlie"]

# Objects
user := {
    "name": "alice",
    "role": "admin",
    "age": 30
}

# Sets
admin_set := {"alice", "bob"}
```

#### 3. Operators

```rego
# Comparison operators
x == y  # equals
x != y  # not equals
x < y   # less than
x <= y  # less than or equal
x > y   # greater than
x >= y  # greater than or equal

# Logical operators (implicit AND)
rule if {
    condition1
    condition2  # AND
}

# Membership operator
"alice" in ["alice", "bob"]  # true
"admin" in user.roles        # check array membership

# Negation
not condition
```

#### 4. Built-in Functions

Rego provides rich built-in functions:

```rego
# String functions
startswith("hello world", "hello")  # true
contains("hello world", "world")     # true
lower("HELLO")                       # "hello"
upper("hello")                       # "HELLO"
split("a,b,c", ",")                  # ["a", "b", "c"]

# Array functions
count([1, 2, 3])                    # 3
concat(",", ["a", "b", "c"])        # "a,b,c"

# Object functions
object.get(obj, "key", "default")   # get with default
object.keys({"a": 1, "b": 2})       # ["a", "b"]

# Set functions
count({"a", "b", "c"})              # 3

# Time functions
time.now_ns()                       # current time (nanoseconds)
time.parse_ns("2006-01-02", "2024-01-01")  # parse time

# Type checking
is_string("hello")                  # true
is_number(42)                       # true
is_boolean(true)                    # true
is_array([1, 2, 3])                 # true
is_object({"a": 1})                 # true
```

#### 5. Iteration and Quantifiers

```rego
# some: existential quantifier
allow if {
    some role in input.user.roles
    role == "admin"
}

# every: universal quantifier
all_positive if {
    every item in [1, 2, 3] {
        item > 0
    }
}

# Iterating collections
user_names contains name if {
    some user in data.users
    name := user.name
}

# Multiple iterations
pairs contains pair if {
    some x in [1, 2, 3]
    some y in ["a", "b", "c"]
    pair := {"x": x, "y": y}
}
```

### Policy Structure

A complete Rego policy file typically includes:

```rego
# 1. Package declaration (required)
package example.authz

# 2. Import statements
import rego.v1
import data.users
import data.roles

# 3. Default rules
default allow := false

# 4. Main decision rules
allow if {
    user_is_admin
}

allow if {
    user_has_permission
}

# 5. Helper rules
user_is_admin if {
    "admin" in input.user.roles
}

user_has_permission if {
    some role in input.user.roles
    some permission in role_permissions[role]
    permission == input.action
}

# 6. Data definitions
role_permissions := {
    "admin": ["read", "write", "delete"],
    "editor": ["read", "write"],
    "viewer": ["read"],
}

# 7. Audit rules
audit_log := {
    "user": input.user.name,
    "action": input.action,
    "resource": input.resource,
    "allowed": allow,
    "timestamp": time.now_ns(),
}
```

### Advanced Rego Features

#### 1. Comprehensions

```rego
# Array comprehension
numbers := [x | x := [1, 2, 3, 4, 5][_]; x > 2]
# Result: [3, 4, 5]

# Object comprehension
admin_ages := {name: age |
    some user in data.users
    user.role == "admin"
    name := user.name
    age := user.age
}

# Set comprehension
admin_names := {name |
    some user in data.users
    user.role == "admin"
    name := user.name
}
```

#### 2. Functions

```rego
# Define function
max_value(a, b) := a if {
    a >= b
} else := b

# Function with default
get_role(user) := role if {
    role := user.role
} else := "viewer"

# Using functions
result := max_value(10, 20)  # 20
```

#### 3. Partial Rules

```rego
# Generate object
user_permissions[user] := permissions if {
    some user in data.users
    permissions := user.permissions
}

# Result is a map: {"alice": [...], "bob": [...]}
```

#### 4. With Keyword

The `with` keyword is used to override input or data during testing or evaluation:

```rego
# Policy
allow if {
    input.user == "alice"
}

# Tests
test_allow_alice if {
    allow with input as {"user": "alice"}
}

test_deny_bob if {
    not allow with input as {"user": "bob"}
}
```

## Policy Writing Best Practices

### 1. Organizing Policy Files

#### Directory Structure

```
policies/
├── common/
│   ├── helpers.rego          # Common helper functions
│   └── constants.rego        # Constant definitions
├── authz/
│   ├── rbac.rego             # RBAC policies
│   ├── rbac_test.rego        # RBAC tests
│   ├── abac.rego             # ABAC policies
│   └── abac_test.rego        # ABAC tests
├── examples/
│   ├── healthcare.rego       # Healthcare example
│   ├── financial.rego        # Financial example
│   └── multi_tenant.rego     # Multi-tenant example
└── data/
    ├── users.json            # User data
    └── roles.json            # Role data
```

#### Package Naming Convention

```rego
# Use hierarchical package names
package example.authz.rbac
package example.authz.abac
package example.api.validation
```

### 2. Default Deny Principle

Always use default deny - it's a security best practice:

```rego
# ✅ Recommended: Default deny
default allow := false

allow if {
    # Explicit allow conditions
}

# ❌ Avoid: Default allow
default allow := true  # Insecure!

deny if {
    # This approach is error-prone
}
```

### 3. Modularize Policies

Break complex policies into small, reusable rules:

```rego
# ✅ Recommended: Modular
allow if {
    user_authenticated
    user_authorized
    resource_accessible
}

user_authenticated if {
    # Authentication logic
}

user_authorized if {
    # Authorization logic
}

resource_accessible if {
    # Resource access check
}

# ❌ Avoid: Monolithic rule
allow if {
    # All logic in one rule
    input.user.token != ""
    some role in input.user.roles
    role_permissions[role][input.action]
    input.resource.owner == input.user.name
    # ... more conditions
}
```

### 4. Use Meaningful Names

```rego
# ✅ Recommended: Clear naming
user_is_admin if {
    "admin" in input.user.roles
}

user_has_permission(action) if {
    some role in input.user.roles
    role_permissions[role][action]
}

# ❌ Avoid: Ambiguous naming
check1 if {
    "admin" in input.user.roles
}

x(a) if {
    some r in input.user.roles
    y[r][a]
}
```

### 5. Add Documentation

Provide clear documentation for policies:

```rego
# RBAC (Role-Based Access Control) Policy
#
# This policy implements role-based access control with support for:
# - Role permission mapping
# - Role resource mapping
# - Super admin support
# - Audit logging
#
# Input format:
#   {
#     "subject": {"user": "alice", "roles": ["editor"]},
#     "action": "read",
#     "resource": "documents"
#   }
#
# Returns:
#   allow: true/false
#
package rbac

import rego.v1

# Default deny all access
default allow := false

# Main decision rule
# Allow access when user has appropriate role permissions
allow if {
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}
```

### 6. Input Validation

Always validate input data integrity:

```rego
# ✅ Recommended: Validate input
allow if {
    # Validate required fields exist
    input.subject
    input.subject.user
    input.action
    input.resource
    
    # Perform actual authorization check
    user_authorized
}

# ❌ Avoid: No input validation
allow if {
    # Direct use of potentially non-existent fields
    input.subject.user == "admin"  # Error if subject doesn't exist
}
```

### 7. Error Handling

Use default values and conditional checks to handle missing data:

```rego
# Use object.get for default values
clearance_level := object.get(
    input.subject.attributes,
    "clearance_level",
    0  # default value
)

# Check if field exists
allow if {
    input.subject.attributes
    input.subject.attributes.clearance_level
    input.subject.attributes.clearance_level >= 3
}

# Or use else clause
user_role := input.subject.role if {
    input.subject.role
} else := "guest"
```

### 8. Performance Considerations

#### Early Exit

```rego
# ✅ Recommended: Fast path first
allow if {
    "admin" in input.subject.roles  # Quick check
}

allow if {
    user_has_permission  # Complex check later
}

# ❌ Avoid: Complex check first
allow if {
    complex_calculation_result >= threshold
}

allow if {
    "admin" in input.subject.roles
}
```

#### Avoid Unnecessary Computation

```rego
# ✅ Recommended: Conditional computation
allow if {
    user_authenticated
    # Only compute complex score after user is authenticated
    dynamic_score >= required_score
}

# ❌ Avoid: Always compute
allow if {
    dynamic_score >= required_score  # Computed even if user not authenticated
}
```

### 9. Test-Driven Development

Write tests for every policy:

```rego
package rbac_test

import rego.v1
import data.rbac

# Test success scenarios
test_admin_has_all_permissions if {
    rbac.allow with input as {
        "subject": {"user": "alice", "roles": ["admin"]},
        "action": "delete",
        "resource": "users"
    }
}

# Test denial scenarios
test_viewer_cannot_delete if {
    not rbac.allow with input as {
        "subject": {"user": "bob", "roles": ["viewer"]},
        "action": "delete",
        "resource": "documents"
    }
}

# Test edge cases
test_empty_roles_denied if {
    not rbac.allow with input as {
        "subject": {"user": "charlie", "roles": []},
        "action": "read",
        "resource": "documents"
    }
}
```

### 10. Version Control Policies

```rego
# Include version information in policies
metadata := {
    "version": "1.0.0",
    "description": "RBAC policy for document management",
    "author": "security-team",
    "last_updated": "2024-01-15"
}
```

## RBAC Policy Examples

Role-Based Access Control (RBAC) is the most commonly used access control model. Here's a complete RBAC implementation example.

### Basic RBAC Policy

```rego
package rbac

import rego.v1

# Default deny
default allow := false

# Main decision rule
allow if {
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}

# Super admin has all permissions
allow if {
    "admin" in user_roles
}

# Get user roles
user_roles contains role if {
    some role in input.subject.roles
}

# Role permission mapping
role_permissions := {
    "admin": {
        "create": true,
        "read": true,
        "update": true,
        "delete": true,
        "list": true,
    },
    "editor": {
        "create": true,
        "read": true,
        "update": true,
        "list": true,
    },
    "viewer": {
        "read": true,
        "list": true,
    },
}

# Role resource mapping
role_resources := {
    "admin": {
        "users": true,
        "documents": true,
        "settings": true,
        "reports": true,
    },
    "editor": {
        "documents": true,
        "reports": true,
    },
    "viewer": {
        "documents": true,
        "reports": true,
    },
}
```

### Using RBAC Policy

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    ctx := context.Background()
    
    // Create evaluator
    evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./pkg/security/opa/policies",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer evaluator.Close(ctx)
    
    // RBAC evaluation
    input := &opa.RBACInput{
        Subject: &opa.Subject{
            User:  "alice",
            Roles: []string{"editor"},
        },
        Action:   "update",
        Resource: "documents",
    }
    
    result, err := evaluator.EvaluateRBAC(ctx, input)
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Allowed {
        log.Println("✅ Access granted")
    } else {
        log.Println("❌ Access denied")
    }
}
```

### Advanced RBAC Features

#### 1. Resource Owner Permissions

```rego
# Users can access their own resources
allow if {
    is_owner
    input.action in ["read", "update", "delete"]
}

is_owner if {
    input.resource == sprintf("users/%s", [input.subject.user])
}
```

#### 2. Group-Based Role Inheritance

```rego
# Inherit roles through user groups
user_roles contains role if {
    some group in input.subject.groups
    some role in group_roles[group]
}

group_roles := {
    "engineering": ["editor", "contributor"],
    "management": ["admin"],
    "support": ["viewer"],
}
```

#### 3. Temporary Permissions

```rego
# Support temporary permissions
allow if {
    has_temporary_permission
}

has_temporary_permission if {
    some temp_perm in input.context.temporary_permissions
    temp_perm.action == input.action
    temp_perm.resource == input.resource
    temp_perm.expires_at > time.now_ns()
}
```

### Complete Example

See the RBAC implementation in the project:
- Policy file: `pkg/security/opa/policies/rbac.rego`
- Test file: `pkg/security/opa/policies/rbac_test.rego`
- Usage guide: `docs/opa-rbac-guide.md`

## ABAC Policy Examples

Attribute-Based Access Control (ABAC) provides finer-grained control than RBAC.

### Basic ABAC Policy

```rego
package abac

import rego.v1

# Default deny
default allow := false

# Main decision rule
allow if {
    subject_attributes_valid
    resource_attributes_valid
    action_attributes_valid
    environment_attributes_valid
}

# Super admin bypasses all checks
allow if {
    "admin" in input.subject.roles
}

# Subject attribute validation
subject_attributes_valid if {
    input.subject.user
    count(input.subject.roles) > 0
}

# Resource attribute validation
resource_attributes_valid if {
    is_string(input.resource)
}

resource_attributes_valid if {
    input.resource.type
    input.resource.id
}

# Action attribute validation
action_attributes_valid if {
    input.action in allowed_actions
}

allowed_actions := [
    "create", "read", "update", "delete", "list",
    "approve", "reject", "export", "share"
]

# Environment attribute validation
environment_attributes_valid if {
    not input.environment
}

environment_attributes_valid if {
    input.environment
    ip_address_valid
    time_valid
    device_type_valid
}
```

### ABAC Access Control Rules

#### 1. Department-Based Access Control

```rego
allow if {
    input.subject.attributes.department == input.resource.attributes.department
    input.action == "read"
}
```

#### 2. Security Level-Based Access Control

```rego
allow if {
    user_clearance_level >= resource_security_level
    input.action in ["read", "list"]
}

user_clearance_level := object.get(
    input.subject.attributes,
    "clearance_level",
    0
)

resource_security_level := object.get(
    input.resource.attributes,
    "security_level",
    0
)
```

#### 3. Time-Based Access Control

```rego
# Business hours restriction
is_business_hours if {
    input.environment.time
    current_time := input.environment.time
    # Monday to Friday
    current_time.weekday >= 1
    current_time.weekday <= 5
    # 9:00-18:00
    current_time.hour >= 9
    current_time.hour < 18
}

allow if {
    is_business_hours
    "employee" in input.subject.roles
    input.action in ["read", "list"]
}
```

#### 4. Location-Based Access Control

```rego
allow if {
    location_allowed
    input.resource.attributes.sensitivity == "high"
    input.action in ["read", "list"]
}

location_allowed if {
    input.environment.location in ["office", "vpn", "headquarters"]
}
```

#### 5. Dynamic Attribute Scoring

```rego
# Dynamic scoring based on multiple dimensions
allow if {
    dynamic_attribute_score >= required_score
}

dynamic_attribute_score := score if {
    score := role_score + 
             department_match_score + 
             clearance_score + 
             time_score + 
             location_score + 
             device_score
}

required_score := 60  # Default requirement: 60 points

# Role score
role_score := 30 if {
    "admin" in input.subject.roles
} else := 20 if {
    "manager" in input.subject.roles
} else := 10 if {
    "employee" in input.subject.roles
} else := 0

# Department match score
department_match_score := 20 if {
    input.subject.attributes.department == input.resource.attributes.department
} else := 0

# Security level score
clearance_score := 30 if {
    user_clearance_level >= resource_security_level
} else := 0
```

### Using ABAC Policy

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    ctx := context.Background()
    
    evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./pkg/security/opa/policies",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer evaluator.Close(ctx)
    
    // ABAC evaluation
    input := &opa.ABACInput{
        Subject: &opa.Subject{
            User:  "alice",
            Roles: []string{"employee"},
            Attributes: map[string]interface{}{
                "department":      "engineering",
                "clearance_level": 3,
            },
        },
        Action: "read",
        Resource: &opa.Resource{
            Type: "document",
            ID:   "doc-123",
            Attributes: map[string]interface{}{
                "department":     "engineering",
                "security_level": 2,
            },
        },
        Environment: &opa.Environment{
            Time: map[string]interface{}{
                "weekday": 3,
                "hour":    14,
            },
            Location:   "office",
            IPAddress:  "10.0.1.100",
            DeviceType: "laptop",
        },
    }
    
    result, err := evaluator.EvaluateABAC(ctx, input)
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Allowed {
        log.Printf("✅ Access granted (Score: %d/%d)", 
            result.Score, result.RequiredScore)
    } else {
        log.Println("❌ Access denied")
    }
}
```

### Complete Example

See the ABAC implementation in the project:
- Policy file: `pkg/security/opa/policies/abac.rego`
- Test file: `pkg/security/opa/policies/abac_test.rego`
- Usage guide: `docs/opa-abac-guide.md`
- Industry examples: `pkg/security/opa/policies/examples/`

## Policy Testing

Testing is a crucial part of policy development to ensure policies work as expected.

### Test File Structure

```rego
package rbac_test

import rego.v1
import data.rbac

# Test success scenarios
test_admin_has_all_permissions if {
    rbac.allow with input as {
        "subject": {"user": "alice", "roles": ["admin"]},
        "action": "delete",
        "resource": "users"
    }
}

# Test failure scenarios
test_viewer_cannot_delete if {
    not rbac.allow with input as {
        "subject": {"user": "bob", "roles": ["viewer"]},
        "action": "delete",
        "resource": "documents"
    }
}

# Test edge cases
test_empty_roles_denied if {
    not rbac.allow with input as {
        "subject": {"user": "charlie", "roles": []},
        "action": "read",
        "resource": "documents"
    }
}

# Test helper rules
test_user_roles if {
    rbac.user_roles == {"editor", "viewer"} with input as {
        "subject": {"user": "dave", "roles": ["editor", "viewer"]}
    }
}
```

### Running Tests

#### Using OPA CLI

```bash
# Run all tests
opa test pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego -v

# Run specific test
opa test pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego \
  -v -r test_admin_has_all_permissions

# Generate coverage report
opa test --coverage \
  pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego
```

#### Using Go Tests

```go
package opa_test

import (
    "context"
    "testing"
    
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRBACPolicy(t *testing.T) {
    ctx := context.Background()
    
    evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
    })
    require.NoError(t, err)
    defer evaluator.Close(ctx)
    
    tests := []struct {
        name     string
        input    *opa.RBACInput
        expected bool
    }{
        {
            name: "admin can delete users",
            input: &opa.RBACInput{
                Subject:  &opa.Subject{User: "alice", Roles: []string{"admin"}},
                Action:   "delete",
                Resource: "users",
            },
            expected: true,
        },
        {
            name: "viewer cannot delete documents",
            input: &opa.RBACInput{
                Subject:  &opa.Subject{User: "bob", Roles: []string{"viewer"}},
                Action:   "delete",
                Resource: "documents",
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := evaluator.EvaluateRBAC(ctx, tt.input)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.Allowed)
        })
    }
}
```

### Testing Best Practices

#### 1. Comprehensive Test Coverage

```rego
# Test all main rules
test_rule_1 if { ... }
test_rule_2 if { ... }

# Test boundary conditions
test_empty_input if { ... }
test_missing_fields if { ... }
test_invalid_data if { ... }

# Test negative scenarios
test_deny_unauthorized if { ... }
test_deny_insufficient_permissions if { ... }
```

#### 2. Table-Driven Tests

```rego
test_role_permissions if {
    test_cases := [
        {"role": "admin", "action": "delete", "expected": true},
        {"role": "editor", "action": "delete", "expected": false},
        {"role": "viewer", "action": "read", "expected": true},
        {"role": "viewer", "action": "update", "expected": false},
    ]
    
    every test_case in test_cases {
        result := rbac.allow with input as {
            "subject": {"user": "test", "roles": [test_case.role]},
            "action": test_case.action,
            "resource": "documents"
        }
        result == test_case.expected
    }
}
```

#### 3. Test Helper Functions

```rego
# Create test helpers
make_input(user, roles, action, resource) := {
    "subject": {"user": user, "roles": roles},
    "action": action,
    "resource": resource
}

# Use in tests
test_admin_access if {
    rbac.allow with input as make_input("alice", ["admin"], "delete", "users")
}
```

#### 4. Performance Benchmarking

```bash
# Run benchmarks
opa test --bench \
  pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego

# Example output:
# PASS: 15/15
# Benchmark results:
# test_admin_access: 0.05ms
# test_viewer_access: 0.03ms
```

### Test Coverage Goals

- **Target**: > 90% coverage
- Test all main decision rules
- Test all helper rules
- Test boundary conditions and error cases
- Test performance-critical paths

## Performance Optimization

OPA policy performance is critical for production environments. Here are some optimization techniques.

### 1. Decision Caching

Enable decision caching for significant performance improvements:

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,           // Cache up to 10000 decisions
        TTL:           5 * time.Minute, // 5-minute cache validity
        EnableMetrics: true,            // Enable cache metrics
    },
}
```

### 2. Policy Optimization

#### Early Exit

```rego
# ✅ Recommended: Fast path first
allow if {
    "admin" in input.subject.roles  # Quick check
}

allow if {
    complex_evaluation  # Complex evaluation later
}

# ❌ Avoid: Complex evaluation first
allow if {
    score := calculate_score  # Always computed
    score >= 60
}
```

#### Avoid Unnecessary Iteration

```rego
# ✅ Recommended: Use set operations
allow if {
    some role in {"admin", "editor"}
    role in input.subject.roles
}

# ❌ Avoid: Iterate entire list
allow if {
    some role in input.subject.roles
    role in ["admin", "editor", "viewer", "contributor", ...]
}
```

#### Index Optimization

```rego
# ✅ Recommended: Use object as index
role_permissions := {
    "admin": {...},
    "editor": {...},
}

allow if {
    some role in input.subject.roles
    role_permissions[role][input.action]  # O(1) lookup
}

# ❌ Avoid: Use array iteration
role_permissions := [
    {"role": "admin", "permissions": {...}},
    {"role": "editor", "permissions": {...}},
]

allow if {
    some perm in role_permissions  # O(n) iteration
    perm.role in input.subject.roles
    perm.permissions[input.action]
}
```

### 3. Data Optimization

#### Preload Static Data

```rego
# Define static data in policy
role_permissions := {
    "admin": {"read": true, "write": true, "delete": true},
    "editor": {"read": true, "write": true},
}

# Or load from external file
import data.permissions.role_permissions
```

#### Limit Data Size

```rego
# ✅ Recommended: Load only needed data
user_roles := input.subject.roles

# ❌ Avoid: Load unnecessary data
all_users := data.users  # May contain thousands of users
```

### 4. Performance Monitoring

#### Enable Performance Metrics

```go
config := &opa.Config{
    MetricsConfig: &opa.MetricsConfig{
        Enabled:     true,
        Port:        9090,
        Path:        "/metrics",
        EnableCache: true,
    },
}
```

#### Use Profiling

```bash
# Analyze policy performance
opa eval -d policies/rbac.rego \
  --profile \
  --input input.json \
  "data.rbac.allow"

# Output includes:
# - Execution time per rule
# - Call count
# - Total execution time
```

### 5. Benchmarking

```bash
# Run benchmarks
opa test --bench policies/

# Compare different versions
opa test --bench policies/ > benchmark-v1.txt
# After policy changes
opa test --bench policies/ > benchmark-v2.txt
diff benchmark-v1.txt benchmark-v2.txt
```

### 6. Production Configuration

```go
// Recommended production configuration
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    // Enable caching
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       50000,  // Adjust based on requirements
        TTL:           10 * time.Minute,
        EnableMetrics: true,
    },
    // Performance monitoring
    MetricsConfig: &opa.MetricsConfig{
        Enabled:     true,
        Port:        9090,
        EnableCache: true,
    },
    // Log level
    LogLevel: "info",  // Use info or warn in production
}
```

### Performance Benchmarks

Based on project tests, here are some performance references:

| Scenario | Avg Response Time | QPS | Cache Hit Rate |
|----------|------------------|-----|----------------|
| RBAC Simple Check | < 1ms | > 10,000 | 95% |
| RBAC Complex Check | < 3ms | > 5,000 | 90% |
| ABAC Simple Check | < 2ms | > 8,000 | 90% |
| ABAC Complex Scoring | < 5ms | > 3,000 | 85% |

## FAQ

### 1. OPA vs Hard-coded Permission Checks

**Q**: Why use OPA instead of checking permissions directly in code?

**A**: OPA provides several advantages:
- **Policy-code decoupling**: Update policies without changing code
- **Centralized management**: Define and manage all policies in one place
- **Testability**: Test policies independently
- **Auditability**: Complete audit trail for policy changes
- **Reusability**: Share policies across multiple services

### 2. RBAC vs ABAC

**Q**: Should I use RBAC or ABAC?

**A**: It depends on your requirements:

| Scenario | Recommendation |
|----------|---------------|
| Clear org structure, stable roles | RBAC |
| Fine-grained control based on multiple conditions | ABAC |
| Simple permission management | RBAC |
| Dynamic, context-aware access control | ABAC |
| Quick implementation | RBAC |
| Flexibility and extensibility | ABAC |

You can also combine them: use RBAC for coarse-grained control and ABAC for fine-grained control.

### 3. Embedded vs Server Mode

**Q**: Should I use embedded mode or standalone OPA server?

**A**: 
- **Embedded mode** (recommended for monoliths):
  - Pros: Low latency, simple deployment, no network overhead
  - Cons: Policy updates require app restart
  
- **Server mode** (recommended for microservices):
  - Pros: Dynamic policy updates, cross-service sharing, centralized management
  - Cons: Network latency, additional deployment complexity

### 4. Policy Updates

**Q**: How do I update policies in production?

**A**: Several approaches:

1. **Embedded mode**:
```go
// Reload policies
evaluator.ReloadPolicies(ctx)
```

2. **Server mode**:
```bash
# Update via API
curl -X PUT http://opa-server:8181/v1/policies/rbac \
  --data-binary @rbac.rego
```

3. **Bundle mode**:
```yaml
# Configure OPA to load policy bundles from remote location
services:
  - name: bundle-server
    url: https://policies.example.com

bundles:
  - name: authz
    service: bundle-server
    resource: bundles/authz.tar.gz
```

### 5. Performance Optimization

**Q**: What if OPA policy evaluation is too slow?

**A**: Try these optimizations:
1. Enable decision caching
2. Optimize policy rules (early exit, avoid unnecessary computation)
3. Use index optimization (objects instead of arrays)
4. Reduce data loading
5. Use profiling tools to find bottlenecks

See [Performance Optimization](#performance-optimization) section.

### 6. Debugging Policies

**Q**: How do I debug OPA policies?

**A**: 
1. **Use OPA REPL**:
```bash
opa run policies/rbac.rego

# Test in REPL
> import data.rbac
> rbac.allow with input as {"subject": {...}, "action": "read"}
```

2. **Add debug output**:
```rego
debug_info := {
    "user_roles": user_roles,
    "permissions": user_permissions,
    "decision": allow,
}
```

3. **Enable verbose logging**:
```go
config := &opa.Config{
    LogLevel: "debug",
}
```

### 7. Error Handling

**Q**: How do I handle policy evaluation errors?

**A**: 
```go
result, err := evaluator.EvaluateRBAC(ctx, input)
if err != nil {
    // Log error
    log.Error("Policy evaluation failed", zap.Error(err))
    
    // Fallback strategy: default deny
    return false, err
}

if !result.Allowed {
    // Log denial reason
    log.Info("Access denied",
        zap.String("user", input.Subject.User),
        zap.String("action", input.Action))
}

return result.Allowed, nil
```

### 8. Test Coverage

**Q**: How do I ensure sufficient test coverage for policies?

**A**: 
```bash
# Generate coverage report
opa test --coverage policies/

# Target: > 90% coverage
# Ensure testing:
# - All main decision rules
# - All helper rules
# - Boundary conditions
# - Error cases
```

### 9. Policy Versioning

**Q**: How do I manage policy versions?

**A**: 
1. Put policy files in Git version control
2. Include version metadata in policies
3. Use Git tags to mark policy versions
4. Review policy changes in PRs
5. Use CI/CD to automatically test policies

```bash
git add policies/rbac.rego
git commit -m "feat(rbac): add new role for auditors"
git tag -a policy-v1.2.0 -m "RBAC Policy v1.2.0"
```

### 10. Integrating into Existing Systems

**Q**: How do I integrate OPA into existing systems?

**A**: 
1. **Identify authorization points**: Find where authorization checks are needed
2. **Define policies**: Convert existing authorization logic to Rego policies
3. **Gradual migration**: Start with non-critical paths, gradually expand
4. **Parallel operation**: Run old and new authorization logic side-by-side, compare results
5. **Monitor and tune**: Monitor performance and correctness, adjust as needed

```go
// Example: Integrate OPA in HTTP middleware
func AuthorizationMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Build OPA input
        input := &opa.RBACInput{
            Subject: &opa.Subject{
                User:  c.GetString("user"),
                Roles: c.GetStringSlice("roles"),
            },
            Action:   c.Request.Method,
            Resource: c.Request.URL.Path,
        }
        
        // Evaluate policy
        result, err := evaluator.EvaluateRBAC(c.Request.Context(), input)
        if err != nil {
            c.AbortWithStatus(http.StatusInternalServerError)
            return
        }
        
        if !result.Allowed {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        
        c.Next()
    }
}
```

## Related Resources

### Official Documentation
- [OPA Official Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA Policy Testing](https://www.openpolicyagent.org/docs/latest/policy-testing/)
- [OPA Performance](https://www.openpolicyagent.org/docs/latest/policy-performance/)

### Project Documentation
- [pkg/security/opa/README.md](../pkg/security/opa/README.md) - OPA package documentation
- [docs/opa-rbac-guide.md](./opa-rbac-guide.md) - RBAC policy usage guide
- [docs/opa-abac-guide.md](./opa-abac-guide.md) - ABAC policy usage guide
- [docs/api/security-opa.md](./api/security-opa.md) - OPA API reference

### Example Code
- [pkg/security/opa/policies/rbac.rego](../pkg/security/opa/policies/rbac.rego) - RBAC implementation
- [pkg/security/opa/policies/abac.rego](../pkg/security/opa/policies/abac.rego) - ABAC implementation
- [pkg/security/opa/policies/examples/](../pkg/security/opa/policies/examples/) - Industry examples

### Related Issues
- [Epic #776: OPA Policy Engine Integration](https://github.com/innovationmech/swit/issues/776)
- [Task #799: RBAC Policy Template](https://github.com/innovationmech/swit/issues/799)
- [Task #800: ABAC Policy Template](https://github.com/innovationmech/swit/issues/800)
- [Task #815: API Documentation](https://github.com/innovationmech/swit/issues/815)

## License

Copyright (c) 2024 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

