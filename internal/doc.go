// internal 私有应用代码，遵循标准 Go 工程布局（https://go.dev/doc/modules/layout）。
//
// 限界上下文（领域驱动、单向依赖）：
//
//	analysis/   AI 多智能体投研、报告、HTTP：interfaces/http（原 api）、agents、graph、tools
//	workbench/  工作区与行情：domain、ports、application、infrastructure、interfaces/http
//
// cmd 仅做依赖组装与进程入口，不包含业务规则。
package internal
