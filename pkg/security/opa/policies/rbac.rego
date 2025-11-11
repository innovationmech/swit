# Copyright (c) 2024 Six-Thirty Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# RBAC (Role-Based Access Control) 策略模板
# 基于角色的访问控制策略

package rbac

import rego.v1

# 默认拒绝所有访问
default allow := false

# 主决策规则
allow if {
    # 检查用户是否具有执行该操作的角色
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}

# 超级管理员拥有所有权限
allow if {
    "admin" in user_roles
}

# 获取用户角色
user_roles contains role if {
    some role in input.subject.roles
}

# 角色权限映射（可从外部数据源加载）
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
    "contributor": {
        "create": true,
        "read": true,
        "list": true,
    },
}

# 角色资源映射（定义角色可以访问的资源类型）
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
    "contributor": {
        "documents": true,
    },
}

# 获取用户的所有权限
user_permissions contains permission if {
    some role in user_roles
    some permission in object.keys(role_permissions[role])
    role_permissions[role][permission]
}

# 检查用户是否有特定权限
has_permission(permission) if {
    permission in user_permissions
}

# 获取用户可访问的资源类型
user_resources contains resource if {
    some role in user_roles
    some resource in object.keys(role_resources[role])
    role_resources[role][resource]
}

# 检查用户是否可以访问特定资源类型
can_access_resource(resource) if {
    resource in user_resources
}

# 基于用户组的角色继承
allow if {
    # 如果用户属于某个组，继承该组的角色权限
    some group in input.subject.groups
    some role in group_roles[group]
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}

# 组角色映射（示例）
group_roles := {
    "engineering": ["editor", "contributor"],
    "management": ["admin"],
    "support": ["viewer"],
    "sales": ["viewer", "contributor"],
}

# 审计日志：记录所有决策（可通过 OPA 决策日志功能实现）
audit_log := {
    "user": input.subject.user,
    "action": input.action,
    "resource": input.resource,
    "roles": user_roles,
    "decision": allow,
    "timestamp": time.now_ns(),
}

# 检查是否为资源所有者
is_owner if {
    input.resource == sprintf("users/%s", [input.subject.user])
}

# 资源所有者拥有完全访问权限
allow if {
    is_owner
    input.action in ["read", "update", "delete"]
}

# 临时权限检查（基于上下文）
allow if {
    # 检查是否在工作时间
    is_business_hours
    # 检查临时权限
    has_temporary_permission
}

# 判断是否在工作时间（示例：周一到周五 9:00-18:00）
is_business_hours if {
    # 这里应该使用实际的时间检查
    # 示例实现
    input.context.time
}

# 检查临时权限
has_temporary_permission if {
    some temp_perm in input.context.temporary_permissions
    temp_perm.action == input.action
    temp_perm.resource == input.resource
    temp_perm.expires_at > time.now_ns()
}

