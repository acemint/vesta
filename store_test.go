package vesta

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, *GormStore) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&Relation{})
	require.NoError(t, err)

	store := NewGormStore(db)
	return db, store
}

func TestNewGormStore(t *testing.T) {
	_, store := setupTestDB(t)

	require.NotNil(t, store)
}

func TestGrant(t *testing.T) {
	t.Run("creates new relation", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		err := store.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)

		var count int64
		db.Model(&Relation{}).Where("object_id = ?", "doc-123").Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("is idempotent - does not error on duplicate", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		err := store.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)

		err = store.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)

		var count int64
		db.Model(&Relation{}).Where("object_id = ?", "doc-123").Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("can create multiple relations for same object", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		err := store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-1",
		})
		require.NoError(t, err)

		err = store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-2",
		})
		require.NoError(t, err)

		var count int64
		db.Model(&Relation{}).Where("object_id = ?", "doc-123").Count(&count)
		require.Equal(t, int64(2), count)
	})
}

func TestCheck(t *testing.T) {
	t.Run("returns true when relation exists", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		err := store.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)

		exists, err := store.Check(ctx, tuple)
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("returns false when relation does not exist", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		exists, err := store.Check(ctx, tuple)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("differentiates between different relations", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		err := store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		})
		require.NoError(t, err)

		exists, err := store.Check(ctx, Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-456",
		})
		require.NoError(t, err)
		require.False(t, exists)
	})
}

func TestRevoke(t *testing.T) {
	t.Run("removes existing relation", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		err := store.Grant(ctx, "test-actor", tuple)
		require.NoError(t, err)

		err = store.Revoke(ctx, tuple)
		require.NoError(t, err)

		exists, err := store.Check(ctx, tuple)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("does not error when relation does not exist", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		err := store.Revoke(ctx, tuple)
		require.NoError(t, err)
	})

	t.Run("only removes specific relation", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		err := store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-1",
		})
		require.NoError(t, err)

		err = store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-1",
		})
		require.NoError(t, err)

		err = store.Revoke(ctx, Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-1",
		})
		require.NoError(t, err)

		exists, err := store.Check(ctx, Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-1",
		})
		require.NoError(t, err)
		require.True(t, exists)
	})
}

func TestListSubjects(t *testing.T) {
	t.Run("returns all subjects with relation", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "viewer",
			SubjectType: "user",
			SubjectID:   "user-1",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "viewer",
			SubjectType: "user",
			SubjectID:   "user-2",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-3",
		})

		subjects, err := store.ListSubjects(ctx, "document", "doc-123", "viewer")
		require.NoError(t, err)
		require.Len(t, subjects, 2)
		require.Contains(t, subjects, "user-1")
		require.Contains(t, subjects, "user-2")
	})

	t.Run("returns empty list when no subjects exist", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		subjects, err := store.ListSubjects(ctx, "document", "doc-123", "viewer")
		require.NoError(t, err)
		require.Len(t, subjects, 0)
	})

	t.Run("filters by relation correctly", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-1",
		})

		subjects, err := store.ListSubjects(ctx, "document", "doc-123", "editor")
		require.NoError(t, err)
		require.Len(t, subjects, 0)
	})
}

func TestListObjects(t *testing.T) {
	t.Run("returns all objects subject has relation to", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-123",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-2",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-123",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-3",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-123",
		})

		objects, err := store.ListObjects(ctx, "document", "user-123", "owner")
		require.NoError(t, err)
		require.Len(t, objects, 2)
		require.Contains(t, objects, "doc-1")
		require.Contains(t, objects, "doc-2")
	})

	t.Run("returns empty list when no objects exist", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		objects, err := store.ListObjects(ctx, "document", "user-123", "owner")
		require.NoError(t, err)
		require.Len(t, objects, 0)
	})

	t.Run("filters by object type", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-123",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "folder",
			ObjectID:    "folder-1",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-123",
		})

		objects, err := store.ListObjects(ctx, "document", "user-123", "owner")
		require.NoError(t, err)
		require.Len(t, objects, 1)
		require.Contains(t, objects, "doc-1")
	})
}

func TestRevokeAll(t *testing.T) {
	t.Run("removes all relations for an object", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-1",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "editor",
			SubjectType: "user",
			SubjectID:   "user-2",
		})

		store.Grant(ctx, "test-actor", Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-456",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-3",
		})

		err := store.RevokeAll(ctx, "document", "doc-123")
		require.NoError(t, err)

		var count int64
		db.Model(&Relation{}).Where("object_id = ?", "doc-123").Count(&count)
		require.Equal(t, int64(0), count)

		db.Model(&Relation{}).Where("object_id = ?", "doc-456").Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("does not error when no relations exist", func(t *testing.T) {
		_, store := setupTestDB(t)
		ctx := context.Background()

		err := store.RevokeAll(ctx, "document", "doc-123")
		require.NoError(t, err)
	})
}

func TestTransactionalOperations(t *testing.T) {
	t.Run("grant and check within transaction", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		err := db.Transaction(func(tx *gorm.DB) error {
			txStore := store.WithTx(tx)

			tuple := Tuple{
				ObjectType:  "document",
				ObjectID:    "doc-123",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "user-456",
			}

			if err := txStore.Grant(ctx, "test-actor", tuple); err != nil {
				return err
			}

			exists, err := txStore.Check(ctx, tuple)
			if err != nil {
				return err
			}

			if !exists {
				t.Error("relation should exist within transaction")
			}

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("rollback on error", func(t *testing.T) {
		db, store := setupTestDB(t)
		ctx := context.Background()

		tuple := Tuple{
			ObjectType:  "document",
			ObjectID:    "doc-123",
			Relation:    "owner",
			SubjectType: "user",
			SubjectID:   "user-456",
		}

		db.Transaction(func(tx *gorm.DB) error {
			txStore := store.WithTx(tx)
			txStore.Grant(ctx, "test-actor", tuple)
			return gorm.ErrInvalidTransaction
		})

		exists, err := store.Check(ctx, tuple)
		require.NoError(t, err)
		require.False(t, exists)
	})
}
