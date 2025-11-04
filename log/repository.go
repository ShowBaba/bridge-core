package log

import (
	"context"
	"time"

	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SortOrder int

const (
	SortAsc SortOrder = iota
	SortDesc
)

type Filter struct {
	UserID      string
	Application string
	Level       string
	Since       *time.Time
	Query       string
}

type ListOptions struct {
	Limit int
	Sort  SortOrder
}

type Repository interface {
	Create(ctx context.Context, doc StreamLog) error
	List(ctx context.Context, f Filter, opt ListOptions) ([]StreamLog, int64, error)
}

type repository struct {
	mongoClient *mongo.Client
	coll        *mongo.Collection
}

func NewRepository(mongoClient *mongo.Client) Repository {
	coll := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)
	return &repository{mongoClient, coll}
}

func (r *repository) Create(ctx context.Context, data StreamLog) error {
	collection := r.mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)
	_, err := collection.InsertOne(ctx, data)
	return err
}

func (r *repository) List(ctx context.Context, f Filter, opt ListOptions) ([]StreamLog, int64, error) {
	q := bson.M{}
	if f.UserID != "" {
		q["user"] = f.UserID
	}
	if f.Application != "" {
		q["application"] = f.Application
	}
	if f.Level != "" {
		q["level"] = f.Level
	}
	if f.Since != nil {
		q["created_at"] = bson.M{"$gte": f.Since.UTC()}
	}
	if f.Query != "" {
		q["message"] = bson.M{"$regex": f.Query, "$options": "i"}
	}

	findOpt := options.Find()
	if opt.Limit > 0 {
		findOpt.SetLimit(int64(opt.Limit))
	}
	sort := 1
	if opt.Sort == SortDesc {
		sort = -1
	}
	findOpt.SetSort(bson.D{{Key: "created_at", Value: sort}})

	cur, err := r.coll.Find(ctx, q, findOpt)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var out []StreamLog
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}

	total, _ := r.coll.CountDocuments(ctx, q)

	return out, total, nil
}
