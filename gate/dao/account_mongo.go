package dao

import (
	"context"

	"hk4e/pkg/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Account struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	OpenId        string             `bson:"open_id"`
	Uid           uint32             `bson:"uid"`
	IsForbid      bool               `bson:"is_forbid"`
	ForbidEndTime uint32             `bson:"forbid_end_time"`
}

func (d *Dao) InsertAccount(account *Account) (primitive.ObjectID, error) {
	db := d.db.Collection("account")
	id, err := db.InsertOne(context.TODO(), account)
	if err != nil {
		return primitive.ObjectID{}, err
	} else {
		_id, ok := id.InsertedID.(primitive.ObjectID)
		if !ok {
			logger.Error("get insert id error")
			return primitive.ObjectID{}, nil
		}
		return _id, nil
	}
}

func (d *Dao) QueryAccountByOpenId(openId string) (*Account, error) {
	db := d.db.Collection("account")
	result := db.FindOne(
		context.TODO(),
		bson.D{{"open_id", openId}},
	)
	account := new(Account)
	err := result.Decode(account)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return account, nil
}
