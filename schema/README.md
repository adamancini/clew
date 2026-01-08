# Clewfile JSON Schema

This directory contains the JSON Schema definition for Clewfile format validation and IDE support.

## IDE Integration

### YAML Files

Add the `$schema` reference at the top of your YAML Clewfile:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/adamancini/clew/main/schema/clewfile.schema.json
---
version: 1

marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official

plugins:
  - context7@claude-plugins-official
```

Or use a local path during development:

```yaml
# yaml-language-server: $schema=../schema/clewfile.schema.json
---
version: 1
```

**VS Code**: Install the [YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)

**Neovim**: Use [yaml-language-server](https://github.com/redhat-developer/yaml-language-server) with your LSP configuration

### JSON Files

Add the `$schema` property:

```json
{
  "$schema": "https://raw.githubusercontent.com/adamancini/clew/main/schema/clewfile.schema.json",
  "version": 1,
  "marketplaces": {
    "claude-plugins-official": {
      "source": "github",
      "repo": "anthropics/claude-plugins-official"
    }
  }
}
```

**VS Code**: Built-in support, no extension needed

### TOML Files

TOML schema support varies by IDE:

**VS Code with Even Better TOML**:
```toml
# No schema support yet, but syntax highlighting works
version = 1

[marketplaces.claude-plugins-official]
source = "github"
repo = "anthropics/claude-plugins-official"
```

**Taplo** (Rust-based TOML toolkit) has experimental JSON Schema support.

## Features Provided by Schema

When properly configured, your IDE will provide:

✅ **Auto-completion**
- Marketplace source types (`github`, `local`)
- Plugin scope options (`user`, `project`)
- MCP transport types (`stdio`, `http`)
- Field names and structure

✅ **Validation**
- Required fields (e.g., `version`, `source`, `transport`)
- Conditional requirements (e.g., `repo` required when `source: github`)
- Pattern matching (e.g., `plugin@marketplace` format)
- Enum validation (e.g., valid transport types)

✅ **Documentation**
- Hover tooltips for each field
- Inline examples
- Format descriptions

✅ **Error Detection**
- Typos in field names
- Missing required fields
- Invalid values
- Format violations

## Publishing to Schema Store

Once the schema is stable, it can be submitted to [JSON Schema Store](https://www.schemastore.org/) for automatic IDE recognition without requiring explicit `$schema` references.

This would enable validation for:
- `Clewfile`, `Clewfile.yaml`, `Clewfile.yml`, `Clewfile.json`
- Files matching these patterns in standard locations

## Schema Maintenance

When adding new features to clew:

1. Update the Go struct definitions in `internal/config/config.go`
2. Update the JSON Schema in `schema/clewfile.schema.json`
3. Update examples in `schema/examples/`
4. Bump the schema version if breaking changes occur

## Validation Testing

Validate a Clewfile against the schema using any JSON Schema validator:

```bash
# Using ajv-cli (Node.js)
npm install -g ajv-cli
ajv validate -s schema/clewfile.schema.json -d ~/.config/claude/Clewfile.yaml

# Using check-jsonschema (Python)
pip install check-jsonschema
check-jsonschema --schemafile schema/clewfile.schema.json ~/.config/claude/Clewfile.yaml
```

## Future Enhancements

Potential schema improvements:

- [ ] Version-specific schemas (`clewfile.v1.schema.json`, `clewfile.v2.schema.json`)
- [ ] Conditional validation for plugin names based on available marketplaces
- [ ] Pattern validation for environment variable syntax (`${VAR:-default}`)
- [ ] Integration with clew CLI for validation command (`clew validate`)
