package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type InboxMessageDataModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	MessageID   uuid.UUID      `gorm:"type:uuid;column:message_id"`
	EventType   string         `gorm:"column:event_type"`
	Payload     datatypes.JSON `gorm:"column:payload;type:jsonb"`
	ProcessedAt time.Time      `gorm:"column:processed_at"`
}

func (InboxMessageDataModel) TableName() string {
	return "inbox_messages"
}

type InboxMessageRepositoryPG struct {
	db *gorm.DB
}

func NewInboxMessageRepository(db *gorm.DB) *InboxMessageRepositoryPG {

	return &InboxMessageRepositoryPG{db}
}

func (r *InboxMessageRepositoryPG) IsProcessed(ctx context.Context, messageID uuid.UUID) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&InboxMessageDataModel{}).
		Where("message_id = ?", messageID).
		Count(&count).Error

	return count > 0, err
}

func (r *InboxMessageRepositoryPG) SaveProcessed(
	ctx context.Context,
	messageID uuid.UUID,
	eventType string,
	payload []byte,
) error {

	inbox := InboxMessageDataModel{
		ID:          uuid.New(),
		MessageID:   messageID,
		EventType:   eventType,
		Payload:     datatypes.JSON(payload),
		ProcessedAt: time.Now().UTC(),
	}

	return r.db.WithContext(ctx).Create(&inbox).Error

}
