package vesta

import "sync"

// Schema manages the mapping from (ObjectType, Action) -> Evaluator.
// Thread-safe for concurrent registration and lookup.
//
// The schema is the central registry of authorization rules.
// Register evaluators during initialization, then use Get() during request handling.
type Schema struct {
	mu      sync.RWMutex
	mapping map[ObjectType]map[string]Evaluator
}

// NewSchema creates an empty schema.
func NewSchema() *Schema {
	return &Schema{
		mapping: make(map[ObjectType]map[string]Evaluator),
	}
}

// Register adds an evaluator for an object type and action.
// If an evaluator already exists for this combination, it will be replaced.
//
//	schema.Register("article", ActionView, DirectEvaluator{...})
func (s *Schema) Register(objType ObjectType, action Action, evaluator Evaluator) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mapping[objType] == nil {
		s.mapping[objType] = make(map[string]Evaluator)
	}
	s.mapping[objType][string(action)] = evaluator
}

// RegisterAll registers multiple evaluators for an object type.
// Convenient for defining all actions for a resource at once.
//
// Example:
//
//	schema.RegisterAll("article", map[Action]Evaluator{
//	    ActionView:   viewEvaluator,
//	    ActionEdit:   editEvaluator,
//	    ActionDelete: deleteEvaluator,
//	})
func (s *Schema) RegisterAll(objType ObjectType, actions map[Action]Evaluator) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mapping[objType] == nil {
		s.mapping[objType] = make(map[string]Evaluator)
	}

	for action, evaluator := range actions {
		s.mapping[objType][string(action)] = evaluator
	}
}

// Get retrieves an evaluator for an object type and action.
// Returns (evaluator, true) if found, (nil, false) if not registered.
func (s *Schema) Get(objType ObjectType, action Action) (Evaluator, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	actions, ok := s.mapping[objType]
	if !ok {
		return nil, false
	}

	evaluator, ok := actions[string(action)]
	return evaluator, ok
}

// ListActions returns all registered actions for an object type.
// Returns nil if the object type is not registered.
func (s *Schema) ListActions(objType ObjectType) []Action {
	s.mu.RLock()
	defer s.mu.RUnlock()

	actions, ok := s.mapping[objType]
	if !ok {
		return nil
	}

	result := make([]Action, 0, len(actions))
	for action := range actions {
		result = append(result, Action(action))
	}
	return result
}

// ListObjectTypes returns all registered object types.
func (s *Schema) ListObjectTypes() []ObjectType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ObjectType, 0, len(s.mapping))
	for objType := range s.mapping {
		result = append(result, objType)
	}
	return result
}

// Clear removes all registered evaluators.
// Primarily useful for testing.
func (s *Schema) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mapping = make(map[ObjectType]map[string]Evaluator)
}
