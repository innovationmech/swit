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

# 示例 RBAC 策略 - 文档管理系统

package rbac

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 角色权限定义
# ===================================

# 管理员拥有所有权限
allow if {
    "admin" in input.subject.roles
}

# 编辑者可以创建、读取、更新文档
allow if {
    "editor" in input.subject.roles
    input.action in ["GET", "POST", "PUT"]
    startswith(input.resource.path, "/api/v1/documents")
}

# 查看者只能读取文档
allow if {
    "viewer" in input.subject.roles
    input.action == "GET"
    startswith(input.resource.path, "/api/v1/documents")
}

# ===================================
# 资源所有者权限
# ===================================

# 文档所有者可以对自己的文档执行任何操作
allow if {
    input.resource.type == "document"
    input.resource.owner == input.subject.user
}

# ===================================
# 公共资源访问
# ===================================

# 所有人都可以访问健康检查端点
allow if {
    input.resource.path == "/api/v1/health"
}

# ===================================
# 审计和决策原因
# ===================================

# 决策原因（用于审计日志）
reason contains msg if {
    "admin" in input.subject.roles
    msg := "Access granted: User has admin role"
}

reason contains msg if {
    "editor" in input.subject.roles
    input.action in ["GET", "POST", "PUT"]
    msg := "Access granted: Editor can read, create, and update documents"
}

reason contains msg if {
    "viewer" in input.subject.roles
    input.action == "GET"
    msg := "Access granted: Viewer can read documents"
}

reason contains msg if {
    not allow
    msg := "Access denied: User lacks required permissions"
}

