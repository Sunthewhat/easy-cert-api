package participantmodel

import (
	"github.com/sunthewhat/easy-cert-api/type/shared/query"
	"go.mongodb.org/mongo-driver/mongo"
)

// ParticipantRepository handles all participant database operations
// It manages both PostgreSQL (for indexes/status) and MongoDB (for dynamic data)
type ParticipantRepository struct {
	q  *query.Query      // PostgreSQL query builder
	db *mongo.Database   // MongoDB database
}

// NewParticipantRepository creates a new participant repository with dependency injection
func NewParticipantRepository(q *query.Query, db *mongo.Database) *ParticipantRepository {
	return &ParticipantRepository{
		q:  q,
		db: db,
	}
}

// Note: The actual methods will delegate to the existing functions in participant.go, participant.sql.go, and participant.nosql.go
// This allows us to gradually migrate the implementation without breaking existing code
// For now, controllers can use this repository pattern while the underlying implementation stays the same
