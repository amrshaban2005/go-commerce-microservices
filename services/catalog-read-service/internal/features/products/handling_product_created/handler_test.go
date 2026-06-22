package handlingproductcreated

import (
	"context"
	"errors"
	"testing"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/mocks"
	"github.com/stretchr/testify/mock"
)

type fakeProductRepository struct {
	upsertCalled bool
	err          error
}

func (f *fakeProductRepository) FindAll(ctx context.Context) ([]domain.Product, error) {
	return nil, nil
}

func (f *fakeProductRepository) Upsert(ctx context.Context, product domain.Product) error {
	f.upsertCalled = true
	return f.err
}

type fakeInboxRepository struct {
	processed           bool
	saveProcessedCalled bool
	saveErr             error
	isProcessedErr      error
}

func (f *fakeInboxRepository) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	return f.processed, f.isProcessedErr
}
func (f *fakeInboxRepository) SaveProcessed(ctx context.Context, messageID string, eventType string, payload []byte) error {
	f.saveProcessedCalled = true
	return f.saveErr
}

func Test_Handler_NewMessage_UpsertsProductAndSavesInbox(t *testing.T) {

	productRepo := mocks.NewProductRepository(t)
	inboxRepo := mocks.NewInboxRepository(t)

	productRepo.On("Upsert", mock.Anything, domain.Product{ID: "product-1"}).Return(nil).Once()
	inboxRepo.On("IsProcessed", mock.Anything, "message1").Return(false, nil).Once()
	inboxRepo.On("SaveProcessed", mock.Anything, "message1", "ProductCreated", []byte("payload")).Return(nil).Once()

	handler := NewHandler(productRepo, inboxRepo)
	_, err := handler.Handle(context.Background(), &Command{
		MessageID: "message1",
		Product:   domain.Product{ID: "product-1"},
		Payload:   []byte("payload"),
	})
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}
}

func Test_Handler_ProductAlreadyCreated(t *testing.T) {
	productRepo := &fakeProductRepository{}
	inboxRepo := &fakeInboxRepository{processed: true}
	handler := NewHandler(productRepo, inboxRepo)
	_, err := handler.Handle(context.Background(), &Command{
		MessageID: "message1",
		Product:   domain.Product{},
		Payload:   []byte("payload"),
	})
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}
}

func Test_Handler_ReturnRepositoryError(t *testing.T) {
	err := errors.New("db error")
	productRepo := &fakeProductRepository{err: err}
	inboxRepo := &fakeInboxRepository{processed: false}

	handler := NewHandler(productRepo, inboxRepo)
	_, err = handler.Handle(context.Background(), &Command{
		MessageID: "message2",
		Product:   domain.Product{},
		Payload:   []byte("payload"),
	})
	if err == nil {
		t.Errorf("expected error but got no error")
	}
}

func Test_Handler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		isProcessed    bool
		isProcessedErr error
		upsertErr      error
		saveErr        error

		wantErr    bool
		wantUpsert bool
		wantSave   bool
	}{
		{
			name:        "message already processed",
			isProcessed: true,
			wantErr:     false,
			wantUpsert:  false,
			wantSave:    false,
		},
		{
			name:        "new message upserts product and saves inbox",
			isProcessed: false,
			wantErr:     false,
			wantUpsert:  true,
			wantSave:    true,
		},
		{
			name:           "is processed error",
			isProcessedErr: errors.New("inbox failed"),
			wantErr:        true,
			wantUpsert:     false,
			wantSave:       false,
		},
		{
			name:        "upsert error",
			upsertErr:   errors.New("mongo failed"),
			isProcessed: false,
			wantErr:     true,
			wantUpsert:  true,
			wantSave:    false,
		},
		{
			name:        "save processed error",
			saveErr:     errors.New("save failed"),
			isProcessed: false,
			wantErr:     true,
			wantUpsert:  true,
			wantSave:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productRepo := &fakeProductRepository{
				err: test.upsertErr,
			}

			inboxRepo := &fakeInboxRepository{
				processed:      test.isProcessed,
				isProcessedErr: test.isProcessedErr,
				saveErr:        test.saveErr,
			}

			handler := NewHandler(productRepo, inboxRepo)

			_, err := handler.Handle(context.Background(), &Command{
				MessageID: "message-1",
				Product: domain.Product{
					ID:          "product-1",
					Name:        "Keyboard",
					Description: "Mechanical keyboard",
					Price:       100,
					Status:      "active",
				},
				Payload: []byte("payload"),
			})

			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}

			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}

			if test.wantUpsert && !productRepo.upsertCalled {
				t.Fatal("expected Upsert to be called")
			}

			if !test.wantUpsert && productRepo.upsertCalled {
				t.Fatal("expected Upsert not to be called")
			}

			if test.wantSave && !inboxRepo.saveProcessedCalled {
				t.Fatal("expected SaveProcessed to be called")
			}

			if !test.wantSave && inboxRepo.saveProcessedCalled {
				t.Fatal("expected SaveProcessed not to be called")
			}
		})
	}
}
