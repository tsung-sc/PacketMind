package agent

import "strings"

// defaultAgentRoleText is loaded from prompts/agent_role.txt at package init.
var defaultAgentRoleText = LoadPrompt("agent_role")

// DefaultAgentRole returns the main agent system prompt loaded from the
// external prompt file. Used by SystemPromptBuilder to construct the full
// system prompt for the ReAct agent loop.
func DefaultAgentRole() string {
	return defaultAgentRoleText
}

type SystemPromptBuilder struct {
	sections []string
}

func NewSystemPromptBuilder() *SystemPromptBuilder {
	return &SystemPromptBuilder{}
}

func (b *SystemPromptBuilder) WithRole(role string) *SystemPromptBuilder {
	return b.WithSection(role)
}

func (b *SystemPromptBuilder) WithInstructions(instructions string) *SystemPromptBuilder {
	if text := strings.TrimSpace(instructions); text != "" {
		b.sections = append([]string{text}, b.sections...)
	}
	return b
}

func (b *SystemPromptBuilder) WithSection(section string) *SystemPromptBuilder {
	if text := strings.TrimSpace(section); text != "" {
		b.sections = append(b.sections, text)
	}
	return b
}

func (b *SystemPromptBuilder) Build() string {
	if b == nil || len(b.sections) == 0 {
		return ""
	}
	nonEmpty := make([]string, 0, len(b.sections))
	for _, section := range b.sections {
		if text := strings.TrimSpace(section); text != "" {
			nonEmpty = append(nonEmpty, text)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}
