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

# 多租户 SaaS 应用 ABAC 策略示例
# Multi-Tenant SaaS Application ABAC Policy Example
# 
# 场景：多租户 SaaS 平台的数据隔离和访问控制
# Scenario: Data isolation and access control for multi-tenant SaaS platforms

package saas.multi_tenant

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 规则 1: 租户数据隔离
# ===================================

# 用户只能访问自己租户的数据
allow if {
	# 用户和资源属于同一租户
	input.subject.tenant_id == input.resource.tenant_id
	# 用户在租户内有相应角色
	has_tenant_role
	# 执行允许的操作
	input.action in allowed_actions_for_role
}

# 检查租户内角色
has_tenant_role if {
	count(input.subject.roles) > 0
}

# 根据角色获取允许的操作
allowed_actions_for_role contains action if {
	"owner" in input.subject.roles
	action := ["create", "read", "update", "delete", "share", "manage"][_]
}

allowed_actions_for_role contains action if {
	"admin" in input.subject.roles
	action := ["create", "read", "update", "delete", "share"][_]
}

allowed_actions_for_role contains action if {
	"editor" in input.subject.roles
	action := ["create", "read", "update"][_]
}

allowed_actions_for_role contains action if {
	"viewer" in input.subject.roles
	action := ["read", "list"][_]
}

# ===================================
# 规则 2: 跨租户共享
# ===================================

# 允许已授权的跨租户访问
allow if {
	# 资源已被共享给用户的租户
	is_shared_with_tenant
	# 用户有访问共享资源的权限
	input.action in shared_access_permissions
}

# 检查资源是否共享给租户
is_shared_with_tenant if {
	some share in input.resource.shares
	share.tenant_id == input.subject.tenant_id
	share.active == true
	# 共享未过期
	share.expires_at > time.now_ns()
}

# 获取共享访问权限
shared_access_permissions contains permission if {
	some share in input.resource.shares
	share.tenant_id == input.subject.tenant_id
	some permission in share.permissions
}

# ===================================
# 规则 3: 基于订阅计划的功能限制
# ===================================

# 免费计划的限制
allow if {
	input.subject.subscription_plan == "free"
	# 免费计划用户数限制
	input.subject.tenant_user_count <= 5
	# 只能访问基础功能
	input.action in ["read", "create"]
	# 资源必须是基础类型
	input.resource.type in ["document", "task"]
}

# 专业计划的权限
allow if {
	input.subject.subscription_plan == "professional"
	# 用户数限制
	input.subject.tenant_user_count <= 50
	# 可以访问高级功能
	input.action in ["read", "create", "update", "delete", "share"]
	# 可以访问更多资源类型
	input.resource.type in ["document", "task", "project", "report"]
}

# 企业计划无限制
allow if {
	input.subject.subscription_plan == "enterprise"
	# 用户有相应权限
	has_tenant_role
	# 所有操作都允许
	input.action in allowed_actions_for_role
}

# ===================================
# 规则 4: API 配额限制
# ===================================

# 基于订阅计划的 API 配额检查
allow if {
	# 检查配额是否已用完
	not quota_exceeded
	# 其他访问条件满足
	input.subject.tenant_id == input.resource.tenant_id
	has_tenant_role
}

# 检查配额是否超限
quota_exceeded if {
	input.subject.subscription_plan == "free"
	input.subject.api_calls_today > 1000
}

quota_exceeded if {
	input.subject.subscription_plan == "professional"
	input.subject.api_calls_today > 100000
}

# 企业计划无配额限制
quota_exceeded if {
	input.subject.subscription_plan == "enterprise"
	false
}

# ===================================
# 规则 5: 基于资源所有者的访问
# ===================================

# 资源创建者拥有完全访问权限
allow if {
	input.resource.created_by == input.subject.user
	input.subject.tenant_id == input.resource.tenant_id
	input.action in ["read", "update", "delete", "share"]
}

# ===================================
# 规则 6: 基于工作空间的访问控制
# ===================================

# 用户只能访问自己所在工作空间的资源
allow if {
	# 用户是工作空间成员
	input.subject.user in workspace_members
	# 基于工作空间角色的权限
	workspace_role_allows_action
}

# 获取工作空间成员
workspace_members contains member if {
	input.resource.workspace_id
	some member in input.resource.workspace_members
}

# 检查工作空间角色权限
workspace_role_allows_action if {
	input.resource.workspace_role == "admin"
	input.action in ["create", "read", "update", "delete", "invite"]
}

workspace_role_allows_action if {
	input.resource.workspace_role == "member"
	input.action in ["create", "read", "update"]
}

workspace_role_allows_action if {
	input.resource.workspace_role == "guest"
	input.action in ["read"]
}

# ===================================
# 规则 7: 基于标签的访问控制
# ===================================

# 用户可以访问带有特定标签的资源
allow if {
	# 用户的访问标签包含资源标签
	resource_tags_match_user_access
	input.subject.tenant_id == input.resource.tenant_id
	input.action in ["read", "list"]
}

# 检查标签匹配
resource_tags_match_user_access if {
	some resource_tag in input.resource.tags
	some user_tag in input.subject.attributes.access_tags
	resource_tag == user_tag
}

# ===================================
# 规则 8: 基于时间的访问限制
# ===================================

# 试用期账户有时间限制
allow if {
	input.subject.subscription_plan == "trial"
	# 试用期未过期
	input.subject.trial_expires_at > time.now_ns()
	# 在租户内
	input.subject.tenant_id == input.resource.tenant_id
	# 基础操作
	input.action in ["read", "create"]
}

# ===================================
# 规则 9: 基于地理位置的数据驻留
# ===================================

# 确保数据驻留合规性
allow if {
	# 用户的地理位置与数据存储位置匹配
	data_residency_compliant
	# 其他访问条件
	input.subject.tenant_id == input.resource.tenant_id
	has_tenant_role
}

# 检查数据驻留合规性
data_residency_compliant if {
	# 没有地理限制
	not input.resource.data_residency_requirement
}

data_residency_compliant if {
	# 用户在允许的地理区域
	input.subject.attributes.region == input.resource.data_residency_requirement
}

data_residency_compliant if {
	# 用户的 IP 地址在允许的国家
	input.context.country == input.resource.data_residency_requirement
}

# ===================================
# 规则 10: 平台管理员访问
# ===================================

# 平台管理员可以访问所有租户数据（用于支持和管理）
allow if {
	"platform_admin" in input.subject.roles
	# 必须有有效的支持工单
	has_valid_support_ticket
	# 记录所有访问
	input.context.audit_required == true
}

# 检查支持工单
has_valid_support_ticket if {
	input.context.support_ticket_id
	input.context.support_ticket_status == "active"
}

# ===================================
# 规则 11: 基于 IP 白名单的访问
# ===================================

# 企业租户可以配置 IP 白名单
allow if {
	input.subject.subscription_plan == "enterprise"
	# 检查 IP 白名单
	ip_whitelist_check
	# 其他访问条件
	input.subject.tenant_id == input.resource.tenant_id
	has_tenant_role
}

# IP 白名单检查
ip_whitelist_check if {
	# 没有配置白名单
	not input.subject.tenant_config.ip_whitelist
}

ip_whitelist_check if {
	# IP 在白名单中
	input.context.ip_address in input.subject.tenant_config.ip_whitelist
}

# ===================================
# 规则 12: 基于设备管理的访问
# ===================================

# 企业租户可以要求设备注册
allow if {
	# 设备已注册
	device_registered
	# 或者订阅计划不要求设备注册
	not device_registration_required
	# 其他访问条件
	input.subject.tenant_id == input.resource.tenant_id
	has_tenant_role
}

# 检查设备是否需要注册
device_registration_required if {
	input.subject.subscription_plan == "enterprise"
	input.subject.tenant_config.require_device_registration == true
}

# 检查设备是否已注册
device_registered if {
	input.context.device_id
	input.context.device_id in input.subject.tenant_config.registered_devices
}

# ===================================
# 审计日志
# ===================================

audit_log := {
	"user": input.subject.user,
	"tenant_id": input.subject.tenant_id,
	"subscription_plan": input.subject.subscription_plan,
	"roles": input.subject.roles,
	"action": input.action,
	"resource_type": input.resource.type,
	"resource_id": input.resource.id,
	"resource_tenant": input.resource.tenant_id,
	"cross_tenant_access": input.subject.tenant_id != input.resource.tenant_id,
	"shared_access": is_shared_with_tenant,
	"workspace_id": input.resource.workspace_id,
	"ip_address": input.context.ip_address,
	"device_id": input.context.device_id,
	"country": input.context.country,
	"decision": allow,
	"timestamp": time.now_ns(),
	"quota_status": {
		"api_calls_today": input.subject.api_calls_today,
		"quota_exceeded": quota_exceeded,
	},
}

