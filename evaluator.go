package vesta

import (
	"context"
	"errors"
)

// Errors
var (
	// ErrSchemaNotRegistered is returned when no evaluator is registered for the object type and action.
	ErrSchemaNotRegistered = errors.New("vesta: no schema registered for object type and action")
)

// Evaluator implements authorization logic for point checks.
type Evaluator interface {
	Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error)
}

// Subject is a marker for domain-specific subject types (e.g., *Member, *ServiceAccount).
// Domain evaluators can type-assert to access rich data: member, ok := req.Subject.(*Member)
type Subject = any

// Object is a marker for domain-specific object types (e.g., *Article, *WorkOrder).
// Domain evaluators can type-assert to access rich data: article, ok := req.Object.(*Article)
type Object = any

// Request represents an authorization check request.
// Subject and Object are optional - when present, domain evaluators can type-assert for custom logic.
type Request struct {
	ObjectType ObjectType
	ObjectID   string
	Object     Object  // Optional: full object for domain evaluators (e.g., *Article)
	SubjectID  string
	Subject    Subject // Optional: full subject for domain evaluators (e.g., *Member)
	Action     Action
}

// EvaluatorFunc adapts a function to the Evaluator interface.
type EvaluatorFunc func(ctx context.Context, engine *Engine, req Request) (bool, error)

func (f EvaluatorFunc) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	return f(ctx, engine, req)
}
