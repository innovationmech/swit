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

# 医疗保健行业 ABAC 策略示例
# Healthcare Industry ABAC Policy Example
# 
# 场景：医疗系统中基于患者隐私保护的访问控制
# Scenario: Access control based on patient privacy protection in healthcare systems

package healthcare.abac

import rego.v1

# 默认拒绝所有访问
default allow := false

# ===================================
# 规则 1: 医生可以访问自己负责的患者记录
# ===================================

allow if {
	# 用户必须是医生
	"doctor" in input.subject.roles
	# 医生在患者的主治医生列表中
	input.subject.user in input.resource.attending_physicians
	# 允许的操作
	input.action in ["read", "update"]
}

# ===================================
# 规则 2: 护士可以读取病房内患者的记录
# ===================================

allow if {
	# 用户必须是护士
	"nurse" in input.subject.roles
	# 护士和患者在同一病房
	input.subject.attributes.ward == input.resource.attributes.ward
	# 只允许读取操作
	input.action == "read"
}

# ===================================
# 规则 3: 药剂师可以访问处方信息
# ===================================

allow if {
	# 用户必须是药剂师
	"pharmacist" in input.subject.roles
	# 资源类型必须是处方
	input.resource.type == "prescription"
	# 允许的操作
	input.action in ["read", "update"]
}

# ===================================
# 规则 4: 急诊访问（紧急情况下的临时访问）
# ===================================

allow if {
	# 用户必须是医疗人员
	input.subject.roles[_] in ["doctor", "nurse", "emergency_staff"]
	# 必须是紧急情况
	input.context.emergency == true
	# 患者必须在急诊室
	input.resource.attributes.location == "emergency_room"
	# 允许读取操作
	input.action == "read"
}

# ===================================
# 规则 5: 基于患者同意的访问
# ===================================

allow if {
	# 检查患者是否授权了访问
	patient_has_consented
	# 允许的操作
	input.action in ["read", "share"]
}

# 检查患者同意
patient_has_consented if {
	# 患者的同意列表包含当前用户
	some consent in input.resource.patient_consents
	consent.authorized_user == input.subject.user
	# 同意尚未过期
	consent.expires_at > time.now_ns()
}

# ===================================
# 规则 6: 研究人员访问去标识化数据
# ===================================

allow if {
	# 用户必须是研究人员
	"researcher" in input.subject.roles
	# 资源必须是去标识化的
	input.resource.attributes.de_identified == true
	# 研究人员必须有有效的IRB批准
	has_valid_irb_approval
	# 只允许读取
	input.action in ["read", "list"]
}

# 检查IRB批准
has_valid_irb_approval if {
	input.subject.attributes.irb_approval
	input.subject.attributes.irb_approval.valid == true
	input.subject.attributes.irb_approval.expires_at > time.now_ns()
}

# ===================================
# 规则 7: 时间限制（工作时间访问）
# ===================================

allow if {
	# 非紧急访问必须在工作时间内
	not input.context.emergency
	is_business_hours
	# 用户必须有适当的角色
	input.subject.roles[_] in ["doctor", "nurse", "admin"]
	# 允许基本操作
	input.action in ["read", "list"]
}

# 判断是否在工作时间
is_business_hours if {
	input.context.time
	current_time := input.context.time
	# 周一到周五
	current_time.weekday >= 1
	current_time.weekday <= 5
	# 8:00-20:00
	current_time.hour >= 8
	current_time.hour < 20
}

# ===================================
# 规则 8: 基于位置的访问
# ===================================

allow if {
	# 访问敏感记录必须从医院内部
	is_from_hospital_network
	# 用户必须是医疗人员
	is_medical_staff
	# 允许的操作
	input.action in ["read", "update"]
}

# 检查是否从医院网络访问
is_from_hospital_network if {
	input.context.ip_address
	# 医院内网 IP 段
	startswith(input.context.ip_address, "10.hospital.")
}

is_from_hospital_network if {
	input.context.location == "hospital"
}

# 检查是否为医疗人员
is_medical_staff if {
	input.subject.roles[_] in ["doctor", "nurse", "pharmacist", "admin"]
}

# ===================================
# 规则 9: 基于患者年龄的访问控制
# ===================================

allow if {
	# 儿科医生可以访问儿童患者记录
	"pediatrician" in input.subject.roles
	input.resource.attributes.patient_age < 18
	input.action in ["read", "update"]
}

# ===================================
# 规则 10: 审计要求
# ===================================

# 所有访问都必须记录审计日志
audit_log := {
	"user": input.subject.user,
	"roles": input.subject.roles,
	"action": input.action,
	"resource_type": input.resource.type,
	"resource_id": input.resource.id,
	"patient_id": input.resource.patient_id,
	"emergency": input.context.emergency,
	"location": input.context.location,
	"ip_address": input.context.ip_address,
	"decision": allow,
	"timestamp": time.now_ns(),
	"reason": deny_reason,
}

# 拒绝原因
deny_reason := "default_deny" if {
	not allow
}

deny_reason := "approved" if {
	allow
}

