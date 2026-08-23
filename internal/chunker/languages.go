package chunker

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
)

// languageHandler bundles a tree-sitter language and the node types considered
// chunk boundaries for that language.
type languageHandler struct {
	name        string
	lang        *sitter.Language
	symbolTypes []string
	calls       callConfig
	kindOf      kindConfig
}

// buildHandlers returns the supported language parsers.
func buildHandlers() map[string]languageHandler {
	return map[string]languageHandler{
		"go": {
			name: "go",
			lang: golang.GetLanguage(),
			symbolTypes: []string{
				"function_declaration",
				"method_declaration",
				"type_declaration",
			},
			calls: callConfig{
				callTypes:   []string{"call_expression"},
				memberTypes: []string{"selector_expression", "scoped_identifier"},
			},
			kindOf: kindConfig{
				"function_declaration": "function",
				"method_declaration":   "method",
				"type_declaration":     "type",
			},
		},
		"python": {
			name: "python",
			lang: python.GetLanguage(),
			symbolTypes: []string{
				"function_definition",
				"class_definition",
			},
			calls: callConfig{
				callTypes:   []string{"call"},
				memberTypes: []string{"attribute"},
			},
			kindOf: kindConfig{
				"function_definition": "function",
				"class_definition":    "type",
			},
		},
		"javascript": {
			name: "javascript",
			lang: javascript.GetLanguage(),
			symbolTypes: []string{
				"function_declaration",
				"class_declaration",
				"method_definition",
			},
			calls: callConfig{
				callTypes:   []string{"call_expression"},
				memberTypes: []string{"member_expression"},
			},
			kindOf: kindConfig{
				"function_declaration": "function",
				"class_declaration":    "type",
				"method_definition":    "method",
			},
		},
		"typescript": {
			name: "typescript",
			lang: javascript.GetLanguage(),
			symbolTypes: []string{
				"function_declaration",
				"class_declaration",
				"method_definition",
				"export_statement",
			},
			calls: callConfig{
				callTypes:   []string{"call_expression"},
				memberTypes: []string{"member_expression"},
			},
			kindOf: kindConfig{
				"function_declaration": "function",
				"class_declaration":    "type",
				"method_definition":    "method",
				"export_statement":     "function",
			},
		},
		"rust": {
			name: "rust",
			lang: rust.GetLanguage(),
			symbolTypes: []string{
				"function_item",
				"impl_item",
				"trait_item",
				"struct_item",
				"enum_item",
			},
			calls: callConfig{
				callTypes:   []string{"call_expression"},
				memberTypes: []string{"field_expression", "scoped_identifier"},
			},
			kindOf: kindConfig{
				"function_item": "function",
				"impl_item":     "type",
				"trait_item":    "type",
				"struct_item":   "type",
				"enum_item":     "type",
			},
		},
	}
}
