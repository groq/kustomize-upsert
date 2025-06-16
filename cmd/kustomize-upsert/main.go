package main

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	goyaml "gopkg.in/yaml.v3"
)

// UpsertTransformer provides generic "append if exists, else create" behavior for arrays
type UpsertTransformer struct {
	// Spec contains the transformer specification
	Spec UpsertTransformerSpec `yaml:"spec,omitempty"`
	
	// Operations define the upsert operations to perform (for backward compatibility)
	Operations []UpsertOperation `yaml:"operations,omitempty"`
}

// UpsertTransformerSpec contains the operations specification
type UpsertTransformerSpec struct {
	Operations []UpsertOperation `yaml:"operations,omitempty"`
}

// UpsertOperation defines a single array upsert operation
type UpsertOperation struct {
	// Target specifies which resources to transform
	Target ResourceSelector `yaml:"target"`
	
	// Path is the JSONPath to the array field (e.g., "spec.dev.nova_args")
	Path string `yaml:"path"`
	
	// Values are the strings to append/set in the array
	Values []string `yaml:"values"`
	
	// AllowDuplicates controls whether to add values that already exist (default: false)
	AllowDuplicates bool `yaml:"allowDuplicates,omitempty"`
}

// ResourceSelector specifies which resources to target (uses kustomize's Selector)
type ResourceSelector struct {
	types.Selector `yaml:",inline"`
}

func main() {
	transformer := &UpsertTransformer{}
	
	cmd := command.Build(&framework.SimpleProcessor{
		Config: transformer,
		Filter: transformer,
	}, command.StandaloneDisabled, false)
	
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Filter implements the transformation logic
func (t *UpsertTransformer) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	// Get operations from either spec or direct field
	operations := t.Operations
	if len(operations) == 0 && len(t.Spec.Operations) > 0 {
		operations = t.Spec.Operations
	}
	
	// If no operations loaded by framework, try manual parsing
	if len(operations) == 0 {
		if configStr := os.Getenv("KUSTOMIZE_PLUGIN_CONFIG_STRING"); configStr != "" {
			var config UpsertTransformer
			if err := goyaml.Unmarshal([]byte(configStr), &config); err != nil {
				return nil, fmt.Errorf("failed to parse config: %w", err)
			}
			operations = config.Operations
			if len(operations) == 0 && len(config.Spec.Operations) > 0 {
				operations = config.Spec.Operations
			}
		}
	}
	

	
	for _, item := range items {
		for _, op := range operations {
			match, err := t.matchesTarget(item, op.Target)
			if err != nil {
				return nil, fmt.Errorf("failed to match target: %w", err)
			}
			if match {
				if err := t.applyUpsert(item, op); err != nil {
					return nil, fmt.Errorf("failed to apply upsert operation: %w", err)
				}
			}
		}
	}
	
	return items, nil
}

// matchesTarget checks if a resource matches the target selector using kustomize's regex selector logic
func (t *UpsertTransformer) matchesTarget(item *yaml.RNode, target ResourceSelector) (bool, error) {
	// Convert the RNode to a ResId
	resourceId := resid.FromRNode(item)
	
	// Create a SelectorRegex from the target to get regex support
	selectorRegex, err := types.NewSelectorRegex(&target.Selector)
	if err != nil {
		return false, fmt.Errorf("failed to create selector regex: %w", err)
	}
	
	// Use kustomize's built-in regex selection logic
	if !selectorRegex.MatchGvk(resourceId.Gvk) {
		return false, nil
	}
	
	if !selectorRegex.MatchName(resourceId.Name) {
		return false, nil
	}
	
	if !selectorRegex.MatchNamespace(resourceId.Namespace) {
		return false, nil
	}
	
	return true, nil
}

// parseJSONPath converts a JSONPath string to kustomize path components
func (t *UpsertTransformer) parseJSONPath(path string) []string {
	// Split on dots and handle array indexing
	parts := strings.Split(path, ".")
	var result []string
	
	for _, part := range parts {
		// Handle array indexing like "containers[0]" -> ["containers", "0"]
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			// Extract field name and index
			fieldName := part[:strings.Index(part, "[")]
			indexPart := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			
			result = append(result, fieldName)
			result = append(result, indexPart)
		} else {
			result = append(result, part)
		}
	}
	
	return result
}

// applyUpsert performs the array upsert operation
func (t *UpsertTransformer) applyUpsert(item *yaml.RNode, op UpsertOperation) error {
	// Parse the JSONPath using kustomize's path handling
	pathParts := t.parseJSONPath(op.Path)
	
	// Use kustomize's LookupCreate to navigate/create the target array
	arrayNode, err := item.Pipe(yaml.LookupCreate(yaml.SequenceNode, pathParts...))
	if err != nil {
		return fmt.Errorf("failed to create/lookup array at path %s: %w", op.Path, err)
	}
	
	// Get existing values to check for duplicates
	var existingValues []string
	if !op.AllowDuplicates {
		existingValues, err = t.getArrayValues(arrayNode)
		if err != nil {
			return fmt.Errorf("failed to read existing array values: %w", err)
		}
	}
	
	// Add each value if not already present (or if duplicates allowed)
	for _, value := range op.Values {
		if op.AllowDuplicates || !contains(existingValues, value) {
			stringNode := yaml.NewScalarRNode(value)
			if err := arrayNode.PipeE(yaml.Append(stringNode.YNode())); err != nil {
				return fmt.Errorf("failed to append %s to %s: %w", value, op.Path, err)
			}
		}
	}
	
	return nil
}

// getArrayValues extracts string values from an array node
func (t *UpsertTransformer) getArrayValues(arrayNode *yaml.RNode) ([]string, error) {
	var values []string
	
	if arrayNode == nil {
		return values, nil
	}
	
	elements, err := arrayNode.Elements()
	if err != nil {
		return nil, err
	}
	
	for _, element := range elements {
		values = append(values, element.YNode().Value)
	}
	
	return values, nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
