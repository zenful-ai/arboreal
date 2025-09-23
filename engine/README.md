# Arboreal Engine

The Arboreal Engine is a Lua scripting runtime that provides a high-level interface for building and executing behavior trees using the Arboreal framework. It allows developers to define complex AI behaviors through Lua scripts while leveraging the full power of the underlying Go implementation.

## Overview

The engine acts as a bridge between Lua scripts and the core Arboreal behavior tree system, providing:

- **Lua Runtime**: Full Lua scripting environment with Go integration
- **Behavior Tree API**: High-level Lua functions for creating and managing behavior trees
- **LLM Integration**: Direct access to language model completion capabilities
- **State Management**: Lua-accessible behavior state creation and management
- **Planning System**: Integration with Arboreal's planning capabilities
- **Message Annotation**: Rich message templating and annotation system

## Core Components

### lua_arboreal.go
Main namespace providing core Arboreal functionality:
- `arboreal.state()` - Create new behavior states
- `arboreal.tree()` - Create behavior trees
- `arboreal.llm_complete()` - LLM completion states
- `arboreal.planner()` - Planning system integration

### lua_behavior.go
Behavior interface for working with individual behaviors:
- Behavior introspection (name, description)
- Behavior execution (`call()`)
- Behavior copying and manipulation

### lua_glue.go
Core glue code that binds Lua to Go:
- Type export system for Go types
- HTTP client integration
- JSON support
- Runtime initialization

### lua_annotation.go & lua_annotated_messages.go
Message annotation system for templating and rich message handling.

### lua_planner.go & lua_signal.go
Planning and signaling system integration for complex workflows.

## Usage

### Basic Example

```lua
-- Create a simple chat behavior tree
local tree = arboreal.tree("chat", "chat with the user", "hi there")

-- Add an LLM completion state
local llm = arboreal.llm_complete({system="Respond only in Haikus"})
tree:add(llm)

-- Set as entry point
arboreal.entry = tree
```

### Examples Directory

The `examples/` directory contains various Lua scripts demonstrating engine capabilities:

- `haiku.lua` - Simple haiku generator
- `poetry_planner.lua` - Complex planning example
- `basic_annotation_test.lua` - Message annotation usage
- `eval_test.lua` - Evaluation examples
- `graph_test.lua` - Behavior tree graph construction
- `request_test.lua` - HTTP request handling

## Binary

The `engine` binary is a standalone executable that can run Lua scripts using the Arboreal engine runtime. Use it to execute behavior tree definitions written in Lua.

## Integration

The engine is designed to be embedded within larger Arboreal applications or used standalone for scripting complex AI behaviors. It provides a more accessible interface to the powerful behavior tree system while maintaining full access to the underlying Go implementation's capabilities.