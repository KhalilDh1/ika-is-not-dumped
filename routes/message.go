package routes

import (
	"apartments-clone-server/models"
	"apartments-clone-server/storage"
	"apartments-clone-server/utils"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func CreateMessage(ctx iris.Context) {
    var req CreateMessageInput
    err := ctx.ReadJSON(&req)
    if err != nil {
        utils.HandleValidationErrors(err, ctx)
        return
    }

    claims := jwt.Get(ctx).(*utils.AccessToken)
    if req.SenderID != claims.ID {
        ctx.StatusCode(iris.StatusForbidden)
        return
    }

    message := models.Message{
        ConversationID: req.ConversationID,
        SenderID:       req.SenderID,
        ReceiverID:     req.ReceiverID,
        Text:           req.Text,
        Read:           false, // Set the Read field to false
    }

    storage.DB.Create(&message)
    ctx.JSON(message)
}

func MarkMessageAsRead(ctx iris.Context) {
    var req MarkMessageAsReadInput
    err := ctx.ReadJSON(&req)
    if err != nil {
        utils.HandleValidationErrors(err, ctx)
        return
    }

    claims := jwt.Get(ctx).(*utils.AccessToken)
    if req.ReceiverID != claims.ID {
        ctx.StatusCode(iris.StatusForbidden)
        return
    }

    var message models.Message
    result := storage.DB.Where("id = ?", req.MessageID).First(&message)
    if result.Error != nil {
        utils.CreateNotFound(ctx)
        return
    }

    message.Read = true
    storage.DB.Save(&message)

    ctx.JSON(message)
}

type MarkMessageAsReadInput struct {
    MessageID  uint `json:"messageID" validate:"required"`
    ReceiverID uint `json:"receiverID" validate:"required"`
}

type CreateMessageInput struct {
	ConversationID uint   `json:"conversationID" validate:"required"`
	SenderID       uint   `json:"senderID" validate:"required"`
	ReceiverID     uint   `json:"receiverID" validate:"required"`
	Text           string `json:"text" validate:"required,lt=5000"`
}
