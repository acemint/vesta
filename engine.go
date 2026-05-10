package vesta

import (
	"context"

	"gorm.io/gorm"
)

// Engine is the central authorization component.
// Coordinates relation storage, schema registration, and policy evaluation.
//
// The Engine is the high-level interface for authorization operations.
// It owns a GormStore for persistence and a Schema for evaluator registration.
//
// Key responsibilities:
//   - Check authorization requests against registered schema
//   - Filter queries to show only authorized objects
//   - Manage relation tuples (grant, revoke)
//   - Provide introspection APIs
type Engine struct {
	db     *gorm.DB
	store  *GormStore
	schema *Schema
}

// NewEngine creates a new authorization engine.
func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:     db,
		store:  NewGormStore(db),
		schema: NewSchema(),
	}
}

// WithTx returns a new Engine instance using the provided transaction.
// Use this for transactional operations.
//
// Example:
//
//	db.Transaction(func(tx *gorm.DB) error {
//	    return engine.WithTx(tx).Grant(ctx, actorID, tuple)
//	})
func (e *Engine) WithTx(tx *gorm.DB) *Engine {
	return &Engine{
		db:     tx,
		store:  e.store.WithTx(tx),
		schema: e.schema, // Schema is shared across transactions
	}
}

// Schema returns the schema registry.
// Use this to register evaluators for object types and actions.
func (e *Engine) Schema() *Schema {
	return e.schema
}

// ============================================================================
// Core Authorization API
// ============================================================================

// Check determines if a request is authorized using the registered schema.
// Returns true if authorized, false if denied.
// Returns ErrSchemaNotRegistered if no evaluator is registered for the object type and action.
func (e *Engine) Check(ctx context.Context, req Request) (bool, error) {
	evaluator, ok := e.schema.Get(req.ObjectType, req.Action)
	if !ok {
		return false, ErrSchemaNotRegistered
	}
	return evaluator.Evaluate(ctx, e, req)
}

// CheckMany checks multiple requests in sequence.
// Returns results in the same order as requests.
func (e *Engine) CheckMany(ctx context.Context, reqs []Request) ([]bool, error) {
	results := make([]bool, len(reqs))
	for i, req := range reqs {
		authorized, err := e.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		results[i] = authorized
	}
	return results, nil
}

// ============================================================================
// Relation Write Operations
// ============================================================================

// Grant creates a relation tuple (idempotent).
// createdBy should be the ID of the actor performing the grant (the authenticated user).
func (e *Engine) Grant(ctx context.Context, createdBy string, tuple Tuple) error {
	return e.store.Grant(ctx, createdBy, tuple)
}

// Revoke removes a relation tuple.
func (e *Engine) Revoke(ctx context.Context, tuple Tuple) error {
	return e.store.Revoke(ctx, tuple)
}

// GrantMany creates multiple relation tuples in a single transaction.
// More efficient than calling Grant repeatedly.
func (e *Engine) GrantMany(ctx context.Context, createdBy string, tuples []Tuple) error {
	return e.db.Transaction(func(tx *gorm.DB) error {
		txEngine := e.WithTx(tx)
		for _, tuple := range tuples {
			if err := txEngine.Grant(ctx, createdBy, tuple); err != nil {
				return err
			}
		}
		return nil
	})
}

// RevokeMany removes multiple relation tuples in a single transaction.
// More efficient than calling Revoke repeatedly.
func (e *Engine) RevokeMany(ctx context.Context, tuples []Tuple) error {
	return e.db.Transaction(func(tx *gorm.DB) error {
		txEngine := e.WithTx(tx)
		for _, tuple := range tuples {
			if err := txEngine.Revoke(ctx, tuple); err != nil {
				return err
			}
		}
		return nil
	})
}

// RevokeAll removes all relations for an object.
// Useful when deleting an object.
func (e *Engine) RevokeAll(ctx context.Context, objectType ObjectType, objectID string) error {
	return e.store.RevokeAll(ctx, objectType, objectID)
}

// ============================================================================
// Relation Queries
// ============================================================================

// ListSubjects returns all subjects with a given relation to an object.
//
// Example: List all members who are "owner" of article "123"
//
//	subjects, err := engine.ListSubjects(ctx, "article", "123", "owner")
func (e *Engine) ListSubjects(ctx context.Context, objectType ObjectType, objectID string, relation RelationType) ([]string, error) {
	return e.store.ListSubjects(ctx, objectType, objectID, relation)
}

// ListObjects returns all objects a subject has a given relation to.
//
// Example: List all articles where user "456" is "owner"
//
//	objects, err := engine.ListObjects(ctx, "article", "user-456", "owner")
func (e *Engine) ListObjects(ctx context.Context, objectType ObjectType, subjectID string, relation RelationType) ([]string, error) {
	return e.store.ListObjects(ctx, objectType, subjectID, relation)
}

// ============================================================================
// Introspection API
// ============================================================================

// ListAllSubjects returns all unique subject IDs for an object type and relation.
func (e *Engine) ListAllSubjects(ctx context.Context, objectType ObjectType, relation RelationType) ([]string, error) {
	return e.store.ListAllSubjects(ctx, objectType, relation)
}

// ListAllObjects returns all unique object IDs for a subject type and relation.
func (e *Engine) ListAllObjects(ctx context.Context, objectType ObjectType, subjectType ObjectType, relation RelationType) ([]string, error) {
	return e.store.ListAllObjects(ctx, objectType, subjectType, relation)
}

// ListAllObjectsGlobally returns all unique object IDs
func (e *Engine) ListAllObjectsGlobally(ctx context.Context, objectType ObjectType) ([]string, error) {
	return e.store.ListAllObjectsGlobally(ctx, objectType)
}

// ListAllRelations returns all unique relations for an object type.
func (e *Engine) ListAllRelations(ctx context.Context, objectType ObjectType) ([]RelationType, error) {
	return e.store.ListAllRelations(ctx, objectType)
}

// GetRelations returns all relations for a specific object.
func (e *Engine) GetRelations(ctx context.Context, objectType ObjectType, objectID string) ([]Tuple, error) {
	return e.store.GetRelations(ctx, objectType, objectID)
}

// FindRelations returns relations matching filter criteria.
func (e *Engine) FindRelations(ctx context.Context, filter RelationFilter) ([]Tuple, error) {
	return e.store.FindRelations(ctx, filter)
}
