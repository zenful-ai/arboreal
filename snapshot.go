package arboreal

type Snapshot map[string]snapshotvalue

func (snap Snapshot) Restore(root Behavior) error {
	// Build up a map of references
	var lookupMap = make(map[string]Behavior)

	// Used to traverse the entire tree
	var s Stack[Behavior]
	var toProcess Stack[Behavior]

	s.Push(root)
	for !s.IsEmpty() {
		next := s.Pop()

		switch t := next.(type) {
		case *BehaviorState:
			lookupMap[t.Hash()] = t

			// No need to process behavior states
		case *BehaviorTree:
			lookupMap[t.Hash()] = t

			for _, node := range t.Graph.Nodes {
				s.Push(node)
			}

			if _, ok := snap[t.Hash()]; ok {
				toProcess.Push(t)
			}
		case *TodoListExecutive:
			lookupMap[t.Hash()] = t

			for _, behavior := range t.Behaviors {
				s.Push(behavior)
			}

			if _, ok := snap[t.Hash()]; ok {
				toProcess.Push(t)
			}
		}
	}

	for !toProcess.IsEmpty() {
		next := toProcess.Pop()

		switch t := next.(type) {
		case *BehaviorState:
		// Do nothing
		case *BehaviorTree:
			behavior := lookupMap[t.Hash()].(*BehaviorTree)
			state := snap[t.Hash()]

			behavior.Traversed = state.Traversed

			// Rehydrate the state by looking up references
			for _, b := range state.State {
				behavior.State.Items = append(behavior.State.Items, lookupMap[string(b)])
			}
		case *TodoListExecutive:
			planner := lookupMap[t.Hash()].(*TodoListExecutive)
			state := snap[t.Hash()]

			planner.History = state.History

			for _, p := range state.Plan {
				behavior := lookupMap[p.Ref].Copy()
				err := p.Snapshot.Restore(behavior)
				if err != nil {
					return err
				}

				planner.plan = append(planner.plan, &ExecGeneratedStep{
					Behavior:        behavior,
					Messages:        p.Messages,
					ReplanTombstone: p.ReplanTombstone,
				})
			}
		}
	}

	return nil
}

type BehaviorRef string

type ExecGeneratedStepSkeleton struct {
	Ref             string            `json:"ref"`
	Snapshot        Snapshot          `json:"snapshot"`
	Messages        AnnotatedMessages `json:"messages"`
	ReplanTombstone bool              `json:"replan_tombstone"`
}

type snapshotvalue struct {
	// BehaviorTree data
	State     []BehaviorRef   `json:"state,omitempty"`
	Traversed map[string]bool `json:"traversed,omitempty"`

	// TodoListExecutive data
	History AnnotatedMessages           `json:"history,omitempty"`
	Plan    []ExecGeneratedStepSkeleton `json:"plan,omitempty"`
}

func TakeSnapshot(root Behavior) (Snapshot, error) {
	var snapshot = make(Snapshot)
	var s Stack[Behavior]

	s.Push(root)
	for !s.IsEmpty() {
		b := s.Pop()

		switch t := b.(type) {
		case *BehaviorState:
		// Do nothing
		case *TodoListExecutive:
			// If there is no plan, no need to snapshot
			if len(t.plan) == 0 {
				break
			}

			// First, push all behaviors that belong to this TodoListExecutive onto the stack
			for _, behavior := range t.Behaviors {
				s.Push(behavior)
			}

			val := snapshotvalue{
				History: t.History,
			}

			for _, p := range t.plan {
				s, err := TakeSnapshot(p.Behavior)
				if err != nil {
					return nil, err
				}

				val.Plan = append(val.Plan, ExecGeneratedStepSkeleton{
					Ref:             p.Behavior.Hash(),
					Snapshot:        s,
					Messages:        p.Messages,
					ReplanTombstone: p.ReplanTombstone,
				})
			}

			snapshot[t.Hash()] = val

		case *BehaviorTree:
			// Traverse all tree nodes
			for _, node := range t.Graph.Nodes {
				s.Push(node)
			}

			if !t.State.IsEmpty() {
				var states []BehaviorRef
				for _, item := range t.State.Items {
					states = append(states, BehaviorRef(item.Hash()))
				}

				val := snapshotvalue{
					State:     states,
					Traversed: t.Traversed,
				}
				snapshot[t.Hash()] = val
			}

		}
	}

	return snapshot, nil
}
