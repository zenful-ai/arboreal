package llm

import "testing"

func TestParseModelURI(t *testing.T) {
	m := ParseModelURI(ProviderOpenAI + ":" + ModelGPT4oMini)

	if m.Type != ProviderOpenAI {
		t.Fail()
	}

	if m.Name != ModelGPT4oMini {
		t.Fail()
	}

	m = ParseModelURI(ProviderOllama + ":" + ModelChatModelLlama8b)

	if m.Type != ProviderOllama {
		t.Fail()
	}

	if m.Name != ModelChatModelLlama8b {
		t.Fail()
	}

	m = ParseModelURI(ModelChatModelLlama8b)

	if m.Type != ProviderUnknown {
		t.Fail()
	}

	if m.Name != ModelChatModelLlama8b {
		t.Fail()
	}
}
