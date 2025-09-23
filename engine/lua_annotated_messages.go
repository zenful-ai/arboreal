package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

var arborealAnnotatedMessage = LuaTypeExport{
	Name:        "message",
	Constructor: newArborealAnnotatedMessage,
	Methods: map[string]lua.LGFunction{
		"role":    arborealAnnotatedMessageRole,
		"content": arborealAnnotatedMessageContent,
	},
}

func arborealAnnotatedMessageFromLua(ud *lua.LUserData) (*arboreal.AnnotatedMessage, error) {
	if vv, ok := ud.Value.(*arboreal.AnnotatedMessage); ok {
		return vv, nil
	}

	return nil, fmt.Errorf("failed to convert userdata to arboreal AnnotatedMessage")
}

func arborealAnnotatedMessagesFromLua(t *lua.LTable) (*arboreal.AnnotatedMessages, error) {
	var err error

	messages := make(arboreal.AnnotatedMessages, t.Len())

	t.ForEach(func(k, v lua.LValue) {
		if ud, ok := v.(*lua.LUserData); ok {
			var m *arboreal.AnnotatedMessage
			m, err = arborealAnnotatedMessageFromLua(ud)
			if err != nil {
				return
			}

			if x, ok := k.(lua.LNumber); ok {
				messages[int(x)-1] = *m
			}
		} else {
			err = fmt.Errorf("expected a message but got %T", v)
			return
		}
	})
	if err != nil {
		return nil, err
	}

	return &messages, nil
}

func arborealAnnotatedMessageToLua(l *lua.LState, m *arboreal.AnnotatedMessage) lua.LValue {
	ud := l.NewUserData()
	ud.Value = m

	l.SetMetatable(ud, l.GetTypeMetatable("message"))
	l.Push(ud)

	return ud
}

func arborealAnnotatedMessagesToLua(l *lua.LState, m arboreal.AnnotatedMessages) lua.LValue {
	messages := l.NewTable()

	for idx, message := range m {
		messages.RawSetInt(idx+1, arborealAnnotatedMessageToLua(l, &message))
	}

	return messages
}

func checkArborealAnnotatedMessage(l *lua.LState) *arboreal.AnnotatedMessage {
	ud := l.CheckUserData(1)

	message, err := arborealAnnotatedMessageFromLua(ud)
	if err != nil {
		l.ArgError(1, err.Error())
		return nil
	}

	return message
}

func checkArborealAnnotatedMessages(l *lua.LState, n int) *arboreal.AnnotatedMessages {
	messageArray := l.CheckTable(n)
	messages, err := arborealAnnotatedMessagesFromLua(messageArray)
	if err != nil {
		l.ArgError(n, err.Error())
	}

	return messages
}

func newArborealAnnotatedMessage(l *lua.LState) int {
	var message = arboreal.AnnotatedMessage{
		ChatCompletionMessage: llm.ChatCompletionMessage{
			Role:    l.CheckString(1),
			Content: l.CheckString(2),
		},
	}

	ud := l.NewUserData()
	ud.Value = &message

	l.SetMetatable(ud, l.GetTypeMetatable("message"))
	l.Push(ud)

	return 1
}

func arborealAnnotatedMessageRole(l *lua.LState) int {
	message := checkArborealAnnotatedMessage(l)

	l.Push(lua.LString(message.Role))
	return 1
}

func arborealAnnotatedMessageContent(l *lua.LState) int {
	message := checkArborealAnnotatedMessage(l)

	l.Push(lua.LString(message.ChatCompletionMessage.Content))
	return 1
}
