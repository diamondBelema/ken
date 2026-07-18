package parser

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func SplitFrontmatter(data []byte) (map[string]interface{}, string, error) {
	if !bytes.HasPrefix(data, []byte("---")) {
		return nil, "", fmt.Errorf("no YAML frontmatter found (file must start with ---)")
	}

	rest := data[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	
	var yamlBytes []byte
	var body string
	
	if idx == -1 {
		yamlBytes = rest
		body = ""
	} else {
		yamlBytes = rest[:idx]
		body = string(rest[idx+4:])
		if strings.HasPrefix(body, "\n") {
			body = body[1:]
		}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return raw, body, nil
}
