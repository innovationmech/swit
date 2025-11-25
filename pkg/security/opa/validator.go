// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package opa

import (
	"fmt"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

// ValidatePolicy 验证 Rego 策略语法
func ValidatePolicy(policy string) error {
	if policy == "" {
		return fmt.Errorf("policy content cannot be empty")
	}

	// 解析策略
	module, err := ast.ParseModule("", policy)
	if err != nil {
		return fmt.Errorf("policy syntax error: %w", err)
	}

	// 编译策略以检查语义错误
	compiler := ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{
		"policy": module,
	})

	if compiler.Failed() {
		var errMsgs []string
		for _, err := range compiler.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("policy compilation errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// ValidatePolicyFromFile 从文件验证策略
func ValidatePolicyFromFile(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// 解析策略
	module, err := ast.ParseModule(filePath, string(content))
	if err != nil {
		return fmt.Errorf("failed to parse policy file: %w", err)
	}

	// 编译检查
	compiler := ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{
		filePath: module,
	})

	if compiler.Failed() {
		var errMsgs []string
		for _, err := range compiler.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("policy compilation errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// PolicyValidator 策略验证器
type PolicyValidator struct {
	// StrictMode 严格模式（启用额外的检查）
	StrictMode bool

	// RequireRegoV1 要求使用 Rego v1 语法
	RequireRegoV1 bool
}

// NewPolicyValidator 创建策略验证器
func NewPolicyValidator() *PolicyValidator {
	return &PolicyValidator{
		StrictMode:    false,
		RequireRegoV1: true, // 默认要求 Rego v1
	}
}

// Validate 验证策略
func (v *PolicyValidator) Validate(policy string) error {
	if policy == "" {
		return fmt.Errorf("policy content cannot be empty")
	}

	// 解析策略
	module, err := ast.ParseModule("", policy)
	if err != nil {
		return fmt.Errorf("policy syntax error: %w", err)
	}

	// 检查是否导入 rego.v1
	if v.RequireRegoV1 {
		if !hasRegoV1Import(module) {
			return fmt.Errorf("policy must import rego.v1")
		}
	}

	// 编译策略
	compiler := ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{
		"policy": module,
	})

	if compiler.Failed() {
		var errMsgs []string
		for _, err := range compiler.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("policy compilation errors: %s", strings.Join(errMsgs, "; "))
	}

	// 严格模式检查
	if v.StrictMode {
		if err := v.strictValidation(module); err != nil {
			return err
		}
	}

	return nil
}

// strictValidation 严格验证
func (v *PolicyValidator) strictValidation(module *ast.Module) error {
	// 检查是否有规则定义
	if len(module.Rules) == 0 {
		return fmt.Errorf("policy must contain at least one rule")
	}

	// 检查是否有包声明
	if module.Package == nil {
		return fmt.Errorf("policy must have a package declaration")
	}

	return nil
}

// hasRegoV1Import 检查是否导入了 rego.v1
func hasRegoV1Import(module *ast.Module) bool {
	for _, imp := range module.Imports {
		if imp.Path.String() == "data.rego.v1" {
			return true
		}
	}
	return false
}

// ValidatePolicyWithOptions 使用选项验证策略
func ValidatePolicyWithOptions(policy string, strictMode bool, requireRegoV1 bool) error {
	validator := &PolicyValidator{
		StrictMode:    strictMode,
		RequireRegoV1: requireRegoV1,
	}
	return validator.Validate(policy)
}
