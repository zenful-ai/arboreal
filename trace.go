package arboreal

import (
	"github.com/zenful-ai/arboreal/llm"
	"time"
)

const (
	TraceMessageTypeLuaSource = "lua_source"
	TraceMessageTypeCallBegin = "begin_call"
	TraceMessageTypeCallEnd   = "end_call"
)

type Trace chan *TraceMessage

func (t Trace) Send(msg *TraceMessage) {
	if t != nil {
		t <- msg
	}
}

type TraceTelemetry struct {
	Start time.Time  `json:"start"`
	End   *time.Time `json:"end"`
}

type TraceHistoryOperation struct {
	Type       string                     `json:"type"`
	Action     string                     `json:"action"`
	Annotation *Annotation                `json:"annotation,omitempty"`
	Message    *llm.ChatCompletionMessage `json:"message,omitempty"`
}

type TraceSignal struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func TraceForSignal(sig Signal) *TraceSignal {
	var t TraceSignal

	switch sig.(type) {
	case *ErrorSignal:
		t.Type = "error"
	case *SkipSignal:
		t.Type = "skip"
	case *TerminalSignal:
		t.Type = "stop"
	case *CollectUserInputSignal:
		t.Type = "user"
	case nil:
		return nil
	default:
		panic("unknown Signal type")
	}

	t.Reason = sig.Description()
	return &t
}

type TraceMessage struct {
	Type       string                   `json:"type"`
	ID         string                   `json:"id"`
	ClientID   string                   `json:"client_id,omitempty"`
	Name       string                   `json:"name"`
	Message    string                   `json:"message"`
	Error      error                    `json:"error,omitempty"`
	Telemetry  *TraceTelemetry          `json:"telemetry,omitempty"`
	Operations []*TraceHistoryOperation `json:"operations,omitempty"`
	Signal     *TraceSignal             `json:"signal,omitempty"`
}
