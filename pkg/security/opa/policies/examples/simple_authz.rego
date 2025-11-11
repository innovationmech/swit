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

# 简单授权策略示例
# 演示基本的 OPA 策略编写

package example.authz

import rego.v1

# 默认拒绝
default allow := false

# 允许管理员执行任何操作
allow if {
    input.subject.user == "admin"
}

# 允许用户读取自己的数据
allow if {
    input.action == "read"
    input.resource == sprintf("users/%s", [input.subject.user])
}

# 允许用户列表操作
allow if {
    input.action == "list"
    "viewer" in input.subject.roles
}

# 允许用户创建资源
allow if {
    input.action == "create"
    "editor" in input.subject.roles
}

# 允许用户更新自己创建的资源
allow if {
    input.action == "update"
    input.resource_owner == input.subject.user
}

