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

package resilience

// Strategy 定义重试退避的策略类型。
//
// - Fixed: 固定间隔
// - Linear: 线性递增
// - Exponential: 指数退避
// - Jittered: 指数退避并叠加抖动
type Strategy string

const (
	// StrategyFixed 固定间隔
	StrategyFixed Strategy = "FIXED"
	// StrategyLinear 线性退避
	StrategyLinear Strategy = "LINEAR"
	// StrategyExponential 指数退避
	StrategyExponential Strategy = "EXPONENTIAL"
	// StrategyJittered 指数退避 + 抖动
	StrategyJittered Strategy = "JITTERED"
	// StrategyNone 不重试
	StrategyNone Strategy = "NONE"
)
