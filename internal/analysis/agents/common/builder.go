package common

import (
	"context"
	"fmt"
	"time"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common/toolerr"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// AgentBuilder 智能体构建器
type AgentBuilder struct {
	name         string
	description  string
	tools        []tool.BaseTool
	toolNames    []string
	template     prompt.ChatTemplate
	templateVars map[string]any
	instruction  string

	// 模型配置
	chatModel model.ToolCallingChatModel

	// 中间件
	middlewares []adk.AgentMiddleware
	handlers    []adk.ChatModelAgentMiddleware
}

// NewAgentBuilder 创建构建器
func NewAgentBuilder(name, description string) *AgentBuilder {
	return &AgentBuilder{
		name:         name,
		description:  description,
		templateVars: map[string]any{"current_time": time.Now().Format("2006-01-02 15:04:05")},
	}
}

// WithTools 设置工具列表
func (b *AgentBuilder) WithTools(tools ...tool.BaseTool) *AgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithToolNames 通过工具名称添加工具
func (b *AgentBuilder) WithToolNames(names ...string) *AgentBuilder {
	b.toolNames = append(b.toolNames, names...)
	return b
}

// WithTemplate 设置提示词模板
func (b *AgentBuilder) WithTemplate(t prompt.ChatTemplate) *AgentBuilder {
	b.template = t
	return b
}

// WithInstruction 直接设置系统提示词（不需要模板解析）
func (b *AgentBuilder) WithInstruction(instruction string) *AgentBuilder {
	b.instruction = instruction
	return b
}

// WithTemplateVar 添加模板变量
func (b *AgentBuilder) WithTemplateVar(key string, value any) *AgentBuilder {
	b.templateVars[key] = value
	return b
}

// WithModel 使用自定义模型
func (b *AgentBuilder) WithModel(m model.ToolCallingChatModel) *AgentBuilder {
	b.chatModel = m
	return b
}

// WithMiddleware 添加 AgentMiddleware
func (b *AgentBuilder) WithMiddleware(m ...adk.AgentMiddleware) *AgentBuilder {
	b.middlewares = append(b.middlewares, m...)
	return b
}

// WithHandler 添加 ChatModelAgentMiddleware
func (b *AgentBuilder) WithHandler(h ...adk.ChatModelAgentMiddleware) *AgentBuilder {
	b.handlers = append(b.handlers, h...)
	return b
}

// Build 构建智能体
func (b *AgentBuilder) Build(ctx context.Context) (adk.Agent, error) {
	// 模型
	chatModel := b.chatModel
	if chatModel == nil {
		chatModel = NewChatModel()
	}

	// 提示词
	var instruction string
	if b.instruction != "" {
		instruction = b.instruction
	} else if b.template != nil {
		msgs, err := b.template.Format(ctx, b.templateVars)
		if err != nil {
			return nil, fmt.Errorf("format prompt: %w", err)
		}
		instruction = msgs[0].Content
	}

	// 构建配置
	cfg := &adk.ChatModelAgentConfig{
		Name:          b.name,
		Description:   b.description,
		Instruction:   instruction,
		Model:         chatModel,
		MaxIterations: 10,
		Middlewares:   b.middlewares,
		Handlers:      b.handlers,
	}

	// 工具（单个工具失败时不中断整张图：将错误写回为工具输出，模型可继续）
	if len(b.tools) > 0 {
		cfg.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: b.tools,
				ToolCallMiddlewares: []compose.ToolMiddleware{
					toolerr.Middleware(),
				},
			},
		}
	}

	return adk.NewChatModelAgent(ctx, cfg)
}

// SimplePrompt 创建简单的系统提示词模板
func SimplePrompt(systemPrompt string) prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemPrompt),
	)
}
