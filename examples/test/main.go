package main

import "github.com/zenful-ai/arboreal"

func main() {

	chatBehavior := arboreal.CreateBehaviorTree("chat_behavior", "A nice conversational bot", "<insert user's input here>")

	chatState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{})
	askForInputState := arboreal.PauseState("Let user respond")

	chatState2 := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{})

	chatBehavior.AddTransition(&chatState, &askForInputState)
	chatBehavior.AddTransition(&askForInputState, &chatState2)

	exec := arboreal.CreateTodoListExecutive("Chat Bot", "Simple chat bot thingy", &chatBehavior)

	err := exec.RunLoop(arboreal.TerminalChannel{})

	if err != nil {
		panic(err)
	}
	//	schema := `-- Real estate listing data dump from the MLS
	//CREATE TABLE IF NOT EXISTS mls_listings (
	//	id INTEGER PRIMMARY KEY ASC,
	//	mls_number INTEGER,
	//	data TEXT NOT NULL
	//);
	//CREATE UNIQUE INDEX IF NOT EXISTS mls_listings_mls_number ON mls_listings(mls_number);
	//
	//-- Clients the user is working with to find homes
	//CREATE TABLE IF NOT EXISTS clients (
	//	id INTEGER PRIMARY KEY ASC,
	//	created_at TEXT,
	//	name TEXT NOT NULL				-- First and last name separated by a space
	//);
	//CREATE INDEX IF NOT EXISTS clients_name ON clients(name);
	//
	//-- Associations between listings and clients who may be interested in those listings
	//CREATE TABLE IF NOT EXISTS clients_to_listings (
	//	client_id INTEGER,
	//	listing_id INTEGER,
	//	FOREIGN KEY(client_id) REFERENCES client(id),
	//	FOREIGN KEY(listing_id) REFERENCES listing(id),
	//);
	//`
	//
	//	s := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//		System: "Respond only in valid SQL (SQLite syntax), no need for any explanation or markdown formatting.",
	//	})
	//
	//	messages, _ := s.Lambda(arboreal.AnnotatedMessages{
	//		{
	//			ChatCompletionMessage: llm.ChatCompletionMessage{
	//				Role:    llm.ChatMessageRoleUser,
	//				Content: fmt.Sprintf("Given the following SQLite schema, write a query which finds clients added in the last 30 days: %s", schema),
	//			},
	//			Annotations: make(map[string]arboreal.Annotation),
	//		},
	//	})
	//
	//	fmt.Println(messages.LastMessage().Content)
	//
	// Toy 3: Poetry bot using multi-try behavior
	//
	//poetryBehavior := arboreal.CreateBehaviorTree("write_poem", "Write a poem in a particular style", "Write a haiku about the beauty of clouds")
	//
	//pause := arboreal.PauseState("Need user input")
	//
	//evalForHaiku := arboreal.LLMUserEvalOrErrorState(arboreal.LLMEvalOptions{
	//	System:     "Decide if the user is requesting a haiku. Only respond in valid JSON with the following fields:\n\tdata - a boolean representing whether the user wants a haiku\n\texplanation - a one sentence explanation for your reason. Example:\n\n{\"data\": \"false\", \"explanation\": \"The user is requesting a limerick\"}",
	//	Annotation: "haiku",
	//}, func(a *arboreal.Annotation) bool {
	//	b, ok := a.Data.(bool)
	//	return ok && b
	//})
	//
	//evalForSonnet := arboreal.LLMUserEvalOrErrorState(arboreal.LLMEvalOptions{
	//	System:     "Decide if the user is requesting a sonnet. Only respond in valid JSON with the following fields:\n\tdata - a boolean representing whether the user wants a sonnet\n\texplanation - a one sentence explanation for your reason. Example:\n\n{\"data\": \"false\", \"explanation\": \"The user is requesting a haiku\"}",
	//	Annotation: "sonnet",
	//}, func(a *arboreal.Annotation) bool {
	//	b, ok := a.Data.(bool)
	//	return ok && b
	//})
	//
	//haikuComposer := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Respond to the user in a haiku.",
	//})
	//
	//sonnetComposer := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Respond to the user in a sonnet.",
	//})
	//
	//poetryBehavior.AddTransition(&pause, &evalForHaiku)
	//poetryBehavior.AddTransition(&pause, &evalForSonnet)
	//poetryBehavior.AddTransition(&evalForHaiku, &haikuComposer)
	//poetryBehavior.AddTransition(&evalForSonnet, &sonnetComposer)
	//
	//var history arboreal.AnnotatedMessages
	//
	//for {
	//	var signal arboreal.Signal
	//	history, signal = poetryBehavior.Lambda(history)
	//
	//	// FIXME
	//	if len(history) > 0 && history[len(history)-1].Role == llm.ChatMessageRoleAssistant {
	//		fmt.Print("[Assistant]\n\n")
	//		fmt.Println(history[len(history)-1].Content)
	//		fmt.Println()
	//	}
	//
	//	switch signal.(type) {
	//	case *arboreal.ErrorSignal:
	//		fmt.Println(signal.BehaviorDescription())
	//	case *arboreal.CollectUserInputSignal:
	//		input := getResponse()
	//		history = arboreal.AppendToMessages(history, llm.ChatCompletionMessage{
	//			Role:    llm.ChatMessageRoleUser,
	//			Content: input,
	//		})
	//	}
	//}

	//
	// Toy 2: TodoListExecutive poetry bot
	//
	//haikuBehavior := arboreal.CreateBehaviorTree("write_haiku", "Write a haiku", "Write a haiku about the fall of civilization.")
	//
	//haikuState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Respond to the user in a haiku fitting the user's request",
	//})
	//
	//haikuBehavior.AddState(&haikuState)
	//
	//sonnetBehavior := arboreal.CreateBehaviorTree("write_sonnet", "Write a sonnet", "Write a sonnet")
	//
	//sonnetState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Respond to the user in a sonnet fitting the user's request",
	//})
	//
	//sonnetBehavior.AddState(&sonnetState)
	//
	//executive := arboreal.TodoListExecutive{
	//	Behaviors: []arboreal.BehaviorTree{
	//		haikuBehavior,
	//		sonnetBehavior,
	//	},
	//	OutOfBoundsHandler: arboreal.CannedResponseState("Sorry, but I can only assist in poetry related requests (Haikus & Sonnets)."),
	//}
	//
	//userInput := getResponse()
	//
	//conversation := arboreal.AnnotatedMessages{
	//	{
	//		ChatCompletionMessage: llm.ChatCompletionMessage{
	//			Role:    llm.ChatMessageRoleUser,
	//			Content: userInput,
	//		},
	//	},
	//}
	//
	//executive.Plan(conversation)
	//executive.Lambda(conversation)
	//
	//fmt.Println(executive.Output)

	//
	// Toy 1: Haiku / not haiku bot
	//
	//behavior := arboreal.CreateBehaviorTree("Haiku Bot", "A fun bot to parrot Haikus back to you, which is sassy if you don't respond in kind.", "blah")
	//
	//pause := arboreal.PauseState("Need user input")
	//
	//eval := arboreal.LLMUserEvalState(arboreal.LLMEvalOptions{
	//	System:     "Decide if the user's message was a valid haiku. Only respond in valid JSON with the following fields:\n\tdata - a boolean representing whether the message is a valid haiku\n\texplanation - a one sentence explanation for your reason. Example:\n\n{\"data\": \"false\", \"explanation\": \"This is an invalid haiku because there is an extra syllable in the first line.\"}",
	//	Annotation: "haiku",
	//})
	//
	//branch := arboreal.BranchOnAnnotation("haiku", arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Respond to the user in a haiku.",
	//}), arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
	//	System: "Shame the user for not writing in a haiku. Your response must be a haiku.",
	//}))
	//
	//behavior.AddTransition(&pause, &eval)
	//behavior.AddTransition(&eval, &branch)
	//behavior.AddTransition(&branch, &pause)
	//
	//var history arboreal.AnnotatedMessages
	//
	//for {
	//	var signal arboreal.Signal
	//	history, signal = behavior.Lambda(history)
	//
	//	// FIXME
	//	if len(history) > 0 && history[len(history)-1].Role == llm.ChatMessageRoleAssistant {
	//		fmt.Print("[Assistant]\n\n")
	//		fmt.Println(history[len(history)-1].Content)
	//		fmt.Println()
	//	}
	//
	//	switch signal.(type) {
	//	case *arboreal.ErrorSignal:
	//		fmt.Println(signal.BehaviorDescription())
	//	case *arboreal.CollectUserInputSignal:
	//		input := getResponse()
	//		history = arboreal.AppendToMessages(history, llm.ChatCompletionMessage{
	//			Role:    llm.ChatMessageRoleUser,
	//			Content: input,
	//		})
	//	}
	//}
}
