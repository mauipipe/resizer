package hcl

import (
	"fmt"

	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/hashicorp/hcl/hcl/ast"
	hclParser "github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/hashicorp/hcl/hcl/parser"
	jsonParser "github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/hashicorp/hcl/json/parser"
)

// Parse parses the given input and returns the root object.
//
// The input format can be either HCL or JSON.
func Parse(input string) (*ast.File, error) {
	switch lexMode(input) {
	case lexModeHcl:
		return hclParser.Parse([]byte(input))
	case lexModeJson:
		return jsonParser.Parse([]byte(input))
	}

	return nil, fmt.Errorf("unknown config format")
}
