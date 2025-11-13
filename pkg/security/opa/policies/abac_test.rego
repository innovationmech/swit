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

# ABAC 策略单元测试
# 测试基于属性的访问控制策略

package abac_test

import rego.v1
import data.abac

# ===================================
# 测试：默认拒绝策略
# ===================================

test_default_deny if {
	not abac.allow with input as {}
}

test_deny_without_subject if {
	not abac.allow with input as {
		"action": "read",
		"resource": "documents",
	}
}

test_deny_without_action if {
	not abac.allow with input as {
		"subject": {"user": "alice", "roles": ["employee"]},
		"resource": "documents",
	}
}

test_deny_without_resource if {
	not abac.allow with input as {
		"subject": {"user": "alice", "roles": ["employee"]},
		"action": "read",
	}
}

# ===================================
# 测试：超级管理员权限
# ===================================

test_admin_has_all_permissions if {
	abac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
		"action": "delete",
		"resource": "sensitive-data",
	}
}

test_admin_can_access_any_resource if {
	abac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
		"action": "create",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 4,
				"sensitivity": "critical",
			},
		},
	}
}

# ===================================
# 测试：资源所有者权限
# ===================================

test_owner_can_read_resource if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"owner": "alice",
		},
	}
}

test_owner_can_update_resource if {
	abac.allow with input as {
		"subject": {
			"user": "bob",
			"roles": ["employee"],
		},
		"action": "update",
		"resource": {
			"type": "document",
			"id": "doc-456",
			"owner": "bob",
		},
	}
}

test_owner_can_delete_resource if {
	abac.allow with input as {
		"subject": {
			"user": "charlie",
			"roles": ["employee"],
		},
		"action": "delete",
		"resource": {
			"type": "document",
			"id": "doc-789",
			"owner": "charlie",
		},
	}
}

test_non_owner_cannot_access if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"owner": "bob",
		},
	}
}

# ===================================
# 测试：基于部门的访问控制
# ===================================

test_same_department_can_read if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"department": "engineering",
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"department": "engineering",
			},
		},
	}
}

test_different_department_cannot_read if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"department": "engineering",
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"department": "finance",
			},
		},
	}
}

# ===================================
# 测试：基于安全级别的访问控制
# ===================================

test_sufficient_clearance_level if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"clearance_level": 3,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 2,
			},
		},
	}
}

test_insufficient_clearance_level if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"clearance_level": 1,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 3,
			},
		},
	}
}

test_equal_clearance_level if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"clearance_level": 2,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 2,
			},
		},
	}
}

# ===================================
# 测试：基于项目成员的访问控制
# ===================================

test_project_member_can_read if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "project",
			"id": "project-123",
			"attributes": {
				"members": ["alice", "bob", "charlie"],
			},
		},
	}
}

test_project_member_can_update if {
	abac.allow with input as {
		"subject": {
			"user": "bob",
			"roles": ["employee"],
		},
		"action": "update",
		"resource": {
			"type": "project",
			"id": "project-123",
			"attributes": {
				"members": ["alice", "bob", "charlie"],
			},
		},
	}
}

test_non_project_member_cannot_access if {
	not abac.allow with input as {
		"subject": {
			"user": "dave",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "project",
			"id": "project-123",
			"attributes": {
				"members": ["alice", "bob", "charlie"],
			},
		},
	}
}

# ===================================
# 测试：基于时间的访问控制
# ===================================

test_business_hours_access if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 14,
			},
		},
	}
}

test_outside_business_hours_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 20,
			},
		},
	}
}

test_weekend_access_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"time": {
				"weekday": 6,
				"hour": 14,
			},
		},
	}
}

test_time_window_within_range if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"time_window": {
				"start": 1000000000000000000,
				"end": 9999999999999999999,
			},
		},
	}
}

test_time_window_outside_range if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"time_window": {
				"start": 1,
				"end": 1000,
			},
		},
	}
}

# ===================================
# 测试：基于地理位置的访问控制
# ===================================

test_allowed_location_access if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"sensitivity": "high",
			},
		},
		"environment": {
			"location": "office",
		},
	}
}

test_disallowed_location_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"sensitivity": "high",
			},
		},
		"environment": {
			"location": "public-wifi",
		},
	}
}

# ===================================
# 测试：基于 IP 地址的访问控制
# ===================================

test_internal_ip_allowed if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "delete",
		"resource": "documents",
		"environment": {
			"ip_address": "10.0.1.100",
		},
	}
}

test_internal_ip_192_allowed if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "update",
		"resource": "documents",
		"environment": {
			"ip_address": "192.168.1.100",
		},
	}
}

test_internal_ip_172_allowed if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "update",
		"resource": "documents",
		"environment": {
			"ip_address": "172.16.0.100",
		},
	}
}

test_external_ip_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "delete",
		"resource": "documents",
		"environment": {
			"ip_address": "1.2.3.4",
		},
	}
}

# ===================================
# 测试：基于设备类型的访问控制
# ===================================

test_allowed_device_access if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": "documents",
		"environment": {
			"device_type": "laptop",
		},
	}
}

test_trusted_device_can_delete if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "delete",
		"resource": "documents",
		"environment": {
			"device_type": "desktop",
		},
	}
}

test_untrusted_device_cannot_delete if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "delete",
		"resource": "documents",
		"environment": {
			"device_type": "mobile",
		},
	}
}

# ===================================
# 测试：基于数据分类的访问控制
# ===================================

test_public_data_access if {
	abac.allow with input as {
		"subject": {
			"user": "anyone",
			"roles": ["guest"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"classification": "public",
			},
		},
	}
}

test_internal_data_requires_employee if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"classification": "internal",
			},
		},
	}
}

test_internal_data_guest_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "guest",
			"roles": ["guest"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"classification": "internal",
			},
		},
	}
}

test_confidential_data_requires_authorization if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"access_levels": ["authorized"],
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"classification": "confidential",
			},
		},
	}
}

test_confidential_data_unauthorized_denied if {
	not abac.allow with input as {
		"subject": {
			"user": "bob",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"classification": "confidential",
			},
		},
	}
}

# ===================================
# 测试：动态属性评分机制
# ===================================

test_high_score_grants_access if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["manager"],
			"attributes": {
				"department": "engineering",
				"clearance_level": 3,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"department": "engineering",
				"security_level": 2,
			},
		},
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 14,
			},
			"location": "office",
			"ip_address": "10.0.1.100",
			"device_type": "desktop",
		},
	}
}

test_low_score_denies_access if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["guest"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 3,
			},
		},
	}
}

test_score_calculation if {
	score := abac.dynamic_attribute_score with input as {
		"subject": {
			"user": "alice",
			"roles": ["admin"],
			"attributes": {
				"department": "engineering",
				"clearance_level": 4,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"department": "engineering",
				"security_level": 3,
			},
		},
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 14,
			},
			"location": "office",
			"ip_address": "10.0.1.100",
			"device_type": "desktop",
		},
	}
	score >= 90
}

# ===================================
# 测试：辅助函数
# ===================================

test_user_clearance_level_extraction if {
	level := abac.user_clearance_level with input as {
		"subject": {
			"user": "alice",
			"attributes": {
				"clearance_level": 3,
			},
		},
	}
	level == 3
}

test_user_clearance_level_default if {
	level := abac.user_clearance_level with input as {
		"subject": {
			"user": "alice",
		},
	}
	level == 0
}

test_resource_security_level_extraction if {
	level := abac.resource_security_level with input as {
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 2,
			},
		},
	}
	level == 2
}

test_resource_security_level_default if {
	level := abac.resource_security_level with input as {
		"resource": "documents",
	}
	level == 0
}

test_is_business_hours_weekday if {
	abac.is_business_hours with input as {
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 14,
			},
		},
	}
}

test_is_not_business_hours_evening if {
	not abac.is_business_hours with input as {
		"environment": {
			"time": {
				"weekday": 3,
				"hour": 20,
			},
		},
	}
}

test_location_allowed_check if {
	abac.location_allowed with input as {
		"environment": {
			"location": "office",
		},
	}
}

test_location_not_allowed if {
	not abac.location_allowed with input as {
		"environment": {
			"location": "unknown",
		},
	}
}

test_ip_allowed_check if {
	abac.is_ip_allowed("10.0.1.100")
}

test_ip_not_allowed if {
	not abac.is_ip_allowed("1.2.3.4")
}

# ===================================
# 测试：审计日志生成
# ===================================

test_audit_log_contains_required_fields if {
	log := abac.audit_log with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
			"attributes": {
				"clearance_level": 2,
			},
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
			"attributes": {
				"security_level": 1,
			},
		},
	}
	log.user == "alice"
	log.action == "read"
	log.timestamp
}

test_audit_log_includes_score_breakdown if {
	log := abac.audit_log with input as {
		"subject": {
			"user": "alice",
			"roles": ["manager"],
		},
		"action": "read",
		"resource": "documents",
	}
	log.score_breakdown.role_score
	log.score_breakdown.clearance_score
	log.score_breakdown.time_score
}

# ===================================
# 测试：边界情况和错误处理
# ===================================

test_empty_roles_with_score if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_missing_attributes if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "read",
		"resource": {
			"type": "document",
			"id": "doc-123",
		},
	}
}

test_simple_resource_string if {
	abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["admin"],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_unknown_action if {
	not abac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": ["employee"],
		},
		"action": "unknown_action",
		"resource": "documents",
	}
}

