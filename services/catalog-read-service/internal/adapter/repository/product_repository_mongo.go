package repository

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ProductReadModel struct {
	ID          string  `bson:"_id"`
	Name        string  `bson:"name"`
	Description string  `bson:"description"`
	Price       float64 `bson:"price"`
	Status      string  `bson:"status"`
}

type ProductRepositoryMongo struct {
	collection *mongo.Collection
}

func NewProductRepositoryMongo(db *mongo.Database) *ProductRepositoryMongo {
	return &ProductRepositoryMongo{
		collection: db.Collection("products"),
	}
}

func (r *ProductRepositoryMongo) Upsert(ctx context.Context, product domain.Product) error {
	model := ProductReadModel{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Status:      product.Status,
	}

	_, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": product.ID},
		model,
		options.Replace().SetUpsert(true),
	)

	return err
}

func (r *ProductRepositoryMongo) FindAll(ctx context.Context) ([]domain.Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []ProductReadModel
	if err := cursor.All(ctx, &models); err != nil {
		return nil, err
	}

	products := make([]domain.Product, 0, len(models))
	for _, model := range models {
		products = append(products, domain.Product{
			ID:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Price:       model.Price,
			Status:      model.Status,
		})
	}

	return products, nil
}
