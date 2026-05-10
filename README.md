# What is Vesta?

**Vesta** is a multi-model authorization engine for Go. Schema-based, developer friendly. Supports ReBAC, RBAC, ABAC,
and custom evaluators. Named after the Roman goddess of hearth and protection, Vesta is about being *vested with
authority* — a lightweight, embedded library that brings fine-grained access control directly into your application
without the overhead of a separate service.

## Overview

### Why Vesta?

Vesta is designed for teams that want **fine-grained access control without operating a separate service**.

Unlike Zanzibar-inspired systems such as [SpiceDB](https://github.com/authzed/spicedb)
or [OpenFGA](https://github.com/openfga/openfga), Vesta is not a standalone database or gRPC service you deploy and
maintain. It is a Go library you import directly into your application:

|                   | Vesta                                                      | SpiceDB / OpenFGA              |
|-------------------|------------------------------------------------------------|--------------------------------|
| **Deployment**    | Embedded library                                           | Standalone service             |
| **API style**     | Plain Go functions                                         | gRPC / HTTP APIs               |
| **Schema**        | Go code, type-safe                                         | Protobuf / DSL                 |
| **Database**      | Your existing DB via GORM                                  | Managed by the service         |
| **Transactions**  | Native DB transactions across business logic + permissions | Separate from app transactions |
| **Model support** | ReBAC + RBAC + ABAC + custom evaluators                    | Primarily ReBAC                |

If you need a distributed permissions service across dozens of microservices, a Zanzibar-style system makes sense. If
you want to add fine-grained authorization to a Go application with minimal infrastructure, Vesta is the lighter fit.

### Features

- **ReBAC** (Direct, Transitive relations)
- **RBAC** (Role-based access control)
- **ABAC** (Attribute-based access control)
- **Combinators** (Or, And, Not)
- **Custom Evaluators** (Extend with your own logic)
- **Schema Registration** (Type-safe actions and object types)
- **Transactions** (Atomic grant/revoke operations)
- **Query Filtering** (ListObjects, ListSubjects)
- **Audit Trail** (Track who granted permissions)
- **Security** (No DB exposure, locked-down API)

## Quick Start

```go
func main() {
    db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    db.AutoMigrate(&vesta.Relation{})

    engine := vesta.NewEngine(db)
    ctx := context.Background()

    // Register schema
    schema := engine.Schema()
    schema.Register(ObjectTypeArticle, ActionView,
    vesta.Direct(RelationOwner, ObjectTypeMember))

    // Grant permission
    engine.Grant(ctx, "admin-user", vesta.Tuple{
        ObjectType:  ObjectTypeArticle,
        ObjectID:    "article-123",
        Relation:    RelationOwner,
        SubjectType: ObjectTypeMember,
        SubjectID:   "user-456",
    })

    // Check permission
    ok, _ := engine.Check(ctx, vesta.Request{
        ObjectType: ObjectTypeArticle,
        ObjectID:   "article-123",
        SubjectID:  "user-456",
        Action:     ActionView,
    })
    // ok == true

    // List accessible objects
    objectIDs, _ := engine.ListObjects(ctx, ObjectTypeArticle, "user-456", RelationOwner)
    // objectIDs == ["article-123"]

    // Revoke permission
    engine.Revoke(ctx, vesta.Tuple{
        ObjectType:  ObjectTypeArticle,
        ObjectID:    "article-123",
        Relation:    RelationOwner,
        SubjectType: ObjectTypeMember,
        SubjectID:   "user-456",
    })
}
```

## Evaluators

### Built-in

```go
func main()  {
    // Direct relation
    vesta.Direct(RelationOwner, ObjectTypeMember)

    // Transitive relation (through intermediate object)
    vesta.Transitive(RelationParentCompany, ObjectTypeCompany, RelationMember, ObjectTypeMember)

    // Combinators
    vesta.Or(eval1, eval2)
    vesta.And(eval1, eval2)
    vesta.Not(eval)
}
```

### Custom Evaluators

```go
type RoleEvaluator struct {
    Role string
}

func HasRole(role string) *RoleEvaluator {
    return &RoleEvaluator{Role: role}
}

func (ev *RoleEvaluator) Evaluate(ctx context.Context, engine *vesta.Engine, req vesta.Request) (bool, error) {
    member, ok := req.Subject.(*Member)
    if !ok {
        return false, fmt.Errorf("Subject must be *Member")
    }
    for _, r := range member.Roles {
        if r == ev.Role {
            return true, nil
        }
    }
    return false, nil
}

// Use in schema
func main() {
    schema.Register(ObjectTypeArticle, ActionEdit, HasRole("editor"))
}
```

## Advanced

### Transactions

```go
func main() {
    db.Transaction(func (tx *gorm.DB) error {
        tx.Create(&article)
        return engine.WithTx(tx).Grant(ctx, actorID, tuple)
    })
}
```

### Query Filtering

```go
func main() {
    if !user.IsSuperAdmin {
        objectIDs, _ := engine.ListObjects(ctx, objType, user.ID, relation)
        if len(objectIDs) == 0 {
            return []Article{}, nil
        }
        db = db.Where("id IN ?", objectIDs)
    }
}
```

### Error Handling

```go
func main() {
    ok, err := engine.Check(ctx, req)
    if errors.Is(err, vesta.ErrSchemaNotRegistered) {
        // No evaluator registered
    }
    if !ok {
        return ErrUnauthorized
    }
}
```
