package chunker

import sitter "github.com/smacker/go-tree-sitter"

// callConfig describes how calls look in a language's grammar.
type callConfig struct {
	callTypes   []string // AST node types representing a call
	memberTypes []string // AST node types for qualified calls (x.y())
}

// kindConfig maps AST node types to symbol kinds.
type kindConfig map[string]string

// walkCalls recursively descends an AST node and collects all call references.
func walkCalls(node *sitter.Node, content []byte, cfg callConfig) []CallRef {
	if node == nil {
		return nil
	}

	var calls []CallRef
	walkCallsRecursive(node, content, cfg, &calls)
	if len(calls) == 0 {
		return nil
	}
	return calls
}

func walkCallsRecursive(node *sitter.Node, content []byte, cfg callConfig, out *[]CallRef) {
	if node == nil {
		return
	}

	if isCallNode(node, cfg.callTypes) {
		fnNode := node.ChildByFieldName("function")
		if fnNode != nil {
			name := calleeName(fnNode, content, cfg)
			if name != "" {
				*out = append(*out, CallRef{
					Name: name,
					Line: int(fnNode.StartPoint().Row) + 1,
				})
			}
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		walkCallsRecursive(node.Child(i), content, cfg, out)
	}
}

func isCallNode(node *sitter.Node, callTypes []string) bool {
	nt := node.Type()
	for _, t := range callTypes {
		if nt == t {
			return true
		}
	}
	return false
}

// calleeName resolves the name of a call's function node.
// For qualified calls (x.y()), only the field part is returned (e.g. "y").
func calleeName(fn *sitter.Node, content []byte, cfg callConfig) string {
	nt := fn.Type()

	switch nt {
	case "identifier", "field_identifier", "type_identifier":
		return string(fn.Content(content))
	}

	for _, mt := range cfg.memberTypes {
		if nt == mt {
			return extractFieldPart(fn, content)
		}
	}

	if nt == "scoped_identifier" {
		return extractLastSegment(fn, content)
	}

	return ""
}

func extractFieldPart(fn *sitter.Node, content []byte) string {
	for _, field := range []string{"field", "property", "attribute"} {
		if child := fn.ChildByFieldName(field); child != nil {
			return string(child.Content(content))
		}
	}

	for i := 0; i < int(fn.ChildCount()); i++ {
		child := fn.Child(i)
		if child == nil {
			continue
		}
		t := child.Type()
		if t == "identifier" || t == "field_identifier" || t == "property_identifier" {
			return string(child.Content(content))
		}
	}

	return ""
}

func extractLastSegment(fn *sitter.Node, content []byte) string {
	var last string
	for i := 0; i < int(fn.ChildCount()); i++ {
		child := fn.Child(i)
		if child == nil {
			continue
		}
		t := child.Type()
		if t == "identifier" || t == "field_identifier" {
			last = string(child.Content(content))
		}
	}
	return last
}
