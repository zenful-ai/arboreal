package arboreal

import (
	"reflect"
	"testing"
)

// Mock implementation of Hashable for testing
type MockHashable struct {
	id   string
	data string
}

func (m MockHashable) Hash() string {
	return m.id
}

// Stack Tests
func TestStackIsEmpty(t *testing.T) {
	t.Run("new stack is empty", func(t *testing.T) {
		s := Stack[int]{}
		if !s.IsEmpty() {
			t.Error("New stack should be empty")
		}
	})

	t.Run("stack with items is not empty", func(t *testing.T) {
		s := Stack[int]{}
		s.Push(1)
		if s.IsEmpty() {
			t.Error("Stack with items should not be empty")
		}
	})

	t.Run("stack becomes empty after popping all items", func(t *testing.T) {
		s := Stack[int]{}
		s.Push(1)
		s.Push(2)
		s.Pop()
		s.Pop()
		if !s.IsEmpty() {
			t.Error("Stack should be empty after popping all items")
		}
	})
}

func TestStackPush(t *testing.T) {
	t.Run("push single item", func(t *testing.T) {
		s := Stack[int]{}
		s.Push(42)

		if len(s.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(s.Items))
		}
		if s.Items[0] != 42 {
			t.Errorf("Expected 42, got %d", s.Items[0])
		}
	})

	t.Run("push multiple items", func(t *testing.T) {
		s := Stack[string]{}
		items := []string{"first", "second", "third"}

		for _, item := range items {
			s.Push(item)
		}

		if len(s.Items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(s.Items))
		}

		// Check order (LIFO)
		expected := []string{"first", "second", "third"}
		if !reflect.DeepEqual(s.Items, expected) {
			t.Errorf("Expected %v, got %v", expected, s.Items)
		}
	})
}

func TestStackPop(t *testing.T) {
	t.Run("pop from stack with items", func(t *testing.T) {
		s := Stack[int]{}
		s.Push(1)
		s.Push(2)
		s.Push(3)

		result := s.Pop()
		if result != 3 {
			t.Errorf("Expected 3, got %d", result)
		}
		if len(s.Items) != 2 {
			t.Errorf("Expected 2 items remaining, got %d", len(s.Items))
		}
	})

	t.Run("LIFO behavior", func(t *testing.T) {
		s := Stack[string]{}
		items := []string{"first", "second", "third"}

		for _, item := range items {
			s.Push(item)
		}

		// Pop in reverse order
		for i := len(items) - 1; i >= 0; i-- {
			result := s.Pop()
			if result != items[i] {
				t.Errorf("Expected %s, got %s", items[i], result)
			}
		}

		if !s.IsEmpty() {
			t.Error("Stack should be empty after popping all items")
		}
	})
}

func TestStackPopPanic(t *testing.T) {
	t.Run("pop from empty stack should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Pop from empty stack should panic")
			}
		}()

		s := Stack[int]{}
		s.Pop()
	})
}

// Graph Tests
func TestGraphAddNode(t *testing.T) {
	t.Run("add single node", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node := MockHashable{id: "node1", data: "test"}

		index := g.AddNode(node)

		if index != 0 {
			t.Errorf("Expected index 0, got %d", index)
		}
		if len(g.Nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(g.Nodes))
		}
		if g.Nodes[0].Hash() != "node1" {
			t.Errorf("Expected node1, got %s", g.Nodes[0].Hash())
		}
	})

	t.Run("add duplicate node returns same index", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node := MockHashable{id: "node1", data: "test"}

		index1 := g.AddNode(node)
		index2 := g.AddNode(node)

		if index1 != index2 {
			t.Errorf("Expected same index for duplicate node, got %d and %d", index1, index2)
		}
		if len(g.Nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(g.Nodes))
		}
	})

	t.Run("add multiple unique nodes", func(t *testing.T) {
		g := Graph[MockHashable]{}
		nodes := []MockHashable{
			{id: "node1", data: "test1"},
			{id: "node2", data: "test2"},
			{id: "node3", data: "test3"},
		}

		for i, node := range nodes {
			index := g.AddNode(node)
			if index != i {
				t.Errorf("Expected index %d, got %d", i, index)
			}
		}

		if len(g.Nodes) != 3 {
			t.Errorf("Expected 3 nodes, got %d", len(g.Nodes))
		}
	})
}

func TestGraphAddTransition(t *testing.T) {
	t.Run("add transition between nodes", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node1 := MockHashable{id: "node1", data: "test1"}
		node2 := MockHashable{id: "node2", data: "test2"}

		g.AddTransition(node1, node2)

		if len(g.Nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(g.Nodes))
		}

		// Check transition matrix
		if len(g.Transitions) != 2 {
			t.Errorf("Expected 2x2 transition matrix, got %dx%d", len(g.Transitions), len(g.Transitions[0]))
		}

		// node1 -> node2 should have value 0 (first transition)
		if g.Transitions[0][1] != 0 {
			t.Errorf("Expected transition value 0, got %d", g.Transitions[0][1])
		}
	})

	t.Run("multiple transitions from same node", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node1 := MockHashable{id: "node1", data: "test1"}
		node2 := MockHashable{id: "node2", data: "test2"}
		node3 := MockHashable{id: "node3", data: "test3"}

		g.AddTransition(node1, node2)
		g.AddTransition(node1, node3)

		// Check that transitions are numbered sequentially
		if g.Transitions[0][1] != 0 {
			t.Errorf("Expected first transition value 0, got %d", g.Transitions[0][1])
		}
		if g.Transitions[0][2] != 1 {
			t.Errorf("Expected second transition value 1, got %d", g.Transitions[0][2])
		}
	})

	t.Run("transition matrix grows correctly", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node1 := MockHashable{id: "node1", data: "test1"}
		node2 := MockHashable{id: "node2", data: "test2"}

		g.AddTransition(node1, node2)

		// Should be 2x2 matrix
		expectedSize := 2
		if len(g.Transitions) != expectedSize {
			t.Errorf("Expected %d rows, got %d", expectedSize, len(g.Transitions))
		}
		for i, row := range g.Transitions {
			if len(row) != expectedSize {
				t.Errorf("Expected %d columns in row %d, got %d", expectedSize, i, len(row))
			}
		}
	})
}

func TestGraphInitial(t *testing.T) {
	t.Run("get initial node", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node1 := MockHashable{id: "node1", data: "test1"}
		node2 := MockHashable{id: "node2", data: "test2"}

		g.AddNode(node1)
		g.AddNode(node2)

		initial := g.Initial()
		if initial.Hash() != "node1" {
			t.Errorf("Expected node1 as initial, got %s", initial.Hash())
		}
	})
}

func TestGraphInitialPanic(t *testing.T) {
	t.Run("initial on empty graph should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Initial() on empty graph should panic")
			}
		}()

		g := Graph[MockHashable]{}
		g.Initial()
	})
}

func TestGraphChildren(t *testing.T) {
	t.Run("node with no children", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node := MockHashable{id: "node1", data: "test1"}

		g.AddNode(node)
		children := g.Children(node)

		if len(children) != 0 {
			t.Errorf("Expected 0 children, got %d", len(children))
		}
	})

	t.Run("node with single child", func(t *testing.T) {
		g := Graph[MockHashable]{}
		parent := MockHashable{id: "parent", data: "parent"}
		child := MockHashable{id: "child", data: "child"}

		g.AddTransition(parent, child)
		children := g.Children(parent)

		if len(children) != 1 {
			t.Errorf("Expected 1 child, got %d", len(children))
		}
		if children[0].Hash() != "child" {
			t.Errorf("Expected child, got %s", children[0].Hash())
		}
	})

	t.Run("node with multiple children in order", func(t *testing.T) {
		g := Graph[MockHashable]{}
		parent := MockHashable{id: "parent", data: "parent"}
		child1 := MockHashable{id: "child1", data: "child1"}
		child2 := MockHashable{id: "child2", data: "child2"}
		child3 := MockHashable{id: "child3", data: "child3"}

		// Add transitions in order
		g.AddTransition(parent, child1)
		g.AddTransition(parent, child2)
		g.AddTransition(parent, child3)

		children := g.Children(parent)

		if len(children) != 3 {
			t.Errorf("Expected 3 children, got %d", len(children))
		}

		// Children should be returned in order of transition creation
		expectedOrder := []string{"child1", "child2", "child3"}
		for i, child := range children {
			if child.Hash() != expectedOrder[i] {
				t.Errorf("Expected %s at position %d, got %s", expectedOrder[i], i, child.Hash())
			}
		}
	})

	t.Run("non-existent node", func(t *testing.T) {
		g := Graph[MockHashable]{}
		node := MockHashable{id: "nonexistent", data: "test"}

		children := g.Children(node)

		if len(children) != 0 {
			t.Errorf("Expected 0 children for non-existent node, got %d", len(children))
		}
	})
}

// Benchmark tests
func BenchmarkStackPush(b *testing.B) {
	s := Stack[int]{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}
}

func BenchmarkStackPop(b *testing.B) {
	s := Stack[int]{}
	// Pre-populate stack
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Pop()
	}
}

func BenchmarkGraphAddNode(b *testing.B) {
	g := Graph[MockHashable]{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := MockHashable{id: string(rune(i)), data: "test"}
		g.AddNode(node)
	}
}