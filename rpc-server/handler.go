package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TikTokTechImmersion/assignment_demo_2023/rpc-server/kitex_gen/rpc"
)

// IMServiceImpl implements the last service interface defined in the IDL.
type IMServiceImpl struct{}

// Obtain and sort roomID in ascending order
func getRoomID(chat string) (string, error) {
	var roomID string

	lowercase := strings.ToLower(chat)
	senders := strings.Split(lowercase, ":")

	//// Debug
	//fmt.Println("senders:", senders)
	//fmt.Println("Length of the senders:", len(senders))

	if len(senders) != 2 {
		err := fmt.Errorf("invalid Chat ID '%s', should be in the format of user1:user2", chat)
		return "", err
	}

	firstPerson, secondPerson := senders[0], senders[1]
	comparison := strings.Compare(firstPerson, secondPerson)
	if comparison == 0 {
		return "", fmt.Errorf("invalid ChatID '%s', ChatID must consist of 2 different person", chat)
	} else if comparison == 1 {
		roomID = fmt.Sprintf("%s:%s", secondPerson, firstPerson)
	} else if comparison == -1 {
		roomID = fmt.Sprintf("%s:%s", firstPerson, secondPerson)
	}

	return roomID, nil
}

// Checks if the send request is valid
func validateSendRequest(req *rpc.SendRequest) error {
	senders := strings.Split(req.Message.Chat, ":")
	if len(senders) != 2 {
		err := fmt.Errorf("invalid ChatID '%s', should be in the format of user1:user2", senders)
		return err
	}
	sender1, sender2 := senders[0], senders[1]

	if req.Message.GetSender() != sender1 && req.Message.GetSender() != sender2 {
		err := fmt.Errorf("sender '%s' not in the chat room", req.Message.GetSender())
		return err
	}

	return nil
}

func (s *IMServiceImpl) Send(ctx context.Context, req *rpc.SendRequest) (*rpc.SendResponse, error) {
	if err := validateSendRequest(req); err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	message := &Message{
		Message:   req.Message.GetText(),
		Sender:    req.Message.GetSender(),
		Timestamp: timestamp,
	}

	roomID, err := getRoomID(req.Message.GetChat())
	if err != nil {
		return nil, err
	}

	err = rdb.SaveMessage(ctx, roomID, message)
	if err != nil {
		return nil, err
	}

	resp := rpc.NewSendResponse()
	resp.Code, resp.Msg = 0, "success"
	fmt.Println("Successful sending of info")
	return resp, nil
}

func (s *IMServiceImpl) Pull(ctx context.Context, req *rpc.PullRequest) (*rpc.PullResponse, error) {
	roomID, err := getRoomID(req.GetChat())
	if err != nil {
		return nil, err
	}

	limit := int64(req.GetLimit())
	if limit == 0 {
		limit = 10 // default limit 10
	}
	start := req.GetCursor()
	end := start + limit // did not minus 1 on purpose for hasMore check later on

	messages, err := rdb.GetMessagesByRoomID(ctx, roomID, start, end, req.GetReverse())
	if err != nil {
		return nil, err
	}

	respMessages := make([]*rpc.Message, 0)
	var counter int64 = 0
	var nextCursor int64 = 0
	hasMore := false
	for _, msg := range messages {
		if counter+1 > limit {
			// having extra value here means it has more data
			hasMore = true
			nextCursor = end
			break // do not return the last message
		}
		temp := &rpc.Message{
			Chat:     req.GetChat(),
			Text:     msg.Message,
			Sender:   msg.Sender,
			SendTime: msg.Timestamp,
		}
		respMessages = append(respMessages, temp)
		counter += 1
	}

	resp := rpc.NewPullResponse()
	resp.Messages = respMessages
	resp.Code = 0
	resp.Msg = "success"
	resp.HasMore = &hasMore
	resp.NextCursor = &nextCursor

	fmt.Println("Successful pulling of info")

	return resp, nil
}