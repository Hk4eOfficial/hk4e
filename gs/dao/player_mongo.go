package dao

import (
	"context"
	"errors"

	"hk4e/gs/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	MaxQueryChatMsgLen = 1000 // 最大可查询聊天记录条数
)

func (d *Dao) InsertPlayer(player *model.Player) error {
	db := d.db.Collection("player")
	_, err := db.InsertOne(context.TODO(), player)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) InsertChatMsg(chatMsg *model.ChatMsg) error {
	db := d.db.Collection("chat_msg")
	_, err := db.InsertOne(context.TODO(), chatMsg)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) InsertPlayerList(playerList []*model.Player) error {
	if len(playerList) == 0 {
		return nil
	}
	db := d.db.Collection("player")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, player := range playerList {
		modelOperate := mongo.NewInsertOneModel().SetDocument(player)
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) InsertChatMsgList(chatMsgList []*model.ChatMsg) error {
	if len(chatMsgList) == 0 {
		return nil
	}
	db := d.db.Collection("chat_msg")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, chatMsg := range chatMsgList {
		modelOperate := mongo.NewInsertOneModel().SetDocument(chatMsg)
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) DeletePlayer(playerId uint32) error {
	db := d.db.Collection("player")
	_, err := db.DeleteOne(context.TODO(), bson.D{{"player_id", playerId}})
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) DeleteChatMsg(id primitive.ObjectID) error {
	db := d.db.Collection("chat_msg")
	_, err := db.DeleteOne(context.TODO(), bson.D{{"_id", id}})
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) DeletePlayerList(playerIdList []uint32) error {
	if len(playerIdList) == 0 {
		return nil
	}
	db := d.db.Collection("player")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, playerId := range playerIdList {
		modelOperate := mongo.NewDeleteOneModel().SetFilter(bson.D{{"player_id", playerId}})
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) DeleteChatMsgList(idList []primitive.ObjectID) error {
	if len(idList) == 0 {
		return nil
	}
	db := d.db.Collection("chat_msg")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, id := range idList {
		modelOperate := mongo.NewDeleteOneModel().SetFilter(bson.D{{"_id", id}})
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdatePlayer(player *model.Player) error {
	db := d.db.Collection("player")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{{"player_id", player.PlayerId}},
		bson.D{{"$set", player}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdateChatMsg(chatMsg *model.ChatMsg) error {
	db := d.db.Collection("chat_msg")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{{"_id", chatMsg.ID}},
		bson.D{{"$set", chatMsg}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdatePlayerList(playerList []*model.Player) error {
	if len(playerList) == 0 {
		return nil
	}
	db := d.db.Collection("player")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, player := range playerList {
		modelOperate := mongo.NewUpdateManyModel().SetFilter(bson.D{{"player_id", player.PlayerId}}).SetUpdate(bson.D{{"$set", player}})
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) UpdateChatMsgList(chatMsgList []*model.ChatMsg) error {
	if len(chatMsgList) == 0 {
		return nil
	}
	db := d.db.Collection("chat_msg")
	modelOperateList := make([]mongo.WriteModel, 0)
	for _, chatMsg := range chatMsgList {
		modelOperate := mongo.NewUpdateManyModel().SetFilter(bson.D{{"_id", chatMsg.ID}}).SetUpdate(bson.D{{"$set", chatMsg}})
		modelOperateList = append(modelOperateList, modelOperate)
	}
	_, err := db.BulkWrite(context.TODO(), modelOperateList)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) QueryPlayerById(playerId uint32) (*model.Player, error) {
	db := d.db.Collection("player")
	result := db.FindOne(
		context.TODO(),
		bson.D{{"player_id", playerId}},
	)
	player := new(model.Player)
	err := result.Decode(player)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return player, nil
}

func (d *Dao) QueryChatMsgById(id primitive.ObjectID) (*model.ChatMsg, error) {
	db := d.db.Collection("chat_msg")
	result := db.FindOne(
		context.TODO(),
		bson.D{{"_id", id}},
	)
	chatMsg := new(model.ChatMsg)
	err := result.Decode(chatMsg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return chatMsg, nil
}

func (d *Dao) QueryPlayerList() ([]*model.Player, error) {
	db := d.db.Collection("player")
	find, err := db.Find(
		context.TODO(),
		bson.D{},
	)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Player, 0)
	for find.Next(context.TODO()) {
		item := new(model.Player)
		err = find.Decode(item)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (d *Dao) QueryChatMsgList() ([]*model.ChatMsg, error) {
	db := d.db.Collection("chat_msg")
	find, err := db.Find(
		context.TODO(),
		bson.D{},
	)
	if err != nil {
		return nil, err
	}
	result := make([]*model.ChatMsg, 0)
	for find.Next(context.TODO()) {
		item := new(model.ChatMsg)
		err = find.Decode(item)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (d *Dao) QueryChatMsgListByUid(uid uint32) ([]*model.ChatMsg, error) {
	db := d.db.Collection("chat_msg")
	result := make([]*model.ChatMsg, 0)
	find, err := db.Find(
		context.TODO(),
		bson.D{
			{"$and", []bson.D{
				{{"$or", []bson.D{{{"to_uid", uid}}, {{"uid", uid}}}}},
				{{"is_delete", false}},
			}},
		},
		options.Find().SetLimit(MaxQueryChatMsgLen),
		options.Find().SetSort(bson.M{"time": 1}),
	)
	if err != nil {
		return nil, err
	}
	for find.Next(context.TODO()) {
		item := new(model.ChatMsg)
		err = find.Decode(item)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (d *Dao) ReadUpdateChatMsgByUid(uid uint32, targetUid uint32) error {
	db := d.db.Collection("chat_msg")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{{"to_uid", uid}, {"uid", targetUid}},
		bson.D{{"$set", bson.D{{"is_read", true}}}},
	)
	if err != nil {
		return err
	}
	_, err = db.UpdateMany(
		context.TODO(),
		bson.D{{"uid", uid}, {"to_uid", targetUid}},
		bson.D{{"$set", bson.D{{"is_read", true}}}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) DeleteAllUpdateChatMsgByUid(uid uint32) error {
	db := d.db.Collection("chat_msg")
	_, err := db.UpdateMany(
		context.TODO(),
		bson.D{{"$or", []bson.D{{{"to_uid", uid}}, {{"uid", uid}}}}},
		bson.D{{"$set", bson.D{{"is_delete", true}}}},
	)
	if err != nil {
		return err
	}
	return nil
}
