package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AmazingCYJ/AgentRAG/internal/mcpserver"
)

type mcpToolParameter struct {
	description  string
	typ          string
	required     bool
	defaultValue any
	enumValues   []string
}

type mcpToolSchema struct {
	toolID      string
	description string
	parameters  map[string]mcpToolParameter
}

func parseToolArguments(rawJSON string, schema mcpToolSchema) (map[string]any, error) {
	if strings.TrimSpace(rawJSON) == "" {
		return map[string]any{}, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, err
	}

	result := make(map[string]any, len(schema.parameters))
	for key, definition := range schema.parameters {
		value, ok := payload[key]
		if !ok {
			if definition.defaultValue != nil {
				result[key] = definition.defaultValue
			}
			continue
		}
		result[key] = normalizeToolValue(value, definition.typ)
	}
	return result, nil
}

func ToChatToolSchema(definition mcpserver.ToolDefinition) mcpToolSchema {
	parameters := make(map[string]mcpToolParameter, len(definition.Parameters))
	for name, parameter := range definition.Parameters {
		parameters[name] = mcpToolParameter{
			description:  parameter.Description,
			typ:          parameter.Type,
			required:     parameter.Required,
			defaultValue: parameter.DefaultValue,
			enumValues:   parameter.EnumValues,
		}
	}
	return mcpToolSchema{
		toolID:      definition.ToolID,
		description: definition.Description,
		parameters:  parameters,
	}
}

func buildToolDefinitionPrompt(schema mcpToolSchema) string {
	var builder strings.Builder
	builder.WriteString("工具ID: " + schema.toolID + "\n")
	builder.WriteString("功能描述: " + schema.description + "\n")
	builder.WriteString("参数列表:\n")
	for name, parameter := range schema.parameters {
		builder.WriteString("  - " + name)
		builder.WriteString(" (类型: " + parameter.typ)
		if parameter.required {
			builder.WriteString(", 必填")
		} else {
			builder.WriteString(", 可选")
		}
		builder.WriteString("): " + parameter.description)
		if parameter.defaultValue != nil {
			builder.WriteString(" [默认值: " + fmt.Sprint(parameter.defaultValue) + "]")
		}
		if len(parameter.enumValues) > 0 {
			builder.WriteString(" [可选值: " + strings.Join(parameter.enumValues, ", ") + "]")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func normalizeToolValue(value any, typ string) any {
	switch typ {
	case "integer", "number":
		switch cast := value.(type) {
		case float64:
			if typ == "integer" {
				return int(cast)
			}
			return cast
		default:
			return value
		}
	default:
		return value
	}
}
