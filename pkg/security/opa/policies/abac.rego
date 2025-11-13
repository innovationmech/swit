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
# 
# 特性：
# - 主体属性匹配（用户、角色、部门、安全级别）
# - 资源属性匹配（类型、所有者、分类、敏感性级别）
# - 环境约束（时间、地理位置、IP 地址、设备类型）
# - 动态属性评分机制
# - 审计日志支持

package abac

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 主决策规则
# ===================================

# 规则 1: 综合属性评估（完整 ABAC）
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

# 规则 2: 超级管理员绕过所有检查
allow if {
    "admin" in input.subject.roles
}

# ===================================
# 属性验证规则
# ===================================

# 主体属性验证
subject_attributes_valid if {
    # 用户必须存在
    input.subject.user
    # 用户必须有有效的角色或属性
    count(input.subject.roles) > 0
}

# 资源属性验证（简化版）
resource_attributes_valid if {
    # 简化：如果资源只是字符串，验证通过
    is_string(input.resource)
}

# 资源属性验证（完整版）
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

# 环境属性验证：无环境限制时默认通过
environment_attributes_valid if {
    not input.environment
}

# 环境属性验证：有环境限制时检查各项
environment_attributes_valid if {
    input.environment
    # IP 地址检查
    ip_address_valid
    # 时间检查
    time_valid
    # 设备类型检查
    device_type_valid
}

# 允许的操作列表
allowed_actions := ["create", "read", "update", "delete", "list", "execute", "share", "export"]

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

# ===================================
# 基于敏感性级别的访问控制
# ===================================

# 规则：用户的安全级别必须高于或等于资源的安全级别
allow if {
    user_clearance_level >= resource_security_level
    input.action in ["read", "list"]
}

# 获取用户的安全级别（clearance level）
# 级别定义：0=无, 1=公开, 2=内部, 3=机密, 4=绝密
user_clearance_level := level if {
    level := input.subject.attributes.clearance_level
}

user_clearance_level := 0 if {
    not input.subject.attributes
}

user_clearance_level := 0 if {
    input.subject.attributes
    not input.subject.attributes.clearance_level
}

# 获取资源的安全级别（security level）
# 级别定义：0=无, 1=公开, 2=内部, 3=机密, 4=绝密
resource_security_level := level if {
    is_object(input.resource)
    input.resource.attributes
    level := input.resource.attributes.security_level
}

resource_security_level := 0 if {
    is_string(input.resource)
}

resource_security_level := 0 if {
    is_object(input.resource)
    not input.resource.attributes
}

resource_security_level := 0 if {
    is_object(input.resource)
    input.resource.attributes
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

# ===================================
# 基于时间约束的访问控制
# ===================================

# 规则：只在工作时间允许访问
allow if {
    is_business_hours
    "employee" in input.subject.roles
    input.action in ["read", "list"]
}

# 规则：支持时间窗口限制
allow if {
    within_time_window
    input.action in ["read", "update"]
}

# 判断是否在工作时间（周一到周五 9:00-18:00）
is_business_hours if {
    # 如果没有时间信息，默认通过
    not input.environment
}

is_business_hours if {
    not input.environment.time
}

is_business_hours if {
    input.environment.time
    # 获取当前时间信息
    current_time := input.environment.time
    # 检查工作日（1=周一, 5=周五）
    current_time.weekday >= 1
    current_time.weekday <= 5
    # 检查工作时间（9:00-18:00）
    current_time.hour >= 9
    current_time.hour < 18
}

# 检查是否在允许的时间窗口内
within_time_window if {
    # 如果没有时间窗口限制，默认通过
    not input.environment
}

within_time_window if {
    not input.environment.time_window
}

within_time_window if {
    input.environment.time_window
    # 获取当前时间戳（纳秒）
    current := time.now_ns()
    # 检查是否在时间窗口内
    window := input.environment.time_window
    current >= window.start
    current <= window.end
}

# ===================================
# 基于地理位置约束的访问控制
# ===================================

# 规则：敏感资源只能从特定位置访问
allow if {
    location_allowed
    is_object(input.resource)
    input.resource.attributes.sensitivity == "high"
    input.action in ["read", "list"]
}

# 规则：某些操作必须从公司网络执行
allow if {
    is_corporate_network
    input.action in ["delete", "update"]
}

# 检查地理位置是否允许
location_allowed if {
    # 如果没有地理位置限制，默认通过
    not input.environment
}

location_allowed if {
    not input.environment.location
}

location_allowed if {
    input.environment.location in allowed_locations
}

# 允许的地理位置列表（可从外部数据加载）
allowed_locations := ["office", "vpn", "headquarters", "branch-office", "home-office"]

# 检查是否从公司网络访问
is_corporate_network if {
    location_allowed
}

is_corporate_network if {
    # 通过 IP 地址判断
    input.environment.ip_address
    is_ip_allowed(input.environment.ip_address)
}

# ===================================
# 基于 IP 地址的访问控制
# ===================================

# IP 地址验证
ip_address_valid if {
    # 如果没有 IP 限制，默认通过
    not input.environment
}

ip_address_valid if {
    not input.environment.ip_address
}

ip_address_valid if {
    # 检查 IP 是否在允许的范围内
    input.environment.ip_address
    is_ip_allowed(input.environment.ip_address)
}

# 检查 IP 是否允许（支持内网 IP 和特定 IP 段）
is_ip_allowed(ip) if {
    # 允许 10.x.x.x 内网段
    startswith(ip, "10.")
}

is_ip_allowed(ip) if {
    # 允许 192.168.x.x 内网段
    startswith(ip, "192.168.")
}

is_ip_allowed(ip) if {
    # 允许 172.16-31.x.x 内网段
    startswith(ip, "172.1")
    parts := split(ip, ".")
    second_octet := to_number(parts[1])
    second_octet >= 16
    second_octet <= 31
}

is_ip_allowed(ip) if {
    # 允许特定的公司 IP 段（示例）
    startswith(ip, "203.0.113.")
}

# 时间有效性检查（通用）
time_valid if {
    # 如果没有时间限制，默认通过
    not input.environment
}

time_valid if {
    not input.environment.time
}

time_valid if {
    # 有时间信息时，检查是否在有效时间范围内
    input.environment.time
    # 基本时间有效性检查（可扩展）
    true
}

# ===================================
# 基于设备类型的访问控制
# ===================================

# 规则：某些操作只能从特定设备执行
allow if {
    device_allowed
    input.action in ["read", "list"]
}

# 规则：敏感操作必须从受信任设备执行
allow if {
    is_trusted_device
    input.action in ["delete", "export"]
}

# 设备类型检查
device_type_valid if {
    not input.environment
}

device_type_valid if {
    not input.environment.device_type
}

device_type_valid if {
    input.environment.device_type in allowed_devices
}

# 检查设备是否允许
device_allowed if {
    not input.environment
}

device_allowed if {
    not input.environment.device_type
}

device_allowed if {
    input.environment.device_type in allowed_devices
}

# 检查是否为受信任设备
is_trusted_device if {
    input.environment.device_type in trusted_devices
}

is_trusted_device if {
    input.environment.device_trusted == true
}

# 允许的设备类型
allowed_devices := ["desktop", "laptop", "mobile", "tablet", "workstation"]

# 受信任的设备类型（用于敏感操作）
trusted_devices := ["desktop", "laptop", "workstation"]

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

# ===================================
# 动态属性评分机制
# ===================================

# 规则：基于综合评分的访问控制
# 当用户没有明确的角色权限时，可以通过累积分数来获得访问权限
# 注意：此规则仅在主体和资源属性都完整时才生效
allow if {
    # 必须有主体和资源属性
    input.subject.roles
    count(input.subject.roles) > 0
    is_object(input.resource)
    input.resource.attributes
    # 评估多个属性的组合分数
    dynamic_attribute_score >= required_score
    # 操作必须是读取相关的
    input.action in ["read", "list"]
}

# 计算动态属性分数（满分 100 分）
dynamic_attribute_score := score if {
    score := role_score + department_score + clearance_score + time_score + location_score + device_score
}

# 角色分数（最高 30 分）
role_score := 30 if {
    "admin" in input.subject.roles
}

role_score := 20 if {
    not "admin" in input.subject.roles
    "manager" in input.subject.roles
}

role_score := 10 if {
    not "admin" in input.subject.roles
    not "manager" in input.subject.roles
    "employee" in input.subject.roles
}

role_score := 0 if {
    not input.subject.roles
}

role_score := 0 if {
    input.subject.roles
    not "admin" in input.subject.roles
    not "manager" in input.subject.roles
    not "employee" in input.subject.roles
}

# 部门匹配分数（20 分）
default department_score := 0

department_score := 20 if {
    is_object(input.resource)
    input.subject.attributes
    input.resource.attributes
    input.subject.attributes.department == input.resource.attributes.department
}

# 安全级别分数（30 分）
default clearance_score := 0

clearance_score := 30 if {
    user_clearance_level >= resource_security_level
    resource_security_level > 0
}

clearance_score := 10 if {
    user_clearance_level >= resource_security_level
    resource_security_level == 0
}

# 时间分数（10 分）
default time_score := 0

time_score := 10 if {
    is_business_hours
}

time_score := 5 if {
    not is_business_hours
    within_time_window
}

# 地理位置分数（10 分）
default location_score := 0

location_score := 10 if {
    location_allowed
    is_corporate_network
}

location_score := 5 if {
    location_allowed
    not is_corporate_network
}

# 设备分数（最高 10 分）
default device_score := 0

device_score := 10 if {
    is_trusted_device
}

device_score := 5 if {
    not is_trusted_device
    device_allowed
}

# 所需的最低分数（可根据资源敏感性配置）
# 默认：60 分（满分 100）
default required_score := 60

required_score := score if {
    is_object(input.resource)
    input.resource.attributes
    input.resource.attributes.required_score
    score := input.resource.attributes.required_score
}

# ===================================
# 审计日志和辅助函数
# ===================================

# 审计日志：记录所有访问决策
audit_log := {
    "user": input.subject.user,
    "roles": input.subject.roles,
    "action": input.action,
    "resource": input.resource,
    "environment": input.environment,
    "decision": allow,
    "score": dynamic_attribute_score,
    "score_breakdown": {
        "role_score": role_score,
        "department_score": department_score,
        "clearance_score": clearance_score,
        "time_score": time_score,
        "location_score": location_score,
        "device_score": device_score,
    },
    "required_score": required_score,
    "clearance_level": user_clearance_level,
    "resource_security_level": resource_security_level,
    "timestamp": time.now_ns(),
}

# ===================================
# 辅助函数
# ===================================

# 检查用户是否有指定属性
has_attribute(attr_name) if {
    input.subject.attributes
    input.subject.attributes[attr_name]
}

# 检查资源是否有指定属性
resource_has_attribute(attr_name) if {
    is_object(input.resource)
    input.resource.attributes
    input.resource.attributes[attr_name]
}

# 获取用户属性值
get_user_attribute(attr_name) := value if {
    input.subject.attributes
    value := input.subject.attributes[attr_name]
}

# 获取资源属性值
get_resource_attribute(attr_name) := value if {
    is_object(input.resource)
    input.resource.attributes
    value := input.resource.attributes[attr_name]
}

# 检查用户是否在指定角色列表中
has_any_role(roles) if {
    some role in roles
    role in input.subject.roles
}

# 检查操作是否为写操作
is_write_operation if {
    input.action in ["create", "update", "delete"]
}

# 检查操作是否为读操作
is_read_operation if {
    input.action in ["read", "list"]
}

# 检查资源是否为敏感资源
is_sensitive_resource if {
    is_object(input.resource)
    input.resource.attributes
    input.resource.attributes.sensitivity in ["high", "critical"]
}

# 检查资源是否为高安全级别
is_high_security_resource if {
    resource_security_level >= 3
}

