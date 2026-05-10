package vesta

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Relation represents a Zanzibar-style authorization tuple.
// Stores relationships between objects and subjects: (object_type, object_id, relation, subject_type, subject_id)
type Relation struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	CreatedBy string    `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`

	ObjectType  string `gorm:"not null;index:idx_relation_lookup;index:idx_relation_object" json:"object_type"`
	ObjectID    string `gorm:"not null;index:idx_relation_lookup;index:idx_relation_object" json:"object_id"`
	Relation    string `gorm:"not null;index:idx_relation_lookup;index:idx_relation_object" json:"relation"`
	SubjectType string `gorm:"not null;index:idx_relation_lookup;index:idx_relation_subject" json:"subject_type"`
	SubjectID   string `gorm:"not null;index:idx_relation_lookup;index:idx_relation_subject" json:"subject_id"`
}

// TableName specifies the table name for GORM.
func (r *Relation) TableName() string {
	return "relation"
}

// BeforeCreate hook to auto-generate ID and set CreatedAt.
func (r *Relation) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.CreatedAt = time.Now()
	return nil
}

// Column name constants for relation table.
// Use these instead of hardcoded strings in queries.
const (
	ColumnObjectType  = "object_type"
	ColumnObjectID    = "object_id"
	ColumnRelation    = "relation"
	ColumnSubjectType = "subject_type"
	ColumnSubjectID   = "subject_id"
	ColumnID          = "id"
)

// Tuple represents a relation triple with type-safe fields.
// In ReBAC, subjects are objects - there's no distinction.
type Tuple struct {
	ObjectType  ObjectType
	ObjectID    string
	Relation    RelationType
	SubjectType ObjectType
	SubjectID   string
}

// ObjectType represents the type of object in a relation tuple.
// In ReBAC, both objects and subjects use this type.
type ObjectType string

// RelationType represents the type of relation between objects.
type RelationType string

// Action represents an authorization action that can be performed on an object.
// Actions are mapped to evaluators via the schema.
type Action string
