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

# ABAC (Attribute-Based Access Control) 策略模板
# 基于属性的访问控制策略

package abac

import rego.v1

# 默认拒绝所有访问
default allow := false

# 主决策规则：综合评估所有属性
allow if {
    # 主体属性检查
    subject_attributes_valid
    # 资源属性检查
    resource_attributes_valid
    # 操作属性检查
    action_attributes_valid
    # 环境属性检查
    environment_attributes_valid
}

# 主体属性验证
subject_attributes_valid if {
    # 用户必须存在
    input.subject.user
    # 用户必须有有效的角色或属性
    count(input.subject.roles) > 0
}

# 资源属性验证
resource_attributes_valid if {
    # 资源类型必须存在
    input.resource.type
    # 资源 ID 必须存在
    input.resource.id
}

# 操作属性验证
action_attributes_valid if {
    # 操作必须是允许的操作之一
    input.action in allowed_actions
}

# 环境属性验证
environment_attributes_valid if {
    # 如果没有环境限制，默认通过
    not input.environment
}

environment_attributes_valid if {
    # 如果有环境限制，进行验证
    input.environment
    # IP 地址检查
    ip_address_valid
    # 时间检查
    time_valid
}

# 允许的操作列表
allowed_actions := ["create", "read", "update", "delete", "list", "execute"]

# 基于资源所有者的访问控制
allow if {
    input.resource.owner == input.subject.user
    input.action in ["read", "update", "delete"]
}

# 基于部门的访问控制
allow if {
    # 同一部门的用户可以读取资源
    input.subject.attributes.department == input.resource.attributes.department
    input.action == "read"
}

# 基于安全级别的访问控制
allow if {
    # 用户的安全级别必须高于或等于资源的安全级别
    user_clearance_level >= resource_security_level
    input.action in ["read", "list"]
}

# 获取用户的安全级别
user_clearance_level := level if {
    level := input.subject.attributes.clearance_level
}

user_clearance_level := 0 if {
    not input.subject.attributes.clearance_level
}

# 获取资源的安全级别
resource_security_level := level if {
    level := input.resource.attributes.security_level
}

resource_security_level := 0 if {
    not input.resource.attributes.security_level
}

# 基于项目成员的访问控制
allow if {
    # 用户必须是项目成员
    input.subject.user in project_members
    # 项目成员可以读取和更新项目资源
    input.action in ["read", "update", "list"]
    input.resource.type == "project"
}

# 项目成员列表（应从外部数据源加载）
project_members contains member if {
    some member in input.resource.attributes.members
}

# 基于时间的访问控制
allow if {
    # 只在工作时间允许访问
    is_business_hours
    # 用户角色检查
    "employee" in input.subject.roles
    input.action in ["read", "list"]
}

# 判断是否在工作时间
is_business_hours if {
    # 检查时间属性是否存在
    input.environment.time
    # 这里应该实现实际的时间范围检查
    # 示例：周一到周五 9:00-18:00
    true
}

# 基于地理位置的访问控制
allow if {
    # 只允许从特定位置访问敏感资源
    location_allowed
    input.resource.attributes.sensitivity == "high"
    input.action in ["read", "list"]
}

# 检查地理位置是否允许
location_allowed if {
    input.environment.location in allowed_locations
}

# 允许的地理位置列表
allowed_locations := ["office", "vpn", "headquarters"]

# 基于 IP 地址的访问控制
ip_address_valid if {
    # 如果没有 IP 限制，默认通过
    not input.environment.ip_address
}

ip_address_valid if {
    # 检查 IP 是否在允许的范围内
    input.environment.ip_address
    is_ip_allowed(input.environment.ip_address)
}

# 检查 IP 是否允许（简化实现）
is_ip_allowed(ip) if {
    # 这里应该实现实际的 IP 范围检查
    # 示例：允许内网 IP
    startswith(ip, "10.")
}

is_ip_allowed(ip) if {
    startswith(ip, "192.168.")
}

# 时间有效性检查
time_valid if {
    # 如果没有时间限制，默认通过
    not input.environment.time
}

time_valid if {
    # 这里应该实现实际的时间范围检查
    input.environment.time
    true
}

# 基于设备类型的访问控制
allow if {
    # 某些操作只能从特定设备执行
    device_allowed
    input.action in ["read", "list"]
}

# 检查设备是否允许
device_allowed if {
    not input.environment.device_type
}

device_allowed if {
    input.environment.device_type in allowed_devices
}

# 允许的设备类型
allowed_devices := ["desktop", "laptop", "mobile", "tablet"]

# 基于数据分类的访问控制
allow if {
    # 根据数据分类级别决定访问权限
    data_classification_check
    input.action in ["read", "list"]
}

# 数据分类检查
data_classification_check if {
    # 公开数据任何人都可以访问
    input.resource.attributes.classification == "public"
}

data_classification_check if {
    # 内部数据只有员工可以访问
    input.resource.attributes.classification == "internal"
    "employee" in input.subject.roles
}

data_classification_check if {
    # 机密数据只有授权用户可以访问
    input.resource.attributes.classification == "confidential"
    "authorized" in input.subject.attributes.access_levels
}

# 动态属性评估：根据运行时属性组合决定访问权限
allow if {
    # 评估多个条件的组合
    dynamic_attribute_score >= required_score
}

# 计算动态属性分数
dynamic_attribute_score := score if {
    score := sum([
        role_score,
        department_score,
        clearance_score,
        time_score,
        location_score,
    ])
}

# 角色分数
role_score := 30 if {
    "admin" in input.subject.roles
}

role_score := 20 if {
    "manager" in input.subject.roles
}

role_score := 10 if {
    "employee" in input.subject.roles
}

role_score := 0 if {
    not input.subject.roles
}

# 部门分数
department_score := 20 if {
    input.subject.attributes.department == input.resource.attributes.department
}

department_score := 0 if {
    not input.subject.attributes.department
}

# 安全级别分数
clearance_score := 30 if {
    user_clearance_level >= resource_security_level
}

clearance_score := 0 if {
    user_clearance_level < resource_security_level
}

# 时间分数
time_score := 10 if {
    is_business_hours
}

time_score := 0 if {
    not is_business_hours
}

# 地理位置分数
location_score := 10 if {
    location_allowed
}

location_score := 0 if {
    not location_allowed
}

# 所需的最低分数（可配置）
required_score := 60

# 审计日志
audit_log := {
    "user": input.subject.user,
    "action": input.action,
    "resource": input.resource,
    "environment": input.environment,
    "decision": allow,
    "score": dynamic_attribute_score,
    "timestamp": time.now_ns(),
}

