package main

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// UpsertTransformer provides generic "append if exists, else create" behavior for arrays
type UpsertTransformer struct {
	// Operations define the upsert operations to perform
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

// ResourceSelector specifies which resources to target
type ResourceSelector struct {
	// APIVersion of the target resource (e.g., "inference-engine.groq.io/v1alpha1")
	APIVersion string `yaml:"apiVersion,omitempty"`
	
	// Kind of the target resource (e.g., "InferenceEngineDeployment")  
	Kind string `yaml:"kind,omitempty"`
	
	// Name of the target resource (optional, if omitted applies to all matching kind)
	Name string `yaml:"name,omitempty"`
	
	// Namespace of the target resource (optional)
	Namespace string `yaml:"namespace,omitempty"`
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
	for _, item := range items {
		for _, op := range t.Operations {
			if t.matchesTarget(item, op.Target) {
				if err := t.applyUpsert(item, op); err != nil {
					return nil, fmt.Errorf("failed to apply upsert operation: %w", err)
				}
			}
		}
	}
	
	return items, nil
}

// matchesTarget checks if a resource matches the target selector
func (t *UpsertTransformer) matchesTarget(item *yaml.RNode, target ResourceSelector) bool {
	// Check API version
	if target.APIVersion != "" && item.GetApiVersion() != target.APIVersion {
		return false
	}
	
	// Check kind
	if target.Kind != "" && item.GetKind() != target.Kind {
		return false
	}
	
	// Check name
	if target.Name != "" && item.GetName() != target.Name {
		return false
	}
	
	// Check namespace  
	if target.Namespace != "" && item.GetNamespace() != target.Namespace {
		return false
	}
	
	return true
}

// applyUpsert performs the array upsert operation
func (t *UpsertTransformer) applyUpsert(item *yaml.RNode, op UpsertOperation) error {
	// Parse the path into components
	pathComponents := strings.Split(op.Path, ".")
	
	// Navigate to the parent of the target array
	current := item
	var err error
	
	// Navigate to parent path, creating nodes as needed
	for i, component := range pathComponents[:len(pathComponents)-1] {
		current, err = current.Pipe(yaml.LookupCreate(yaml.MappingNode, component))
		if err != nil {
			return fmt.Errorf("failed to navigate to %s (component %d: %s): %w", 
				op.Path, i, component, err)
		}
	}
	
	// Get or create the target array
	arrayFieldName := pathComponents[len(pathComponents)-1]
	arrayNode, err := current.Pipe(yaml.LookupCreate(yaml.SequenceNode, arrayFieldName))
	if err != nil {
		return fmt.Errorf("failed to create array field %s: %w", arrayFieldName, err)
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
			if err := arrayNode.PipeE(yaml.Append(yaml.NewStringRNode(value))); err != nil {
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
