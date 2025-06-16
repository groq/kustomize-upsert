# Kustomize Upsert Transformer

A generic Go plugin for Kustomize that provides "append if exists, else create" behavior for array fields in Kubernetes resources. Supports regex-based resource targeting for gradual rollouts.

## Problem Solved

Kustomize has fundamental limitations with array handling:
- **Strategic merge patches** clobber entire arrays in custom resources
- **JSON patches** fail if target paths don't exist  
- **No conditional logic** for "append if exists, else create" behavior

This transformer provides transparent array upsert operations for any Kubernetes resource and field.

## Features

- ✅ **Generic**: Works with any Kubernetes resource type and array field
- ✅ **Transparent**: Handles both "create" and "append" cases automatically
- ✅ **Type safe**: Pure Go implementation with proper error handling
- ✅ **Configurable**: Support for multiple operations and target selectors
- ✅ **Duplicate handling**: Optional deduplication of array values
- ✅ **JSONPath support**: Flexible path targeting (e.g., `spec.containers[0].args`)
- ✅ **Regex targeting**: Supports regex patterns for resource names (e.g., `name: "web-.*"`)

## Usage

### 1. Install the transformer

```bash
make build
# Binary created at ./kustomize-upsert
# Copy to your PATH or distribute via internal tooling
```

### 2. Create configuration file

```yaml
apiVersion: groq.io/v1alpha1
kind: UpsertTransformer
metadata:
  name: array-upsert
spec:
  operations:
    # Add OTEL environment variables to InferenceEngineDeployments
    - target:
        group: "inference-engine.groq.io"
        version: "v1alpha1"
        kind: "InferenceEngineDeployment"
      path: "spec.dev.nova_args"
      values:
        - "--agent-env-vars=GROQ_ENABLE_DRIVER_OTEL=1"
        - "--agent-env-vars=GROQ_QSFP_OTEL_MONITOR=1"
      allowDuplicates: false
      
    # Add debug flags using regex targeting (gradual rollout)
    - target:
        group: "apps"
        version: "v1"
        kind: "Deployment" 
        name: "(web-app|api-service|worker-.*)"
      path: "spec.template.spec.containers[0].args"
      values:
        - "--debug"
        - "--verbose"
        
    # Add args to specific pods  
    - target:
        version: "v1"
        kind: "Pod"
        name: "my-pod"
      path: "spec.containers[0].args"
      values:
        - "--feature-flag=enabled"
```

### 3. Use in kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml

transformers:
  - upsert-config.yaml
```

### 4. Build with kustomize

```bash
kustomize build --enable-alpha-plugins .
```

## Configuration Reference

### UpsertOperation Fields

| Field | Type | Description |
|-------|------|-------------|
| `target` | `ResourceSelector` | Specifies which resources to transform |
| `path` | `string` | JSONPath to the array field (e.g., `spec.dev.nova_args`) |
| `values` | `[]string` | Values to append/set in the array |
| `allowDuplicates` | `bool` | Whether to add values that already exist (default: false) |

### ResourceSelector Fields

| Field | Type | Description |
|-------|------|-------------|
| `group` | `string` | API group to match (optional) |
| `version` | `string` | API version to match (optional) |
| `kind` | `string` | Resource kind to match (optional) |
| `name` | `string` | Resource name to match - supports regex (optional) |
| `namespace` | `string` | Resource namespace to match - supports regex (optional) |

## Regex Rollout Pattern

The transformer supports a gradual rollout pattern using regex selectors:

### Phase 1: Target specific instances
```yaml
- target:
    name: "(foo-instance|bar-instance)"
    kind: "InferenceEngineDeployment"
  path: "spec.dev.nova_args"
  values: ["--new-feature=enabled"]
```

### Phase 2: Add more instances to regex
```yaml
- target:
    name: "(foo-instance|bar-instance|baz-instance|qux-.*)"
    kind: "InferenceEngineDeployment"  
  path: "spec.dev.nova_args"
  values: ["--new-feature=enabled"]
```

### Phase 3: Remove name selector (apply to all)
```yaml
- target:
    kind: "InferenceEngineDeployment"
  path: "spec.dev.nova_args" 
  values: ["--new-feature=enabled"]
```

## Examples

### OTEL Environment Variables

```yaml
operations:
  - target:
      group: "inference-engine.groq.io"
      version: "v1alpha1"
      kind: "InferenceEngineDeployment"
    path: "spec.dev.nova_args"
    values:
      - "--agent-env-vars=GROQ_ENABLE_DRIVER_OTEL=1"
      - "--agent-env-vars=GROQ_QSFP_OTEL_MONITOR=1"
```

### Container Arguments

```yaml
operations:
  - target:
      group: "apps"
      version: "v1"
      kind: "Deployment"
    path: "spec.template.spec.containers[0].args"
    values:
      - "--feature-flag=new-ui"
      - "--log-level=debug"
```

### Pod Security Context

```yaml
operations:
  - target:
      version: "v1"
      kind: "Pod"
    path: "spec.securityContext.supplementalGroups"
    values:
      - "1000"
      - "2000"
```

## Behavior Examples

### Before (no array exists)
```yaml
apiVersion: inference-engine.groq.io/v1alpha1
kind: InferenceEngineDeployment
spec:
  replicas: 2
```

### After (array created)
```yaml
apiVersion: inference-engine.groq.io/v1alpha1
kind: InferenceEngineDeployment
spec:
  replicas: 2
  dev:
    nova_args:
      - "--agent-env-vars=GROQ_ENABLE_DRIVER_OTEL=1"
      - "--agent-env-vars=GROQ_QSFP_OTEL_MONITOR=1"
```

### Before (array exists)
```yaml
spec:
  dev:
    nova_args:
      - "--log-level=info"
      - "--timeout=30s"
```

### After (values appended)
```yaml
spec:
  dev:
    nova_args:
      - "--log-level=info"                               # ✅ Preserved
      - "--timeout=30s"                                  # ✅ Preserved
      - "--agent-env-vars=GROQ_ENABLE_DRIVER_OTEL=1"    # ✅ Added
      - "--agent-env-vars=GROQ_QSFP_OTEL_MONITOR=1"     # ✅ Added
```

## Distribution

This transformer can be distributed using the same mechanism as other internal tools like `kustomize-lint`:

1. **Build releases** for multiple platforms
2. **Distribute via internal package management**
3. **Version and update** through standard internal processes
4. **Document** in internal developer guides

## Benefits over Manual Patches

- ✅ **Generic**: Not limited to specific use cases
- ✅ **Reusable**: One tool for all array upsert needs
- ✅ **Maintainable**: Single codebase vs. many manual patches
- ✅ **Reliable**: No shell scripting or external dependencies
- ✅ **Extensible**: Easy to add new features and configurations
