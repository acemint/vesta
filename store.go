package vesta

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// GormStore implements relation storage using GORM.
// Provides CRUD operations and query helpers that encapsulate field names.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore creates a new GORM-based relation store.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// WithTx returns a new store instance using the provided transaction.
// Use this for transactional operations.
func (s *GormStore) WithTx(tx *gorm.DB) *GormStore {
	return &GormStore{db: tx}
}

// Check verifies if a relation tuple exists.
func (s *GormStore) Check(ctx context.Context, tuple Tuple) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ? AND "+ColumnRelation+" = ? AND "+ColumnSubjectType+" = ? AND "+ColumnSubjectID+" = ?",
			tuple.ObjectType, tuple.ObjectID, tuple.Relation, tuple.SubjectType, tuple.SubjectID).
		Count(&count).Error
	return count > 0, err
}

// Grant creates a relation tuple (idempotent - no error if already exists).
// createdBy should be the ID of the actor performing the grant operation (the authenticated user).
func (s *GormStore) Grant(ctx context.Context, createdBy string, tuple Tuple) error {
	relation := Relation{
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		ObjectType:  string(tuple.ObjectType),
		ObjectID:    tuple.ObjectID,
		Relation:    string(tuple.Relation),
		SubjectType: string(tuple.SubjectType),
		SubjectID:   tuple.SubjectID,
	}

	return s.db.WithContext(ctx).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ? AND "+ColumnRelation+" = ? AND "+ColumnSubjectType+" = ? AND "+ColumnSubjectID+" = ?",
			relation.ObjectType, relation.ObjectID, relation.Relation, relation.SubjectType, relation.SubjectID).
		FirstOrCreate(&relation).Error
}

// Revoke removes a relation tuple.
func (s *GormStore) Revoke(ctx context.Context, tuple Tuple) error {
	return s.db.WithContext(ctx).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ? AND "+ColumnRelation+" = ? AND "+ColumnSubjectType+" = ? AND "+ColumnSubjectID+" = ?",
			tuple.ObjectType, tuple.ObjectID, tuple.Relation, tuple.SubjectType, tuple.SubjectID).
		Delete(&Relation{}).Error
}

// ListSubjects returns all subject IDs with the given relation to an object.
func (s *GormStore) ListSubjects(ctx context.Context, objectType ObjectType, objectID string, relation RelationType) ([]string, error) {
	var subjectIDs []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ? AND "+ColumnRelation+" = ?", objectType, objectID, relation).
		Pluck(ColumnSubjectID, &subjectIDs).Error
	return subjectIDs, err
}

// ListObjects returns all object IDs the subject has the given relation to.
func (s *GormStore) ListObjects(ctx context.Context, objectType ObjectType, subjectID string, relation RelationType) ([]string, error) {
	var objectIDs []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ? AND "+ColumnSubjectID+" = ? AND "+ColumnRelation+" = ?", objectType, subjectID, relation).
		Pluck(ColumnObjectID, &objectIDs).Error
	return objectIDs, err
}

// RevokeAll removes all relations for an object.
func (s *GormStore) RevokeAll(ctx context.Context, objectType ObjectType, objectID string) error {
	return s.db.WithContext(ctx).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ?", objectType, objectID).
		Delete(&Relation{}).Error
}

// ListAllSubjects returns all unique subject IDs for an object type and relation.
func (s *GormStore) ListAllSubjects(ctx context.Context, objectType ObjectType, relation RelationType) ([]string, error) {
	var subjects []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ? AND "+ColumnRelation+" = ?", objectType, relation).
		Distinct(ColumnSubjectID).
		Pluck(ColumnSubjectID, &subjects).Error
	return subjects, err
}

// ListAllObjects returns all unique object IDs for a subject type and relation.
func (s *GormStore) ListAllObjects(ctx context.Context, objectType ObjectType, subjectType ObjectType, relation RelationType) ([]string, error) {
	var objects []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ? AND "+ColumnSubjectType+" = ? AND "+ColumnRelation+" = ?",
			objectType, subjectType, relation).
		Distinct(ColumnObjectID).
		Pluck(ColumnObjectID, &objects).Error
	return objects, err
}

// ListAllObjectsGlobally returns every object ID of a given type, regardless of subject.
// Used when a global role grants access to all objects.
func (s *GormStore) ListAllObjectsGlobally(ctx context.Context, objectType ObjectType) ([]string, error) {
	var objectIDs []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ?", objectType).
		Distinct(ColumnObjectID).
		Pluck(ColumnObjectID, &objectIDs).Error
	return objectIDs, err
}

// ListAllRelations returns all unique relations for an object type.
func (s *GormStore) ListAllRelations(ctx context.Context, objectType ObjectType) ([]RelationType, error) {
	var relations []string
	err := s.db.WithContext(ctx).Model(&Relation{}).
		Where(ColumnObjectType+" = ?", objectType).
		Distinct(ColumnRelation).
		Pluck(ColumnRelation, &relations).Error

	result := make([]RelationType, len(relations))
	for i, r := range relations {
		result[i] = RelationType(r)
	}
	return result, err
}

// GetRelations returns all relations for a specific object.
func (s *GormStore) GetRelations(ctx context.Context, objectType ObjectType, objectID string) ([]Tuple, error) {
	var relations []Relation
	err := s.db.WithContext(ctx).
		Where(ColumnObjectType+" = ? AND "+ColumnObjectID+" = ?", objectType, objectID).
		Find(&relations).Error
	if err != nil {
		return nil, err
	}

	tuples := make([]Tuple, len(relations))
	for i, r := range relations {
		tuples[i] = Tuple{
			ObjectType:  ObjectType(r.ObjectType),
			ObjectID:    r.ObjectID,
			Relation:    RelationType(r.Relation),
			SubjectType: ObjectType(r.SubjectType),
			SubjectID:   r.SubjectID,
		}
	}
	return tuples, nil
}

// RelationFilter defines criteria for filtering relations.
type RelationFilter struct {
	ObjectType  ObjectType
	ObjectID    string
	Relation    RelationType
	SubjectType ObjectType
	SubjectID   string
}

// FindRelations returns relations matching filter criteria.
func (s *GormStore) FindRelations(ctx context.Context, filter RelationFilter) ([]Tuple, error) {
	query := s.db.WithContext(ctx).Model(&Relation{})

	if filter.ObjectType != "" {
		query = query.Where(ColumnObjectType+" = ?", filter.ObjectType)
	}
	if filter.ObjectID != "" {
		query = query.Where(ColumnObjectID+" = ?", filter.ObjectID)
	}
	if filter.Relation != "" {
		query = query.Where(ColumnRelation+" = ?", filter.Relation)
	}
	if filter.SubjectType != "" {
		query = query.Where(ColumnSubjectType+" = ?", filter.SubjectType)
	}
	if filter.SubjectID != "" {
		query = query.Where(ColumnSubjectID+" = ?", filter.SubjectID)
	}

	var relations []Relation
	if err := query.Find(&relations).Error; err != nil {
		return nil, err
	}

	tuples := make([]Tuple, len(relations))
	for i, r := range relations {
		tuples[i] = Tuple{
			ObjectType:  ObjectType(r.ObjectType),
			ObjectID:    r.ObjectID,
			Relation:    RelationType(r.Relation),
			SubjectType: ObjectType(r.SubjectType),
			SubjectID:   r.SubjectID,
		}
	}
	return tuples, nil
}

// CheckTransitive checks for a transitive relation in a single query using JOIN.
// This avoids N+1 queries by checking the relationship in one database operation.
//
// Example: Check if subject has viaRelation to any intermediate object that has relation to the target object.
//
//	exists := store.CheckTransitive(ctx,
//	    "work_order", "wo-123", "parent_company",  // object → company
//	    "company", "member",                        // company → subject
//	    "member", "user-456")                       // subject ID
func (s *GormStore) CheckTransitive(
	ctx context.Context,
	objectType ObjectType, objectID string, relation RelationType,
	viaType ObjectType, viaRelation RelationType,
	subjectType ObjectType, subjectID string,
) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("relation AS r1").
		Joins("INNER JOIN relation AS r2 ON r1.subject_id = r2.object_id AND r1.subject_type = r2.object_type").
		Where("r1.object_type = ? AND r1.object_id = ? AND r1.relation = ?", objectType, objectID, relation).
		Where("r2.relation = ? AND r2.subject_type = ? AND r2.subject_id = ?", viaRelation, subjectType, subjectID).
		Count(&count).Error
	return count > 0, err
}
