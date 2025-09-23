# Arboreal

Arboreal is a Go framework for building agentic AI systems using behavior trees and LLM integration. It provides a structured way to create complex AI workflows with planning, execution, and state management capabilities.

## Features

- **Behavior Trees**: Define complex AI behaviors using composable behavior trees
- **LLM Integration**: Built-in support for OpenAI, Anthropic, and Ollama models
- **Planning & Execution**: TodoListExecutive for autonomous task planning and execution
- **State Management**: Persistent state handling with snapshots and memory
- **Vector Search**: SQLite-vec integration for semantic search and retrieval
- **Lua Scripting**: Extensible runtime with Lua scripting support
- **Annotation System**: Rich message annotation and templating

## Quick Start

### Installation

```bash
go get github.com/zenful-ai/arboreal
```

### Basic Example

```go
package main

import "github.com/zenful-ai/arboreal"

func main() {
    // Create a behavior tree for a chat bot
    chatBehavior := arboreal.CreateBehaviorTree(
        "chat_behavior",
        "A conversational bot",
        "<insert user's input here>",
    )

    // Define states
    chatState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{})
    pauseState := arboreal.PauseState("Let user respond")

    // Add transitions
    chatBehavior.AddTransition(&chatState, &pauseState)

    // Create executive
    exec := arboreal.CreateTodoListExecutive(
        "Chat Bot",
        "Simple chat bot",
        &chatBehavior,
    )

    // Run the bot
    err := exec.RunLoop(arboreal.TerminalChannel{})
    if err != nil {
        panic(err)
    }
}
```

## Core Concepts

### Behavior Trees

Behavior trees define the logic flow of your AI agent. They consist of states and transitions:

```go
behavior := arboreal.CreateBehaviorTree("name", "description", "example")

// Add states
state1 := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
    System: "You are a helpful assistant",
})
state2 := arboreal.PauseState("Wait for user input")

// Connect states
behavior.AddTransition(&state1, &state2)
```

### TodoListExecutive

The TodoListExecutive provides autonomous planning and execution capabilities:

```go
exec := arboreal.CreateTodoListExecutive("Agent Name", "Description", behaviors...)
exec.MaxPlanDepth = 5  // Configure planning depth
exec.Preamble = "You are a helpful AI assistant"
```

### LLM Integration

Arboreal supports multiple LLM providers through environment variables:

```bash
export OPENAI_TOKEN=your_openai_key
export ANTHROPIC_TOKEN=your_anthropic_key
export OLLAMA_SERVICE_URL=http://localhost:11434
```

## Advanced Features

### Memory and Persistence

```go
// Save agent state
snapshot := arboreal.CreateSnapshot(behaviors...)
data := snapshot.Serialize()

// Restore agent state
snapshot, err := arboreal.DeserializeSnapshot(data)
```

### Vector Search and RAG

```go
// Semantic chunking and storage
chunks := arboreal.SemanticChunk(content, maxTokens)
// Store in vector database for retrieval
```

### Custom Channels

Implement custom communication channels:

```go
type CustomChannel struct {
    // Your implementation
}

func (c CustomChannel) Send(message string) error {
    // Send message through your channel
    return nil
}

func (c CustomChannel) Receive() (string, error) {
    // Receive message from your channel
    return "", nil
}
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `OPENAI_TOKEN` | OpenAI API key | For OpenAI models |
| `ANTHROPIC_TOKEN` | Anthropic API key | For Claude models |
| `OLLAMA_SERVICE_URL` | Ollama service URL | For local models |
| `TWILIO_ACCOUNT_SID` | Twilio account SID | For SMS channel |
| `TWILIO_AUTH_TOKEN` | Twilio auth token | For SMS channel |

### Model Configuration

```go
// Configure LLM options
options := arboreal.LLMCompletionOptions{
    Model:       llm.GPT4o,  // or llm.Claude3Sonnet, etc.
    Temperature: 0.7,
    MaxTokens:   1000,
    System:      "System prompt",
}
```

## Examples

The `examples/` directory contains various use cases:

- **Chat Bot** (`examples/test/`) - Basic conversational agent
- **CRM Assistant** (`examples/crm/`) - Customer relationship management

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Project Structure

```
├── README.md           # This file
├── go.mod             # Go module definition
├── *.go               # Core framework files
├── llm/               # LLM provider integrations
├── examples/          # Example applications
├── engine/            # Lua scripting engine (see engine/README.md)
└── util/              # Utility functions
```

## License

This project is licensed under the BSD License - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Vector store re-implementation
- [ ] More LLM provider integrations
- [ ] Enhanced debugging tools
- [ ] Better documentation
- [ ] Enhanced testing coverage
- [ ] Performance optimizations

## Support

- **Issues**: [GitHub Issues](https://github.com/zenful-ai/arboreal/issues)
- **Discussions**: [GitHub Discussions](https://github.com/zenful-ai/arboreal/discussions)
