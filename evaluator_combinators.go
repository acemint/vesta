package vesta

import "context"

// Combinator evaluators for building complex authorization rules

// OrEvaluator returns true if ANY child evaluator returns true.
type OrEvaluator struct {
	Evaluators []Evaluator
}

func Or(evals ...Evaluator) Evaluator {
	return &OrEvaluator{Evaluators: evals}
}

func (ev *OrEvaluator) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	for _, child := range ev.Evaluators {
		ok, err := child.Evaluate(ctx, engine, req)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// AndEvaluator returns true only if ALL child evaluators return true.
type AndEvaluator struct {
	Evaluators []Evaluator
}

func And(evals ...Evaluator) Evaluator {
	return &AndEvaluator{Evaluators: evals}
}

func (ev *AndEvaluator) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	for _, child := range ev.Evaluators {
		ok, err := child.Evaluate(ctx, engine, req)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

// NotEvaluator inverts an evaluator result.
type NotEvaluator struct {
	Evaluator Evaluator
}

func Not(eval Evaluator) Evaluator {
	return &NotEvaluator{Evaluator: eval}
}

func (ev *NotEvaluator) Evaluate(ctx context.Context, engine *Engine, req Request) (bool, error) {
	ok, err := ev.Evaluator.Evaluate(ctx, engine, req)
	if err != nil {
		return false, err
	}
	return !ok, nil
}
