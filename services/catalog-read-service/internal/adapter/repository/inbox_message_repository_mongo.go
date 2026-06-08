package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type InboxMessageModel struct {
	MessageID   string    `bson:"message_id"`
	EventType   string    `bson:"event_type"`
	Payload     string    `bson:"payload"`
	ProcessedAt time.Time `bson:"processed_at"`
}

type InboxMessageMongoRepository struct {
	collection *mongo.Collection
}

func NewInboxMessageMongoRepository(db *mongo.Database) (*InboxMessageMongoRepository, error) {
	collection := db.Collection("inbox_messages")

	_, err := collection.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys: bson.M{"message_id": 1},
			Options: options.Index().
				SetUnique(true).
				SetName("idx_inbox_message_id_unique"),
		},
	)
	if err != nil {
		return nil, err
	}

	return &InboxMessageMongoRepository{
		collection: collection,
	}, nil
}

func (r *InboxMessageMongoRepository) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"message_id": messageID})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *InboxMessageMongoRepository) SaveProcessed(
	ctx context.Context,
	messageID string,
	eventType string,
	payload []byte,
) error {
	model := InboxMessageModel{
		MessageID:   messageID,
		EventType:   eventType,
		Payload:     string(payload),
		ProcessedAt: time.Now().UTC(),
	}

	_, err := r.collection.InsertOne(ctx, model)
	return err
}
