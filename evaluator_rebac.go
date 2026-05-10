package vesta

import "context"

// ReBAC evaluators for relation-based access control

// DirectEvaluator checks for a direct relation in the relations table.
type DirectEvaluator struct {
	Relation    RelationType
	SubjectType ObjectType
}

// Direct creates a direct relation evaluator.
func Direct(rel RelationType, subType ObjectType) *DirectEvaluator {
	return &DirectEvaluator{
		Relation:    rel,
		SubjectType: subType,
	}
}

func (ev *DirectEvaluator) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	return engine.store.Check(ctx, Tuple{
		ObjectType:  req.ObjectType,
		ObjectID:    req.ObjectID,
		Relation:    ev.Relation,
		SubjectType: ev.SubjectType,
		SubjectID:   req.SubjectID,
	})
}

// TransitiveEvaluator checks transitive relationships through an intermediate object.
// Example: User can view WorkOrder if User is member of Company that owns the WorkOrder.
//
// Uses a single JOIN query instead of N+1 queries for performance.
type TransitiveEvaluator struct {
	Relation    RelationType // Relation from object to intermediate (e.g., "parent_company")
	ViaType     ObjectType   // Intermediate object type (e.g., "company")
	ViaRelation RelationType // Relation from intermediate to subject (e.g., "member")
	SubjectType ObjectType   // Subject type (e.g., "member")
}

// Transitive creates a transitive relation evaluator.
func Transitive(rel RelationType, viaType ObjectType, viaRelation RelationType, subType ObjectType) *TransitiveEvaluator {
	return &TransitiveEvaluator{
		Relation:    rel,
		ViaType:     viaType,
		ViaRelation: viaRelation,
		SubjectType: subType,
	}
}

func (ev *TransitiveEvaluator) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	// Use optimized single-query JOIN check instead of N+1 queries
	return engine.store.CheckTransitive(
		ctx,
		req.ObjectType, req.ObjectID, ev.Relation,
		ev.ViaType, ev.ViaRelation,
		ev.SubjectType, req.SubjectID,
	)
}
