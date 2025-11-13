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

# 金融行业 ABAC 策略示例
# Financial Industry ABAC Policy Example
# 
# 场景：银行系统中基于交易金额、风险级别和合规要求的访问控制
# Scenario: Access control based on transaction amount, risk level and compliance requirements

package financial.abac

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 规则 1: 基于交易金额的分级授权
# ===================================

# 普通员工可以处理小额交易（<= $10,000）
allow if {
	"teller" in input.subject.roles
	input.action == "approve"
	input.resource.type == "transaction"
	input.resource.amount <= 10000
}

# 主管可以处理中等交易（<= $100,000）
allow if {
	"supervisor" in input.subject.roles
	input.action == "approve"
	input.resource.type == "transaction"
	input.resource.amount <= 100000
}

# 经理可以处理大额交易（<= $1,000,000）
allow if {
	"manager" in input.subject.roles
	input.action == "approve"
	input.resource.type == "transaction"
	input.resource.amount <= 1000000
}

# 超大额交易需要高级管理层批准（> $1,000,000）
allow if {
	"senior_management" in input.subject.roles
	input.action == "approve"
	input.resource.type == "transaction"
	input.resource.amount > 1000000
}

# ===================================
# 规则 2: 基于风险评分的访问控制
# ===================================

# 高风险交易需要风险管理团队审核
allow if {
	"risk_manager" in input.subject.roles
	input.resource.type == "transaction"
	input.resource.risk_score >= 70
	input.action in ["review", "approve", "reject"]
}

# 低风险交易可以自动处理
allow if {
	input.subject.roles[_] in ["teller", "agent"]
	input.resource.type == "transaction"
	input.resource.risk_score < 30
	input.action in ["process", "approve"]
}

# ===================================
# 规则 3: 四眼原则（Maker-Checker）
# ===================================

# 创建者不能批准自己创建的交易
allow if {
	input.action == "approve"
	input.resource.type == "transaction"
	# 用户不是创建者
	input.subject.user != input.resource.created_by
	# 用户有批准权限
	has_approval_authority
}

# 检查批准权限
has_approval_authority if {
	input.subject.roles[_] in ["supervisor", "manager", "senior_management"]
}

# ===================================
# 规则 4: 基于客户类型的访问
# ===================================

# VIP客户账户需要专门的客户经理
allow if {
	"vip_account_manager" in input.subject.roles
	input.resource.type == "account"
	input.resource.customer_tier == "vip"
	input.action in ["read", "update", "transaction"]
}

# 普通客户账户可以由任何客服处理
allow if {
	input.subject.roles[_] in ["agent", "teller"]
	input.resource.type == "account"
	input.resource.customer_tier in ["standard", "bronze", "silver"]
	input.action in ["read", "inquiry"]
}

# ===================================
# 规则 5: 时间限制（交易时间窗口）
# ===================================

# 大额交易只能在工作时间处理
allow if {
	is_business_hours
	input.resource.type == "transaction"
	input.resource.amount > 50000
	input.action == "approve"
	has_approval_authority
}

# 判断是否在工作时间
is_business_hours if {
	input.context.time
	current_time := input.context.time
	# 周一到周五
	current_time.weekday >= 1
	current_time.weekday <= 5
	# 9:00-17:00
	current_time.hour >= 9
	current_time.hour < 17
}

# ===================================
# 规则 6: 地理位置限制
# ===================================

# 国际转账需要从总部或特定分行发起
allow if {
	input.resource.type == "international_transfer"
	input.action == "initiate"
	# 位置必须是授权地点
	input.context.location in ["headquarters", "international_branch"]
	# 用户必须有国际业务权限
	has_international_authority
}

# 检查国际业务权限
has_international_authority if {
	input.subject.attributes.certifications[_] == "international_transfer"
}

# ===================================
# 规则 7: 合规要求（KYC/AML）
# ===================================

# 高风险客户的交易需要完成 KYC
allow if {
	input.resource.type == "transaction"
	input.resource.customer_risk_level == "high"
	# 必须完成 KYC
	kyc_completed
	# 必须完成 AML 检查
	aml_check_passed
	input.action in ["process", "approve"]
}

# 检查 KYC 状态
kyc_completed if {
	input.resource.kyc_status == "verified"
	input.resource.kyc_date
	# KYC 在一年内有效
	time.now_ns() - input.resource.kyc_date < 31536000000000000
}

# 检查 AML 状态
aml_check_passed if {
	input.resource.aml_status == "clear"
}

# ===================================
# 规则 8: 基于 IP 地址的访问控制
# ===================================

# 敏感操作必须从内网执行
allow if {
	is_internal_network
	input.action in ["export", "bulk_transfer", "system_config"]
	input.subject.roles[_] in ["admin", "senior_management"]
}

# 检查是否为内网
is_internal_network if {
	input.context.ip_address
	# 银行内网
	startswith(input.context.ip_address, "10.bank.")
}

is_internal_network if {
	input.context.ip_address
	# VPN 地址段
	startswith(input.context.ip_address, "172.vpn.")
}

# ===================================
# 规则 9: 职责分离（Segregation of Duties）
# ===================================

# 交易员不能同时是风险审核员
allow if {
	input.action == "trade"
	"trader" in input.subject.roles
	# 确保用户不是风险审核员
	not "risk_manager" in input.subject.roles
}

# ===================================
# 规则 10: 账户所有者自助服务
# ===================================

# 账户所有者可以查看和管理自己的账户
allow if {
	input.resource.type == "account"
	input.resource.owner == input.subject.user
	input.action in ["read", "view_statement", "transfer", "pay_bill"]
	# 转账金额限制
	transfer_amount_within_limit
}

# 检查转账金额限制
transfer_amount_within_limit if {
	not input.action == "transfer"
}

transfer_amount_within_limit if {
	input.action == "transfer"
	input.resource.amount <= input.subject.attributes.daily_transfer_limit
}

# ===================================
# 规则 11: 基于设备信任级别
# ===================================

# 高价值操作需要从受信任设备执行
allow if {
	is_trusted_device
	input.resource.type == "transaction"
	input.resource.amount > 100000
	input.action == "approve"
	has_approval_authority
}

# 检查设备是否受信任
is_trusted_device if {
	input.context.device_id
	input.context.device_trusted == true
}

is_trusted_device if {
	input.context.device_type == "workstation"
	is_internal_network
}

# ===================================
# 规则 12: 监管审计访问
# ===================================

# 审计员可以只读访问所有数据
allow if {
	"auditor" in input.subject.roles
	input.action in ["read", "view", "export_audit_log"]
	# 必须记录审计员的所有访问
	audit_trail_enabled
}

# 检查审计追踪是否启用
audit_trail_enabled if {
	input.context.audit_enabled == true
}

# ===================================
# 审计日志
# ===================================

audit_log := {
	"user": input.subject.user,
	"roles": input.subject.roles,
	"action": input.action,
	"resource_type": input.resource.type,
	"resource_id": input.resource.id,
	"amount": input.resource.amount,
	"risk_score": input.resource.risk_score,
	"customer_tier": input.resource.customer_tier,
	"location": input.context.location,
	"ip_address": input.context.ip_address,
	"device_id": input.context.device_id,
	"decision": allow,
	"timestamp": time.now_ns(),
	"compliance_flags": {
		"kyc_completed": kyc_completed,
		"aml_passed": aml_check_passed,
		"from_trusted_device": is_trusted_device,
		"during_business_hours": is_business_hours,
	},
}

