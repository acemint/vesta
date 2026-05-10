package vesta

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestEngine(t *testing.T) (*Engine, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate the relation table
	err = db.AutoMigrate(&Relation{})
	require.NoError(t, err)

	engine := NewEngine(db)
	return engine, db
}

func TestEngine_Check_NoSchema(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// No schema registered yet - should return error
	ok, err := engine.Check(ctx, Request{
		ObjectType: "article",
		ObjectID:   "123",
		SubjectID:  "user-1",
		Action:     "view",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSchemaNotRegistered)
	assert.False(t, ok, "Should deny when no schema registered")
}

func TestEngine_Check_WithSchema(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Register a simple evaluator that always allows
	engine.Schema().Register("article", "view", EvaluatorFunc(func(ctx context.Context, engine *Engine, req Request) (bool, error) {
		return true, nil
	}))

	// Create a relation
	err := engine.Grant(ctx, "test-actor", Tuple{
		ObjectType:  "article",
		ObjectID:    "123",
		Relation:    "owner",
		SubjectType: "member",
		SubjectID:   "user-1",
	})
	require.NoError(t, err)

	// Check should succeed
	ok, err := engine.Check(ctx, Request{
		ObjectType: "article",
		ObjectID:   "123",
		SubjectID:  "user-1",
		Action:     "view",
	})

	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEngine_CheckMany(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Register schema: always allow "view", always deny "edit"
	engine.Schema().Register("article", "view", EvaluatorFunc(func(ctx context.Context, engine *Engine, req Request) (bool, error) {
		return true, nil
	}))
	engine.Schema().Register("article", "edit", EvaluatorFunc(func(ctx context.Context, engine *Engine, req Request) (bool, error) {
		return false, nil
	}))

	results, err := engine.CheckMany(ctx, []Request{
		{ObjectType: "article", ObjectID: "123", SubjectID: "user-1", Action: "view"},
		{ObjectType: "article", ObjectID: "123", SubjectID: "user-1", Action: "edit"},
		{ObjectType: "article", ObjectID: "456", SubjectID: "user-1", Action: "view"},
	})

	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, results)
}

func TestEngine_GrantAndRevoke(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	tuple := Tuple{
		ObjectType:  "article",
		ObjectID:    "123",
		Relation:    "owner",
		SubjectType: "member",
		SubjectID:   "user-1",
	}

	// Grant
	err := engine.Grant(ctx, "test-actor", tuple)
	require.NoError(t, err)

	// Verify exists
	exists, err := engine.store.Check(ctx, tuple)
	require.NoError(t, err)
	assert.True(t, exists)

	// Revoke
	err = engine.Revoke(ctx, tuple)
	require.NoError(t, err)

	// Verify removed
	exists, err = engine.store.Check(ctx, tuple)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestEngine_GrantMany(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	tuples := []Tuple{
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
		{ObjectType: "article", ObjectID: "456", Relation: "owner", SubjectType: "member", SubjectID: "user-2"},
	}

	// Grant multiple
	err := engine.GrantMany(ctx, "test-actor", tuples)
	require.NoError(t, err)

	// Verify both exist
	for _, tuple := range tuples {
		exists, err := engine.store.Check(ctx, tuple)
		require.NoError(t, err)
		assert.True(t, exists)
	}
}

func TestEngine_RevokeMany(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	tuples := []Tuple{
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
		{ObjectType: "article", ObjectID: "456", Relation: "owner", SubjectType: "member", SubjectID: "user-2"},
	}

	// Grant first
	for _, tuple := range tuples {
		err := engine.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)
	}

	// Revoke multiple
	err := engine.RevokeMany(ctx, tuples)
	require.NoError(t, err)

	// Verify both removed
	for _, tuple := range tuples {
		exists, err := engine.store.Check(ctx, tuple)
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestEngine_ListSubjects(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Grant multiple subjects access to same object
	tuples := []Tuple{
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-2"},
	}

	for _, tuple := range tuples {
		err := engine.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)
	}

	// List subjects
	subjects, err := engine.ListSubjects(ctx, "article", "123", "owner")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"user-1", "user-2"}, subjects)
}

func TestEngine_ListObjects(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Grant subject access to multiple objects
	tuples := []Tuple{
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
		{ObjectType: "article", ObjectID: "456", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
	}

	for _, tuple := range tuples {
		err := engine.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)
	}

	// List objects
	objects, err := engine.ListObjects(ctx, "article", "user-1", "owner")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"123", "456"}, objects)
}

func TestEngine_RevokeAll(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Grant multiple relations to an object
	tuples := []Tuple{
		{ObjectType: "article", ObjectID: "123", Relation: "owner", SubjectType: "member", SubjectID: "user-1"},
		{ObjectType: "article", ObjectID: "123", Relation: "viewer", SubjectType: "member", SubjectID: "user-2"},
	}

	for _, tuple := range tuples {
		err := engine.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)
	}

	// Revoke all
	err := engine.RevokeAll(ctx, "article", "123")
	require.NoError(t, err)

	// Verify all removed
	for _, tuple := range tuples {
		exists, err := engine.store.Check(ctx, tuple)
		require.NoError(t, err)
		assert.False(t, exists)
	}
}
