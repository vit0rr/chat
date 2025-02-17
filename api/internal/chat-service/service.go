package chatservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/pkg/database/repositories"
	"github.com/vit0rr/chat/pkg/deps"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	rooms sync.Map
)

// Room represents a chat room containing connected clients
// and a mutex for safe concurrent access
type Room struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

// Client represents a connected websocket client with associated metadata
type Client struct {
	conn     *websocket.Conn // WebSocket connection
	roomID   string          // ID of the room client is connected to
	userID   string          // Unique identifier for the client
	nickname string          // Display name of the client
	mu       sync.Mutex      // Mutex for thread-safe operations
}

// MessageType defines the type of messages that can be sent
type MessageType string

const (
	TextMessage   MessageType = "text"   // Regular chat messages
	SystemMessage MessageType = "system" // System notifications and alerts
)

// ChatMessage represents a message in the chat system
type ChatMessage struct {
	Type      MessageType `json:"type"`      // Type of message (text/system)
	Content   string      `json:"content"`   // Actual message content
	RoomID    string      `json:"room_id"`   // Room the message belongs to
	SenderID  string      `json:"sender_id"` // ID of message sender
	Nickname  string      `json:"nickname"`  // Sender's display name
	Timestamp time.Time   `json:"timestamp"` // When message was sent
}

// Service handles the chat service operations including WebSocket,
// MongoDB, and Redis interactions
type Service struct {
	deps  *deps.Deps
	Mongo *mongo.Database
	redis *redis.Client
}

type Response struct {
	Message string   `json:"message"`
	Keys    []string `json:"keys"`
}

type NewChatServiceBody struct {
	Message string `json:"message"`
}

type RegisterUserBody struct {
	UserID   string `json:"user_id"`
	RoomID   string `json:"room_id"`
	Nickname string `json:"nickname"`
}

type GetMessagesQuery struct {
	RoomID   string `json:"room_id"`
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type RegisterUserResponse struct {
	UserID   string `json:"user_id"`
	RoomID   string `json:"room_id"`
	Nickname string `json:"nickname"`
}

type LockRoomBody struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

func NewService(deps *deps.Deps, db *mongo.Database, redisClient *redis.Client) *Service {
	return &Service{
		deps:  deps,
		Mongo: db,
		redis: redisClient,
	}
}

// @summary WebSocket
// @description WebSocket
// @router /api/ws/ [get]
// @success 200         {object}    Response                         "Successfully processed chat"
// @failure 400         {object}    Response                         "Invalid request body"
// @failure 500         {object}    Response                         "Internal server error during processing"
func (s *Service) WebSocket(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := context.Background()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("websocket accept error: %v", err)
	}

	roomID := r.URL.Query().Get("room_id")
	userID := r.URL.Query().Get("user_id")
	nickname := r.URL.Query().Get("nickname")

	room, err := repositories.GetRooms(ctx, s.Mongo, repositories.GetRoomData{
		RoomID: roomID,
	})

	if err != nil {
		log.Error(ctx, "Failed to get room", log.ErrAttr(err))
		conn.Close(websocket.StatusInternalError, "Failed to get room")
		return nil, fmt.Errorf("failed to get room: %v", err)
	}

	if room == nil {
		log.Error(ctx, "Room not found", log.AnyAttr("room_id", roomID))
		conn.Close(websocket.StatusInternalError, "Room not found")
		return nil, fmt.Errorf("room not found")
	}

	userAuthorized := false
	for _, user := range room.Users {
		if user.UserID == userID {
			userAuthorized = true
			break
		}
	}

	if !userAuthorized {
		log.Error(ctx, "User not authorized to join room",
			log.AnyAttr("room_id", roomID),
			log.AnyAttr("user_id", userID))
		conn.Close(websocket.StatusInternalError, "User not authorized to join room")
		return nil, fmt.Errorf("user not authorized to join room")
	}

	client := &Client{
		conn:     conn,
		roomID:   roomID,
		userID:   userID,
		nickname: nickname,
		mu:       sync.Mutex{},
	}

	// Subscribe to room channel
	pubsub := s.redis.Subscribe(ctx, roomID)
	defer pubsub.Close()

	// Add client to Redis room set with expiration
	roomKey := fmt.Sprintf("room:%s:clients", roomID)
	pipe := s.redis.Pipeline()
	pipe.SAdd(ctx, roomKey, userID)
	pipe.Expire(ctx, roomKey, 24*time.Hour) // Set 24h expiration, adjust as needed
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error(ctx, "Failed to add client to Redis", log.ErrAttr(err))
		conn.Close(websocket.StatusInternalError, "Failed to initialize connection")
		return nil, err
	}

	// Cleanup on disconnect
	defer func() {
		s.redis.SRem(ctx, roomKey, userID)
		// If room is empty, delete the key
		if members, _ := s.redis.SCard(ctx, roomKey).Result(); members == 0 {
			s.redis.Del(ctx, roomKey)
		}
	}()

	// Handle incoming messages
	go func() {
		defer func() {
			conn.Close(websocket.StatusNormalClosure, "")
		}()

		// Listen for messages from Redis
		ch := pubsub.Channel()
		for msg := range ch {
			var chatMsg ChatMessage
			if err := json.Unmarshal([]byte(msg.Payload), &chatMsg); err != nil {
				log.Error(ctx, "Failed to unmarshal message", log.ErrAttr(err))
				continue
			}

			client.mu.Lock()
			err := wsjson.Write(ctx, conn, chatMsg)
			client.mu.Unlock()

			if err != nil {
				log.Error(ctx, "Failed to send message to client", log.ErrAttr(err))
				return
			}
		}
	}()

	// Handle WebSocket messages
	for {
		var message ChatMessage
		err := wsjson.Read(ctx, conn, &message)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				log.Error(ctx, "Error reading message", log.ErrAttr(err))
			}
			return nil, err
		}

		// Check room lock status
		room, err := repositories.GetRooms(ctx, s.Mongo, repositories.GetRoomData{
			RoomID: roomID,
		})
		if err != nil {
			log.Error(ctx, "Failed to check room lock status", log.ErrAttr(err))
			continue
		}

		if room.LockedBy != "" && room.LockedBy != userID {
			client.mu.Lock()
			wsjson.Write(ctx, conn, ChatMessage{
				Type:      SystemMessage,
				Content:   "Room is locked. Messages cannot be sent.",
				RoomID:    roomID,
				Timestamp: time.Now(),
			})
			client.mu.Unlock()
			continue
		}

		message.Timestamp = time.Now()
		message.SenderID = userID
		message.Nickname = nickname
		message.RoomID = roomID

		// Broadcast message using Redis
		s.broadcastToRoom(ctx, roomID, message)
	}
}

func (s *Service) RegisterUser(c context.Context, b io.ReadCloser, dbClient *mongo.Database) (interface{}, error) {
	var body RegisterUserBody
	err := json.NewDecoder(b).Decode(&body)
	if err != nil {
		log.Error(c, "Failed to decode RegisterUserBody", log.ErrAttr(err))
		return nil, fmt.Errorf("failed to decode RegisterUserBody: %v", err)
	}
	defer b.Close()

	// Check if user is already registered in the room
	existingRoom, err := repositories.GetRooms(c, dbClient, repositories.GetRoomData{
		RoomID: body.RoomID,
	})
	if err != nil {
		log.Error(c, "Failed to check existing room", log.ErrAttr(err))
		return nil, fmt.Errorf("failed to check existing room: %v", err)
	}

	// If room exists, check if user is already registered
	if existingRoom != nil {
		for _, user := range existingRoom.Users {
			if user.UserID == body.UserID {
				log.Error(c, "User already registered in room",
					log.AnyAttr("room_id", body.RoomID),
					log.AnyAttr("user_id", body.UserID))
				return existingRoom, fmt.Errorf("user already registered in room")
			}
		}
	}

	// Register new user in room
	_, err = repositories.CreateRoom(c, dbClient, repositories.CreateRoomData{
		UserID:   body.UserID,
		RoomID:   body.RoomID,
		Nickname: body.Nickname,
	})

	if err != nil {
		log.Error(c, "Failed to create room", log.ErrAttr(err))
		return nil, fmt.Errorf("failed to create room: %v", err)
	}

	// Get the updated room to return
	updatedRoom, err := repositories.GetRooms(c, dbClient, repositories.GetRoomData{
		RoomID: body.RoomID,
	})
	if err != nil {
		log.Error(c, "Failed to get updated room", log.ErrAttr(err))
		return nil, fmt.Errorf("failed to get updated room: %v", err)
	}

	return updatedRoom, nil
}

func (s *Service) LockRoom(c context.Context, b io.ReadCloser) (interface{}, error) {
	var body LockRoomBody
	err := json.NewDecoder(b).Decode(&body)
	if err != nil {
		log.Error(c, "Failed to decode LockRoomBody", log.ErrAttr(err))
		return nil, fmt.Errorf("failed to decode LockRoomBody: %v", err)
	}
	defer b.Close()

	if body.RoomID == "" || body.UserID == "" {
		return nil, fmt.Errorf("room_id and user_id are required")
	}

	room, err := repositories.GetRooms(c, s.Mongo, repositories.GetRoomData{
		RoomID: body.RoomID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %v", err)
	}

	if room == nil {
		return nil, fmt.Errorf("room not found")
	}

	userAuthorized := false
	for _, user := range room.Users {
		if user.UserID == body.UserID {
			userAuthorized = true
			break
		}
	}

	if !userAuthorized {
		return nil, fmt.Errorf("user not authorized to lock room")
	}

	collection := s.Mongo.Collection(constants.RoomsCollection)
	_, err = collection.UpdateOne(c,
		bson.M{"_id": body.RoomID},
		bson.M{"$set": bson.M{"lockedBy": body.UserID}})
	if err != nil {
		return nil, fmt.Errorf("failed to lock room: %v", err)
	}

	roomToLock, err := repositories.GetRooms(c, s.Mongo, repositories.GetRoomData{
		RoomID: body.RoomID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	userNickname := ""
	for _, user := range roomToLock.Users {
		if user.UserID == body.UserID {
			userNickname = user.Nickname
		}
	}

	s.broadcastToRoom(c, body.RoomID, ChatMessage{
		Type:      SystemMessage,
		Content:   fmt.Sprintf("Room has been locked by %s", userNickname),
		RoomID:    body.RoomID,
		Timestamp: time.Now(),
	})

	return map[string]string{"status": "room locked"}, nil
}

func (s *Service) GetMessages(ctx context.Context, query GetMessagesQuery) ([]ChatMessage, error) {
	if query.RoomID == "" {
		return nil, fmt.Errorf("room_id is required")
	}

	page := 1
	limit := 50

	if query.PageStr != "" {
		if p, err := strconv.Atoi(query.PageStr); err == nil && p > 0 {
			page = p
		}
	}

	if query.LimitStr != "" {
		if l, err := strconv.Atoi(query.LimitStr); err == nil && l > 0 {
			limit = l
		}
	}

	skip := int64((page - 1) * limit)
	cursor, err := repositories.GetMessages(ctx, s.Mongo, repositories.GetMessagesData{
		RoomID: query.RoomID,
		Limit:  int64(limit),
		Skip:   skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []ChatMessage
	for cursor.Next(ctx) {
		var msg repositories.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Error(ctx, "Failed to decode message", log.ErrAttr(err))
			continue
		}

		messages = append(messages, ChatMessage{
			Type:      TextMessage,
			Content:   msg.Message,
			RoomID:    msg.RoomID,
			Nickname:  msg.Nickname,
			SenderID:  msg.FromUserID,
			Timestamp: msg.CreatedAt,
		})
	}

	return messages, nil
}

// broadcastToRoom sends a message to all clients in a room by:
// 1. Saving the message to MongoDB for persistence
// 2. Publishing the message to Redis for real-time distribution
func (s *Service) broadcastToRoom(ctx context.Context, roomID string, message ChatMessage) {
	// Save message to MongoDB
	_, err := repositories.CreateMessage(ctx, s.Mongo, repositories.CreateMessageData{
		RoomID:     message.RoomID,
		Message:    message.Content,
		FromUserID: message.SenderID,
		Nickname:   message.Nickname,
	})

	if err != nil {
		log.Error(ctx, "Failed to save message to database",
			log.AnyAttr("room_id", roomID),
			log.AnyAttr("error", err))
	}

	// Publish message to Redis channel
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Error(ctx, "Failed to marshal message",
			log.AnyAttr("room_id", roomID),
			log.AnyAttr("error", err))
		return
	}

	err = s.redis.Publish(ctx, roomID, messageJSON).Err()
	if err != nil {
		log.Error(ctx, "Failed to publish message to Redis",
			log.AnyAttr("room_id", roomID),
			log.AnyAttr("error", err))
	}
}
