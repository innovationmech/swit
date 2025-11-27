# OPA Authorization Policy for Integration Testing
# This policy implements RBAC with role hierarchy

package authz

import rego.v1

default allow := false

# Role hierarchy definition
role_hierarchy := {
    "super_admin": ["admin", "editor", "viewer"],
    "admin": ["editor", "viewer"],
    "editor": ["viewer"],
    "viewer": [],
}

# Get effective roles (including inherited roles)
effective_roles contains role if {
    some r in input.subject.roles
    role := role_hierarchy[r][_]
}

effective_roles contains role if {
    some role in input.subject.roles
}

# Resource permissions
resource_permissions := {
    "documents": {
        "admin": ["create", "read", "update", "delete"],
        "editor": ["create", "read", "update"],
        "viewer": ["read"],
    },
    "users": {
        "admin": ["create", "read", "update", "delete"],
        "viewer": ["read"],
    },
    "settings": {
        "super_admin": ["create", "read", "update", "delete"],
        "admin": ["read", "update"],
    },
}

# Allow if user has required permission through effective roles
allow if {
    some role in effective_roles
    input.action in resource_permissions[input.resource][role]
}

# Super admin bypass
allow if {
    "super_admin" in input.subject.roles
}

