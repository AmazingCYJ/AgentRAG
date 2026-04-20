package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	jsonRPCVersion    = "2.0"
	methodNotFound    = -32601
	invalidParamsCode = -32602
	serverName        = "ragent-mcp-server"
	serverVersion     = "0.0.1"
)

// JSONRPCRequest 定义 JSON-RPC 2.0 请求体。
type JSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// JSONRPCResponse 定义 JSON-RPC 2.0 响应体。
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError 定义 JSON-RPC 错误对象。
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition 定义 MCP 工具元信息。
type ToolDefinition struct {
	ToolID      string                  `json:"toolId"`
	Description string                  `json:"description"`
	Parameters  map[string]ParameterDef `json:"parameters,omitempty"`
}

// ParameterDef 定义工具参数。
type ParameterDef struct {
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	DefaultValue any      `json:"defaultValue,omitempty"`
	EnumValues   []string `json:"enumValues,omitempty"`
}

// ToolSchema 定义 tools/list 返回的 MCP schema。
type ToolSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema 定义工具输入 schema。
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef 定义 schema 字段。
type PropertyDef struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolRequest 定义工具调用请求。
type ToolRequest struct {
	ToolID     string
	Parameters map[string]any
}

// ToolResponse 定义工具调用响应。
type ToolResponse struct {
	Success      bool
	ToolID       string
	TextResult   string
	ErrorCode    string
	ErrorMessage string
}

// ToolExecutor 定义工具执行器接口。
type ToolExecutor interface {
	Definition() ToolDefinition
	Execute(request ToolRequest) ToolResponse
}

// Registry 管理 MCP 工具注册。
type Registry struct {
	executors map[string]ToolExecutor
}

// NewRegistry 创建默认工具注册表。
func NewRegistry() *Registry {
	registry := &Registry{
		executors: make(map[string]ToolExecutor),
	}
	for _, executor := range []ToolExecutor{
		NewWeatherTool(),
		NewTicketTool(),
		NewSalesTool(),
	} {
		registry.executors[executor.Definition().ToolID] = executor
	}
	return registry
}

// ListTools 返回全部工具定义。
func (r *Registry) ListTools() []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(r.executors))
	for _, executor := range r.executors {
		definitions = append(definitions, executor.Definition())
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].ToolID < definitions[j].ToolID
	})
	return definitions
}

// GetExecutor 返回指定工具执行器。
func (r *Registry) GetExecutor(toolID string) (ToolExecutor, bool) {
	executor, ok := r.executors[toolID]
	return executor, ok
}

// NewHTTPHandler 创建 MCP JSON-RPC HTTP 处理器。
func NewHTTPHandler() http.Handler {
	registry := NewRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, JSONRPCResponse{
				JSONRPC: jsonRPCVersion,
				Error: &JSONRPCError{
					Code:    invalidParamsCode,
					Message: "invalid request body",
				},
			})
			return
		}

		response := dispatch(registry, request)
		if response == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	return mux
}

func dispatch(registry *Registry, request JSONRPCRequest) *JSONRPCResponse {
	if request.ID == nil {
		return nil
	}

	switch request.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      request.ID,
			Result: map[string]any{
				"protocolVersion": "2026-02-28",
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]any{
					"name":    serverName,
					"version": serverVersion,
				},
			},
		}
	case "tools/list":
		schemas := make([]ToolSchema, 0, len(registry.executors))
		for _, definition := range registry.ListTools() {
			schemas = append(schemas, toToolSchema(definition))
		}
		return &JSONRPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      request.ID,
			Result: map[string]any{
				"tools": schemas,
			},
		}
	case "tools/call":
		toolName, _ := request.Params["name"].(string)
		if strings.TrimSpace(toolName) == "" {
			return errorResponse(request.ID, invalidParamsCode, "Missing 'name' in params")
		}
		executor, ok := registry.GetExecutor(toolName)
		if !ok {
			return errorResponse(request.ID, methodNotFound, "Tool not found: "+toolName)
		}
		arguments := map[string]any{}
		if rawArguments, ok := request.Params["arguments"].(map[string]any); ok {
			arguments = rawArguments
		}
		result := executor.Execute(ToolRequest{
			ToolID:     toolName,
			Parameters: arguments,
		})
		text := result.TextResult
		if text == "" && !result.Success {
			text = result.ErrorMessage
		}
		return &JSONRPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      request.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": text,
					},
				},
				"isError": !result.Success,
			},
		}
	default:
		return errorResponse(request.ID, methodNotFound, "Unknown method: "+request.Method)
	}
}

func errorResponse(id any, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

func toToolSchema(definition ToolDefinition) ToolSchema {
	properties := make(map[string]PropertyDef, len(definition.Parameters))
	required := make([]string, 0)
	for name, parameter := range definition.Parameters {
		properties[name] = PropertyDef{
			Type:        parameter.Type,
			Description: parameter.Description,
			Enum:        parameter.EnumValues,
		}
		if parameter.Required {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return ToolSchema{
		Name:        definition.ToolID,
		Description: definition.Description,
		InputSchema: InputSchema{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WeatherTool 提供天气查询工具。
type WeatherTool struct{}

// NewWeatherTool 创建天气工具。
func NewWeatherTool() *WeatherTool { return &WeatherTool{} }

// Definition 返回天气工具定义。
func (t *WeatherTool) Definition() ToolDefinition {
	return ToolDefinition{
		ToolID:      "weather_query",
		Description: "查询城市天气信息，支持当前天气和未来预报",
		Parameters: map[string]ParameterDef{
			"city": {
				Description: "城市名称，如北京、上海",
				Type:        "string",
				Required:    true,
			},
			"queryType": {
				Description:  "查询类型：current 或 forecast",
				Type:         "string",
				DefaultValue: "current",
				EnumValues:   []string{"current", "forecast"},
			},
		},
	}
}

// Execute 执行天气查询。
func (t *WeatherTool) Execute(request ToolRequest) ToolResponse {
	city := strings.TrimSpace(asString(request.Parameters["city"]))
	if city == "" {
		return ToolResponse{
			Success:      false,
			ToolID:       "weather_query",
			ErrorCode:    "INVALID_PARAMS",
			ErrorMessage: "请提供城市名称",
		}
	}
	queryType := strings.TrimSpace(asString(request.Parameters["queryType"]))
	if queryType == "" {
		queryType = "current"
	}
	now := time.Now().Format("2006-01-02")
	text := fmt.Sprintf("【%s 天气】\n日期: %s\n模式: %s\n天气: 晴\n温度: 24°C\n湿度: 55%%", city, now, queryType)
	return ToolResponse{
		Success:    true,
		ToolID:     "weather_query",
		TextResult: text,
	}
}

// TicketTool 提供工单查询工具。
type TicketTool struct{}

// NewTicketTool 创建工单工具。
func NewTicketTool() *TicketTool { return &TicketTool{} }

// Definition 返回工单工具定义。
func (t *TicketTool) Definition() ToolDefinition {
	return ToolDefinition{
		ToolID:      "ticket_query",
		Description: "查询客户技术支持工单数据，支持汇总、列表和统计分析",
		Parameters: map[string]ParameterDef{
			"region": {
				Description: "地区筛选",
				Type:        "string",
				EnumValues:  []string{"华东", "华南", "华北", "西南", "西北"},
			},
			"queryType": {
				Description:  "查询类型：summary、list、stats",
				Type:         "string",
				DefaultValue: "summary",
				EnumValues:   []string{"summary", "list", "stats"},
			},
		},
	}
}

// Execute 执行工单查询。
func (t *TicketTool) Execute(request ToolRequest) ToolResponse {
	region := strings.TrimSpace(asString(request.Parameters["region"]))
	queryType := strings.TrimSpace(asString(request.Parameters["queryType"]))
	if queryType == "" {
		queryType = "summary"
	}
	if region == "" {
		region = "全国"
	}
	text := fmt.Sprintf("【工单查询】\n地区: %s\n模式: %s\n工单总数: 42\n待处理: 6\n处理中: 9", region, queryType)
	return ToolResponse{
		Success:    true,
		ToolID:     "ticket_query",
		TextResult: text,
	}
}

// SalesTool 提供销售查询工具。
type SalesTool struct{}

// NewSalesTool 创建销售工具。
func NewSalesTool() *SalesTool { return &SalesTool{} }

// Definition 返回销售工具定义。
func (t *SalesTool) Definition() ToolDefinition {
	return ToolDefinition{
		ToolID:      "sales_query",
		Description: "查询软件销售数据，支持汇总、排名、明细和趋势分析",
		Parameters: map[string]ParameterDef{
			"region": {
				Description: "地区筛选",
				Type:        "string",
				EnumValues:  []string{"华东", "华南", "华北", "西南", "西北"},
			},
			"period": {
				Description:  "时间范围",
				Type:         "string",
				DefaultValue: "本月",
				EnumValues:   []string{"本月", "上月", "本季度", "上季度", "本年"},
			},
			"queryType": {
				Description:  "查询类型：summary、ranking、detail、trend",
				Type:         "string",
				DefaultValue: "summary",
				EnumValues:   []string{"summary", "ranking", "detail", "trend"},
			},
		},
	}
}

// Execute 执行销售查询。
func (t *SalesTool) Execute(request ToolRequest) ToolResponse {
	region := strings.TrimSpace(asString(request.Parameters["region"]))
	if region == "" {
		region = "全国"
	}
	period := strings.TrimSpace(asString(request.Parameters["period"]))
	if period == "" {
		period = "本月"
	}
	queryType := strings.TrimSpace(asString(request.Parameters["queryType"]))
	if queryType == "" {
		queryType = "summary"
	}
	text := fmt.Sprintf("【销售查询】\n时间: %s\n地区: %s\n模式: %s\n总销售额: ¥128.50 万", period, region, queryType)
	return ToolResponse{
		Success:    true,
		ToolID:     "sales_query",
		TextResult: text,
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
