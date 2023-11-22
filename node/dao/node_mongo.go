package dao

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Region struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Ec2bData []byte             `bson:"ec2b_data"`
	NextUid  uint32             `bson:"next_uid"`
}

func (d *Dao) InsertRegion(region *Region) error {
	db := d.db.Collection("region")
	_, err := db.InsertOne(context.TODO(), region)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdateRegion(region *Region) error {
	db := d.db.Collection("region")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{},
		bson.D{{"$set", region}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) QueryRegion() (*Region, error) {
	db := d.db.Collection("region")
	result := db.FindOne(
		context.TODO(),
		bson.D{},
	)
	region := new(Region)
	err := result.Decode(region)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return region, nil
}

type StopServerInfo struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	StopServer      bool               `bson:"stop_server"`
	StartTime       uint32             `bson:"start_time"`
	EndTime         uint32             `bson:"end_time"`
	IpAddrWhiteList []string           `bson:"ip_addr_white_list"`
}

func (d *Dao) InsertStopServerInfo(stopServerInfo *StopServerInfo) error {
	db := d.db.Collection("stop_server_info")
	_, err := db.InsertOne(context.TODO(), stopServerInfo)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdateStopServerInfo(stopServerInfo *StopServerInfo) error {
	db := d.db.Collection("stop_server_info")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{},
		bson.D{{"$set", stopServerInfo}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) QueryStopServerInfo() (*StopServerInfo, error) {
	db := d.db.Collection("stop_server_info")
	result := db.FindOne(
		context.TODO(),
		bson.D{},
	)
	stopServerInfo := new(StopServerInfo)
	err := result.Decode(stopServerInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return stopServerInfo, nil
}
