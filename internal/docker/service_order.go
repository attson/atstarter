package docker

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

var defaultComposeFiles = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

var defaultComposeOverrideFiles = []string{
	"compose.override.yml",
	"compose.override.yaml",
	"docker-compose.override.yml",
	"docker-compose.override.yaml",
}

// orderServiceNames keeps Docker's effective service set while restoring the
// declaration order lost by Compose versions that emit map entries randomly.
func orderServiceNames(dir, file string, names []string) []string {
	available := make(map[string]struct{}, len(names))
	for _, name := range names {
		available[name] = struct{}{}
	}

	ordered := make([]string, 0, len(available))
	seen := make(map[string]struct{}, len(available))
	for _, path := range composeDefinitionFiles(dir, file) {
		for _, name := range serviceNamesInFile(path) {
			if _, ok := available[name]; !ok {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			ordered = append(ordered, name)
			seen[name] = struct{}{}
		}
	}

	remaining := make([]string, 0, len(available)-len(ordered))
	for name := range available {
		if _, ok := seen[name]; !ok {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	return append(ordered, remaining...)
}

func composeDefinitionFiles(dir, file string) []string {
	if file != "" {
		if filepath.IsAbs(file) {
			return []string{file}
		}
		return []string{filepath.Join(dir, file)}
	}

	files := make([]string, 0, 2)
	if base := firstExistingComposeFile(dir, defaultComposeFiles); base != "" {
		files = append(files, base)
	}
	if override := firstExistingComposeFile(dir, defaultComposeOverrideFiles); override != "" {
		files = append(files, override)
	}
	return files
}

func firstExistingComposeFile(dir string, candidates []string) string {
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func serviceNamesInFile(path string) []string {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil || len(document.Content) == 0 {
		return nil
	}
	root := dereferenceAlias(document.Content[0])
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "services" {
			continue
		}
		services := dereferenceAlias(root.Content[i+1])
		if services == nil || services.Kind != yaml.MappingNode {
			return nil
		}
		result := make([]string, 0, len(services.Content)/2)
		for j := 0; j+1 < len(services.Content); j += 2 {
			result = append(result, services.Content[j].Value)
		}
		return result
	}
	return nil
}

func dereferenceAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}
