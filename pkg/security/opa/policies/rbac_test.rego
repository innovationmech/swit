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

# RBAC 策略单元测试
# 测试基于角色的访问控制策略

package rbac_test

import rego.v1
import data.rbac

# ===================================
# 测试：默认拒绝策略
# ===================================

test_default_deny if {
	not rbac.allow with input as {}
}

test_deny_without_roles if {
	not rbac.allow with input as {
		"subject": {"user": "john"},
		"action": "read",
		"resource": "documents",
	}
}

# ===================================
# 测试：超级管理员权限
# ===================================

test_admin_has_all_permissions if {
	rbac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
		"action": "delete",
		"resource": "settings",
	}
}

test_admin_can_create if {
	rbac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
		"action": "create",
		"resource": "users",
	}
}

test_admin_can_delete_any_resource if {
	rbac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
		"action": "delete",
		"resource": "reports",
	}
}

# ===================================
# 测试：编辑者角色权限
# ===================================

test_editor_can_create if {
	rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "create",
		"resource": "documents",
	}
}

test_editor_can_update if {
	rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "update",
		"resource": "documents",
	}
}

test_editor_can_read if {
	rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "read",
		"resource": "reports",
	}
}

test_editor_cannot_delete if {
	not rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "delete",
		"resource": "documents",
	}
}

test_editor_cannot_access_users if {
	not rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "read",
		"resource": "users",
	}
}

test_editor_cannot_access_settings if {
	not rbac.allow with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
		"action": "read",
		"resource": "settings",
	}
}

# ===================================
# 测试：查看者角色权限
# ===================================

test_viewer_can_read if {
	rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_viewer_can_list if {
	rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "list",
		"resource": "reports",
	}
}

test_viewer_cannot_create if {
	not rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "create",
		"resource": "documents",
	}
}

test_viewer_cannot_update if {
	not rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "update",
		"resource": "documents",
	}
}

test_viewer_cannot_delete if {
	not rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "delete",
		"resource": "documents",
	}
}

# ===================================
# 测试：贡献者角色权限
# ===================================

test_contributor_can_create if {
	rbac.allow with input as {
		"subject": {
			"user": "contrib-user",
			"roles": ["contributor"],
		},
		"action": "create",
		"resource": "documents",
	}
}

test_contributor_can_read if {
	rbac.allow with input as {
		"subject": {
			"user": "contrib-user",
			"roles": ["contributor"],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_contributor_cannot_update if {
	not rbac.allow with input as {
		"subject": {
			"user": "contrib-user",
			"roles": ["contributor"],
		},
		"action": "update",
		"resource": "documents",
	}
}

test_contributor_cannot_delete if {
	not rbac.allow with input as {
		"subject": {
			"user": "contrib-user",
			"roles": ["contributor"],
		},
		"action": "delete",
		"resource": "documents",
	}
}

# ===================================
# 测试：多角色场景
# ===================================

test_user_with_multiple_roles if {
	rbac.allow with input as {
		"subject": {
			"user": "multi-role-user",
			"roles": ["viewer", "contributor"],
		},
		"action": "create",
		"resource": "documents",
	}
}

test_user_highest_permission_wins if {
	rbac.allow with input as {
		"subject": {
			"user": "multi-role-user",
			"roles": ["viewer", "admin"],
		},
		"action": "delete",
		"resource": "settings",
	}
}

# ===================================
# 测试：资源所有者检查
# ===================================

test_owner_can_read_own_resource if {
	rbac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "read",
		"resource": "users/alice",
	}
}

test_owner_can_update_own_resource if {
	rbac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "update",
		"resource": "users/alice",
	}
}

test_owner_can_delete_own_resource if {
	rbac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "delete",
		"resource": "users/alice",
	}
}

test_non_owner_cannot_access if {
	not rbac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "read",
		"resource": "users/bob",
	}
}

test_owner_cannot_create_on_own_resource if {
	not rbac.allow with input as {
		"subject": {
			"user": "alice",
			"roles": [],
		},
		"action": "create",
		"resource": "users/alice",
	}
}

# ===================================
# 测试：用户组角色继承
# ===================================

test_engineering_group_has_editor_role if {
	rbac.allow with input as {
		"subject": {
			"user": "engineer1",
			"roles": [],
			"groups": ["engineering"],
		},
		"action": "update",
		"resource": "documents",
	}
}

test_management_group_has_admin_role if {
	rbac.allow with input as {
		"subject": {
			"user": "manager1",
			"roles": [],
			"groups": ["management"],
		},
		"action": "delete",
		"resource": "users",
	}
}

test_support_group_has_viewer_role if {
	rbac.allow with input as {
		"subject": {
			"user": "support1",
			"roles": [],
			"groups": ["support"],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_support_group_cannot_write if {
	not rbac.allow with input as {
		"subject": {
			"user": "support1",
			"roles": [],
			"groups": ["support"],
		},
		"action": "create",
		"resource": "documents",
	}
}

test_sales_group_mixed_permissions if {
	rbac.allow with input as {
		"subject": {
			"user": "sales1",
			"roles": [],
			"groups": ["sales"],
		},
		"action": "create",
		"resource": "documents",
	}
}

# ===================================
# 测试：临时权限
# ===================================

test_temporary_permission_grants_access if {
	rbac.allow with input as {
		"subject": {
			"user": "temp-user",
			"roles": [],
		},
		"action": "write",
		"resource": "project/secret",
		"context": {
			"time": true,
			"temporary_permissions": [{
				"action": "write",
				"resource": "project/secret",
				"expires_at": 9999999999999999999,
			}],
		},
	}
}

test_expired_temporary_permission_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "temp-user",
			"roles": [],
		},
		"action": "write",
		"resource": "project/secret",
		"context": {
			"time": true,
			"temporary_permissions": [{
				"action": "write",
				"resource": "project/secret",
				"expires_at": 1,
			}],
		},
	}
}

# ===================================
# 测试：工作时间检查
# ===================================

test_business_hours_with_temp_permission if {
	rbac.allow with input as {
		"subject": {
			"user": "employee1",
			"roles": [],
		},
		"action": "read",
		"resource": "internal-docs",
		"context": {
			"time": true,
			"temporary_permissions": [{
				"action": "read",
				"resource": "internal-docs",
				"expires_at": 9999999999999999999,
			}],
		},
	}
}

# ===================================
# 测试：辅助函数
# ===================================

test_user_roles_extraction if {
	result := rbac.user_roles with input as {
		"subject": {
			"user": "test-user",
			"roles": ["admin", "editor"],
		},
	}
	result == {"admin", "editor"}
}

test_user_permissions_includes_admin_perms if {
	result := rbac.user_permissions with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["admin"],
		},
	}
	"create" in result
	"read" in result
	"update" in result
	"delete" in result
	"list" in result
}

test_has_permission_checks_correctly if {
	rbac.has_permission("read") with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
	}
}

test_user_resources_for_editor if {
	result := rbac.user_resources with input as {
		"subject": {
			"user": "editor-user",
			"roles": ["editor"],
		},
	}
	"documents" in result
	"reports" in result
}

test_can_access_resource_checks_correctly if {
	rbac.can_access_resource("documents") with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
	}
}

test_is_owner_identifies_correctly if {
	rbac.is_owner with input as {
		"subject": {"user": "alice"},
		"resource": "users/alice",
	}
}

test_is_owner_fails_for_non_owner if {
	not rbac.is_owner with input as {
		"subject": {"user": "alice"},
		"resource": "users/bob",
	}
}

# ===================================
# 测试：审计日志生成
# ===================================

test_audit_log_contains_required_fields if {
	log := rbac.audit_log with input as {
		"subject": {
			"user": "test-user",
			"roles": ["viewer"],
		},
		"action": "read",
		"resource": "documents",
	}
	log.user == "test-user"
	log.action == "read"
	log.resource == "documents"
	log.timestamp
}

# ===================================
# 测试：边界情况
# ===================================

test_empty_roles_denies_access if {
	not rbac.allow with input as {
		"subject": {
			"user": "user-no-roles",
			"roles": [],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_missing_subject_denies if {
	not rbac.allow with input as {
		"action": "read",
		"resource": "documents",
	}
}

test_missing_action_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "test-user",
			"roles": ["viewer"],
		},
		"resource": "documents",
	}
}

test_missing_resource_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "test-user",
			"roles": ["viewer"],
		},
		"action": "read",
	}
}

test_unknown_role_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "test-user",
			"roles": ["unknown_role"],
		},
		"action": "read",
		"resource": "documents",
	}
}

test_unknown_action_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "viewer-user",
			"roles": ["viewer"],
		},
		"action": "unknown_action",
		"resource": "documents",
	}
}

test_unknown_resource_denies if {
	not rbac.allow with input as {
		"subject": {
			"user": "admin-user",
			"roles": ["editor"],
		},
		"action": "read",
		"resource": "unknown_resource",
	}
}

