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

# 示例 ABAC 策略 - 基于属性的文档访问控制

package abac

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 主决策规则
# ===================================

# 规则 1: 管理员绕过所有检查
allow if {
    "admin" in input.user.roles
}

# 规则 2: 基于时间和 IP 的访问控制
allow if {
    # 用户必须有有效角色
    count(input.user.roles) > 0
    # 必须是工作时间 (假设 Unix 时间戳)
    is_business_hours
    # IP 地址必须在允许的范围内
    is_allowed_ip
    # 操作必须被允许
    is_allowed_action
}

# 规则 3: 资源所有者访问
allow if {
    input.resource.owner == input.user.username
    is_allowed_action
}

# ===================================
# 辅助规则
# ===================================

# 检查是否是工作时间 (9:00 - 18:00)
is_business_hours if {
    not input.request.time
}

is_business_hours if {
    input.request.time
    # 简化版本：实际应用中应该检查时区和具体小时
    # 这里仅做示范
    hour := time.clock([input.request.time, "UTC"])[0]
    hour >= 9
    hour < 18
}

# 检查 IP 地址是否在允许的范围内
is_allowed_ip if {
    not input.request.client_ip
}

is_allowed_ip if {
    input.request.client_ip
    # 允许内网 IP
    startswith(input.request.client_ip, "192.168.")
}

is_allowed_ip if {
    input.request.client_ip
    # 允许本地 IP
    startswith(input.request.client_ip, "127.0.")
}

is_allowed_ip if {
    input.request.client_ip
    # 允许 Docker 网络
    startswith(input.request.client_ip, "172.")
}

# 检查操作是否被允许
is_allowed_action if {
    input.request.method in ["GET", "POST", "PUT", "DELETE"]
}

# ===================================
# 资源级别的细粒度控制
# ===================================

# 编辑者只能在工作时间修改文档
allow if {
    "editor" in input.user.roles
    input.request.method in ["POST", "PUT"]
    input.resource.type == "document"
    is_business_hours
    is_allowed_ip
}

# 查看者可以随时查看文档（不受时间限制）
allow if {
    "viewer" in input.user.roles
    input.request.method == "GET"
    input.resource.type == "document"
    is_allowed_ip
}

# ===================================
# 审计和决策原因
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
    not allow
    msg := "Access denied: Policy conditions not met"
}
