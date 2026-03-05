package arboreal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"math/big"
	insecureRand "math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/zenful-ai/arboreal/llm"
)

var seedRNGOnce sync.Once

func MonotonicIdGenerator(prefix string) func() string {
	var nextId uint64 = 0
	return func() string {
		id := atomic.AddUint64(&nextId, 1)
		return fmt.Sprintf("%s%d", prefix, id)
	}
}

// GenerateStringIdentifier generates a random string of a given length and prefix.
func GenerateStringIdentifier(prefix string, length int) (string, error) {
	var err error
	var buf bytes.Buffer
	var n = 0

	encoding := base32.NewEncoding("0123456789abcdefghijklmnopqrstuv").WithPadding(base32.NoPadding)
	encoder := base32.NewEncoder(encoding, &buf)

	// We'll generate at most 8 bytes at a time (maximum value of an int64)
	mx := big.NewInt(9223372036854775807)

	// When running in a test environment, it is sometimes valuable to have deterministic pseudo-random numbers
	var useInsecureRNG bool
	if s := os.Getenv("ZEN_SEED_RNG"); s != "" {
		seedRNGOnce.Do(func() {
			randomSeed, err := strconv.Atoi(s)
			if err != nil {
				return
			}

			insecureRand.Seed(int64(randomSeed))
		})
		useInsecureRNG = true
	}

	// Ensure we generate at least n bytes
	for n < length {
		var randomNumber *big.Int

		if useInsecureRNG {
			randomNumber = big.NewInt(int64(insecureRand.Int()))
		} else {
			randomNumber, err = rand.Int(rand.Reader, mx)
			if err != nil {
				return "", err
			}
		}

		written, err := encoder.Write(randomNumber.Bytes())
		if err != nil {
			return "", err
		}

		n += written
	}

	// Must close the encoder when finished to flush any partial blocks.
	// If you comment out the following line, the last partial block "r"
	// won't be encoded.
	err = encoder.Close()
	if err != nil {
		return "", err
	}

	// Trim the buffer to the correct length
	result := buf.String()[:length-len(prefix)]

	return prefix + result, nil
}

type BehaviorState struct {
	StateName        string
	StateDescription string
	HashId           string
	ClientID         string
	Lambda           func(ctx context.Context, history AnnotatedMessages) (AnnotatedMessages, Signal)
}

func (b *BehaviorState) Hash() string {
	return b.HashId
}

func (b *BehaviorState) Name() string {
	return b.StateName
}

func (b *BehaviorState) Description() string {
	return b.StateDescription
}

func (b *BehaviorState) Call(ctx context.Context, history AnnotatedMessages) (AnnotatedMessages, Signal) {
	var trace Trace
	var isTracing bool

	historySize := len(history)

	// Check for a trace channel in the context
	if ctx != nil && ctx.Value("arboreal_trace") != nil {
		isTracing = true
		trace, _ = ctx.Value("arboreal_trace").(Trace)
	}

	telem := TraceTelemetry{
		Start: time.Now(),
	}

	trace.Send(&TraceMessage{
		Type:     TraceMessageTypeCallBegin,
		ID:       b.Hash(),
		ClientID: b.ClientID,
		Name:     b.Name(),
		Message:  "entering custom state",
		Telemetry: &TraceTelemetry{
			Start: telem.Start,
		},
	})

	m, s := b.Lambda(ctx, history)

	var err error
	if e, ok := s.(*ErrorSignal); ok {
		err = e
	}

	now := time.Now()
	telem.End = &now

	var ops []*TraceHistoryOperation
	if isTracing {
		a := m.GetAnnotation("__trace_annotations")
		if a != nil {
			s, ok := a.Data.(string)
			if ok {
				annotation_names := strings.Split(s, ",")
				for _, name := range annotation_names {
					x := m.GetAnnotation(name)
					if x != nil {
						ops = append(ops, &TraceHistoryOperation{
							Type:       "annotation",
							Action:     "add",
							Annotation: x,
						})
					}
				}
			}
		}

		// Detect history adds
		if len(m) > historySize {
			for _, message := range history[historySize:] {
				ops = append(ops, &TraceHistoryOperation{
					Type:    "history",
					Action:  "add",
					Message: &message.ChatCompletionMessage,
				})
			}
		}
	}

	// Clear __trace_annotations
	delete(m.LastMessage().Annotations, "__trace_annotations")

	trace.Send(&TraceMessage{
		Type:       TraceMessageTypeCallEnd,
		ID:         b.Hash(),
		ClientID:   b.ClientID,
		Name:       b.Name(),
		Error:      err,
		Message:    "leaving custom state",
		Telemetry:  &telem,
		Operations: ops,
		Signal:     TraceForSignal(s),
	})

	return m, s
}

func (b *BehaviorState) Copy() Behavior {
	var s BehaviorState

	s.StateName = b.StateName
	s.StateDescription = b.StateDescription
	s.HashId = b.HashId
	s.Lambda = b.Lambda
	s.ClientID = b.ClientID

	return &s
}

type LLMCompletionOptions struct {
	Name         string
	Description  string
	ClientID     string
	Id           string
	System       string
	Model        string
	ExtraContext []string
	Annotation   string
	Terminal     bool
	AllowTools   bool
}

func CannedResponseState(message string) *BehaviorState {
	id, _ := GenerateStringIdentifier("id-", 16)
	return &BehaviorState{
		HashId: id,
		Lambda: func(ctx context.Context, history AnnotatedMessages) (AnnotatedMessages, Signal) {
			return append(history, AnnotatedMessage{
				ChatCompletionMessage: llm.ChatCompletionMessage{
					Role:    llm.ChatMessageRoleAssistant,
					Content: message,
				},
			}), nil
		},
	}
}

func evalIntoAnnotation(history AnnotatedMessages, options LLMCompletionOptions) (AnnotatedMessages, Signal) {
	provider, err := llm.CreateModelProvider(options.Model, llm.ProviderOpenAI)
	if err != nil {
		return history, &ErrorSignal{
			ErrorMessage: err.Error(),
			ErrorType:    StateErrorTypeUnrecoverable,
		}
	}
	ctx := context.Background()

	system := options.System
	if len(options.ExtraContext) > 0 {
		system += "\n\nExtra Context:\n\n"
		for _, extraContext := range options.ExtraContext {
			if a := history.GetAnnotation(extraContext); a != nil {
				system += fmt.Sprintf("%v\n\n", a.Data)
			}
		}
	}

	truncatedHistory := AnnotatedMessages{
		{
			ChatCompletionMessage: llm.ChatCompletionMessage{
				Role:    llm.ChatMessageRoleSystem,
				Content: system,
			},
		},
		{
			ChatCompletionMessage: llm.ChatCompletionMessage{
				Role: llm.ChatMessageRoleUser,
			},
		},
	}

	var lastUserMessageIndex int
	for idx, message := range history {
		if message.Role == llm.ChatMessageRoleUser {
			truncatedHistory[1].ChatCompletionMessage = message.ChatCompletionMessage
			lastUserMessageIndex = idx
		}
	}

	if options.Model == "" {
		options.Model = llm.GPT4oMini
	}

	res, err := provider.CreateChatCompletion(ctx, &llm.ChatCompletionRequest{
		Model:    options.Model,
		Messages: truncatedHistory.ChatCompletionMessages(),
	})
	if err != nil {
		return nil, &ErrorSignal{
			ErrorMessage: err.Error(),
			ErrorType:    StateErrorTypeUnknown,
		}
	}

	var annotation Annotation
	err = json.Unmarshal([]byte(res.Message.Content), &annotation)
	if err != nil || annotation.Data == nil {
		annotation = Annotation{
			Name: options.Annotation,
			Data: res.Message.Content,
		}
	}

	if history[lastUserMessageIndex].Annotations == nil {
		history[lastUserMessageIndex].Annotations = make(map[string]Annotation)
	}

	history[lastUserMessageIndex].Annotations[options.Annotation] = annotation
	history.AddTraceInformation(options.Annotation)

	return history, nil
}

func LLMCompletionState(options LLMCompletionOptions) BehaviorState {
	id, _ := GenerateStringIdentifier("id-", 16)
	if options.Id != "" {
		id = options.Id
	}
	return BehaviorState{
		HashId:           id,
		StateName:        options.Name,
		StateDescription: options.Description,
		ClientID:         options.ClientID,
		Lambda: func(ctx context.Context, history AnnotatedMessages) (AnnotatedMessages, Signal) {
			var system string
			{
				var buf bytes.Buffer

				var tmpl AnnotationTemplate
				_, err := tmpl.Parse(options.System)
				if err != nil {
					return history, &ErrorSignal{
						ErrorMessage: err.Error(),
					}
				}
				err = tmpl.Execute(&buf, history)
				if err != nil {
					return history, &ErrorSignal{
						ErrorMessage: err.Error(),
					}
				}

				system = buf.String()
			}

			if options.Annotation != "" {
				opts := options
				opts.System = system
				return evalIntoAnnotation(history, opts)
			}

			provider, err := llm.CreateModelProvider(options.Model, llm.ProviderOpenAI)
			if err != nil {
				return history, &ErrorSignal{
					ErrorMessage: err.Error(),
					ErrorType:    StateErrorTypeUnrecoverable,
				}
			}

			if len(options.ExtraContext) > 0 {
				system += "\n\nExtra Context:\n\n"
				for _, extraContext := range options.ExtraContext {
					annotation := history.GetAnnotation(extraContext)
					if annotation == nil {
						continue
					}
					system += fmt.Sprintf("%v\n\n", annotation.Data)
				}
			}

			if options.System != "" {
				if len(history) > 0 && history[0].Role == llm.ChatMessageRoleSystem {
					history[0].Content = system
				} else {
					history = append(AnnotatedMessages{
						AnnotatedMessage{
							ChatCompletionMessage: llm.ChatCompletionMessage{
								Role:    llm.ChatMessageRoleSystem,
								Content: system,
							},
							Annotations: make(map[string]Annotation),
						},
					}, history...)
				}
			}

			if options.Model == "" {
				options.Model = llm.GPT4oMini
			}

			request := llm.ChatCompletionRequest{
				Model:    options.Model,
				Messages: history.ChatCompletionMessages(),
			}

			var client *MCPClientMux
			if options.AllowTools {
				c := ctx.Value("arboreal_mcp_client")
				if c != nil {
					var ok bool
					client, ok = c.(*MCPClientMux)
					if ok {
						request.Tools = client.Tools()
					}
				}
			}

			var res *llm.ChatCompletionResponse
			for {
				res, err = provider.CreateChatCompletion(ctx, &request)
				if err != nil {
					return history, &ErrorSignal{
						ErrorMessage: err.Error(),
						ErrorType:    StateErrorTypeUnknown,
					}
				}

				// FIXME: Tool calls don't currently make it into the conversation history
				if len(res.Message.ToolCalls) == 0 || client == nil {
					break
				}

				r, err := client.CallTool(ctx, &mcp.CallToolParams{
					Name:      res.Message.ToolCalls[0].Name,
					Arguments: res.Message.ToolCalls[0].Arguments,
				})
				if err != nil {
					return history, &ErrorSignal{
						ErrorMessage: err.Error(),
					}
				}

				// TODO: Handle other types of content
				var content string
				switch t := r.Content[0].(type) {
				case *mcp.TextContent:
					content = t.Text
				default:
					b, err := r.Content[0].MarshalJSON()
					if err != nil {
						return history, &ErrorSignal{
							ErrorMessage: err.Error(),
						}
					}
					content = string(b)
				}

				toolResultMessage := llm.ChatCompletionMessage{
					Meta:    res.Message.ToolCalls[0].Meta,
					Name:    res.Message.ToolCalls[0].Name,
					Role:    llm.ChatMessageRoleFunction,
					Content: content,
				}

				// Keep tool call and result in history so callers can inspect executed tools.
				history = append(history,
					AnnotatedMessage{ChatCompletionMessage: res.Message},
					AnnotatedMessage{ChatCompletionMessage: toolResultMessage},
				)

				request.Messages = append(request.Messages, res.Message, toolResultMessage)
			}

			s := Signal(nil)
			if options.Terminal {
				s = &TerminalSignal{}
			}

			return append(history, AnnotatedMessage{ChatCompletionMessage: res.Message}), s
		},
	}
}

func PauseState(reason string) BehaviorState {
	id, _ := GenerateStringIdentifier("id-", 16)
	return BehaviorState{
		HashId: id,
		Lambda: func(ctx context.Context, history AnnotatedMessages) (AnnotatedMessages, Signal) {
			return history, &CollectUserInputSignal{
				Reason: reason,
			}
		},
	}
}
