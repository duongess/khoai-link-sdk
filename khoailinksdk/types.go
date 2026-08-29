package khoailinksdk

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/duongess/khoai-link-protocol/core"
)

// Config chứa toàn bộ thông số cấu hình của một Node
type Config struct {
	NodeName      string                `json:"node_name"`
	NodeID        string                `json:"node_id"`
	MCPGatewayURL string                `json:"mcp_gateway_url"`
	Tasks         []core.TaskDefinition `json:"tasks"`
	Metadata      any                   `json:"metadata,omitempty"`
}

var (
	regexIdentifier   = regexp.MustCompile(`^[a-z0-9_\-]+$`)
	regexSemanticType = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9_\-]+)+$`)
)

// SupportedTypes danh sach cac kieu du lieu co ban hop le
var supportedDataTypes = map[string]struct{}{
	"string":  {},
	"number":  {},
	"boolean": {},
	"json":    {},
}

func (c *Config) Validate() error {
	// 1. Kiem tra NodeID
	if strings.TrimSpace(c.NodeID) == "" {
		return errors.New("config error: 'node_id' cannot be blank")
	}
	if !regexIdentifier.MatchString(c.NodeID) {
		return fmt.Errorf("config error: 'node_id' (%s) contains invalid characters (allowed: a-z, 0-9, '-', '_')", c.NodeID)
	}

	// 2. Kiem tra MCPGatewayURL
	if strings.TrimSpace(c.MCPGatewayURL) == "" {
		return errors.New("config error: 'mcp_gateway_url' cannot be blank")
	}
	parsedURL, err := url.ParseRequestURI(c.MCPGatewayURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("config error: 'mcp_gateway_url' (%s) must be a valid HTTP/HTTPS URL", c.MCPGatewayURL)
	}

	// 3. Kiem tra danh sach Tasks
	if len(c.Tasks) == 0 {
		return errors.New("config error: 'tasks' array must contain at least 1 task")
	}

	taskNameSet := make(map[string]struct{})

	for i, task := range c.Tasks {
		// Validate Task Metadata
		if strings.TrimSpace(task.Name) == "" {
			return fmt.Errorf("config error: task[%d] missing 'name'", i)
		}
		if !regexIdentifier.MatchString(task.Name) {
			return fmt.Errorf("config error: task[%d] name '%s' contains invalid characters", i, task.Name)
		}
		if _, exists := taskNameSet[task.Name]; exists {
			return fmt.Errorf("config error: duplicate task name '%s' detected", task.Name)
		}
		taskNameSet[task.Name] = struct{}{}

		if strings.TrimSpace(task.Domain) == "" {
			return fmt.Errorf("config error: task '%s' missing 'domain'", task.Name)
		}
		if strings.TrimSpace(task.Intent) == "" {
			return fmt.Errorf("config error: task '%s' missing 'intent'", task.Name)
		}

		// Validate InputSchema (map[string]ParameterDef)
		for paramName, paramDef := range task.InputSchema {
			if strings.TrimSpace(paramName) == "" {
				return fmt.Errorf("config error: task '%s' has blank input parameter key", task.Name)
			}
			if _, ok := supportedDataTypes[paramDef.Type]; !ok {
				return fmt.Errorf("config error: task '%s', input '%s' has unsupported type '%s'", task.Name, paramName, paramDef.Type)
			}
			if !regexSemanticType.MatchString(paramDef.SemanticType) {
				return fmt.Errorf("config error: task '%s', input '%s' has invalid semantic_type '%s' (must follow dot.notation, e.g. production.lot.id)", task.Name, paramName, paramDef.SemanticType)
			}
		}

		// Validate OutputSchema (map[string]OutputFieldDef)
		for fieldName, fieldDef := range task.OutputSchema {
			if strings.TrimSpace(fieldName) == "" {
				return fmt.Errorf("config error: task '%s' has blank output field key", task.Name)
			}
			if _, ok := supportedDataTypes[fieldDef.Type]; !ok {
				return fmt.Errorf("config error: task '%s', output '%s' has unsupported type '%s'", task.Name, fieldName, fieldDef.Type)
			}
			if !regexSemanticType.MatchString(fieldDef.SemanticType) {
				return fmt.Errorf("config error: task '%s', output '%s' has invalid semantic_type '%s' (must follow dot.notation, e.g. quality.defect.rate)", task.Name, fieldName, fieldDef.SemanticType)
			}
		}
	}

	return nil
}
