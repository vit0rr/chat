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
	"github.com/vit0rr/chat/api/constants"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/pkg/database/repositories"
	"github.com/vit0rr/chat/pkg/deps"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Client represents a connected websocket client with associated metadata
type Client struct {
	conn     *websocket.Conn // WebSocket connection
	roomID   string          // ID of the room client is connected to
	userID   string          // Unique identifier for the client
	nickname string          // Display name of the client
	mu       sync.Mutex      // Mutex for thread-safe operations
	isOnline bool            // Online status of the client
}

// MessageType defines the type of messages that can be sent
type MessageType string

const (
	TextMessage   MessageType = "text"   // Regular chat messages
	SystemMessage MessageType = "system" // System notifications and alerts
	MaxMessageLen             = 5000     // Maximum characters allowed per message
)

// ChatMessage represents a message in the chat system
type ChatMessage struct {
	Type      MessageType `json:"type"`      // Type of message (text/system)
	Content   string      `json:"content"`   // Actual message content
	RoomId    string      `json:"room_id"`   // Room the message belongs to
	SenderId  string      `json:"sender_id"` // ID of message sender
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

// Response is a generic response structure
type Response struct {
	Message string   `json:"message"`
	Keys    []string `json:"keys"`
}

// NewChatServiceBody is the body of the new chat service
type NewChatServiceBody struct {
	Message string `json:"message"`
}

// RegisterUserBody is the body of the register user
type RegisterUserBody struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

// GetMessagesQuery is the query of the get messages
type GetMessagesQuery struct {
	RoomID   string `json:"room_id"`
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type GetOnlineUsersFromAllRoomsQuery struct {
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type GetRoomsQuery struct {
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type GetUsersQuery struct {
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type GetUsersWhoSentMessagesInTheLastDaysQuery struct {
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
	Days     int    `json:"days"`
}

type GetUserContactsQuery struct {
	ID       string `json:"id"`
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

// RegisterUserResponse is the response of the register user
type RegisterUserResponse struct {
	UserID   string `json:"user_id"`
	RoomID   string `json:"room_id"`
	Nickname string `json:"nickname"`
}

// LockRoomBody is the body of the lock room
type LockRoomBody struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

type Error struct {
	ErrorMessage *string `json:"error_message"`
	ErrorID      *string `json:"error_id"`
	ErrorCode    *int    `json:"error_code"`
}

type RoomsList struct {
	Rooms []RoomListDetails `json:"rooms"`
}

// Create the types to the GetRoom now
type RoomDetails struct {
	RoomId    string                 `json:"room_id"`
	Users     []repositories.UserRef `json:"users"`
	LockedBy  *string                `json:"locked_by,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type RoomListDetails struct {
	RoomID    string         `json:"room_id"`
	Users     []RoomListUser `json:"users"`
	LockedBy  *string        `json:"locked_by,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type RoomListUser struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
}

type ServiceError struct {
	Message string
	ID      string
	Code    int
}

func (e ServiceError) Error() string {
	return e.Message
}

func NewServiceError(key string) error {
	if msg, ok := constants.ErrorMessages[key]; ok {
		return ServiceError{
			Message: msg.Message,
			ID:      msg.ID,
			Code:    msg.Code,
		}
	}
	return ServiceError{
		Message: "Unknown error",
		ID:      "unknown_error",
		Code:    500,
	}
}

// NewService creates a new chat service
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
		if user.ID == userID {
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
		isOnline: true,
	}

	// Subscribe to room channel
	pubsub := s.redis.Subscribe(ctx, roomID)
	defer pubsub.Close()

	// Add client to Redis room set with expiration
	roomKey := fmt.Sprintf("room:%s:clients", roomID)
	onlineKey := fmt.Sprintf("room:%s:online", roomID)
	pipe := s.redis.Pipeline()
	pipe.SAdd(ctx, roomKey, userID)
	pipe.SAdd(ctx, onlineKey, userID)
	pipe.Expire(ctx, roomKey, 24*time.Hour)
	pipe.Expire(ctx, onlineKey, 1*time.Second)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error(ctx, "Failed to add client to Redis", log.ErrAttr(err))
		conn.Close(websocket.StatusInternalError, "Failed to initialize connection")
		return nil, err
	}

	repositories.UpdateUser(ctx, s.Mongo, repositories.UpdateUserData{
		UserID:   userID,
		Activity: &[]string{"online"}[0],
	})

	// Cleanup on disconnect
	defer func() {
		s.redis.SRem(ctx, roomKey, userID)
		s.redis.SRem(ctx, onlineKey, userID)

		repositories.UpdateUser(ctx, s.Mongo, repositories.UpdateUserData{
			UserID:   userID,
			Activity: &[]string{"offline"}[0],
		})

		// If room is empty, delete the keys
		if members, _ := s.redis.SCard(ctx, roomKey).Result(); members == 0 {
			s.redis.Del(ctx, roomKey, onlineKey)
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

		// Validate message length
		if len(message.Content) > MaxMessageLen {
			client.mu.Lock()
			wsjson.Write(ctx, conn, ChatMessage{
				Type:      SystemMessage,
				Content:   fmt.Sprintf("Message exceeds maximum length of %d characters", MaxMessageLen),
				RoomId:    roomID,
				Timestamp: time.Now(),
			})
			client.mu.Unlock()
			continue
		}

		// Check room lock status
		room, err := repositories.GetRooms(ctx, s.Mongo, repositories.GetRoomData{
			RoomID: roomID,
		})
		if err != nil {
			log.Error(ctx, "Failed to check room lock status", log.ErrAttr(err))
			continue
		}

		// If the room is locked by this user, unlock it when they send any message
		if room.LockedBy == userID {
			collection := s.Mongo.Collection(constants.RoomsCollection)
			_, err = collection.UpdateOne(ctx,
				bson.M{"_id": roomID},
				bson.M{"$set": bson.M{"lockedBy": ""}})
			if err != nil {
				log.Error(ctx, "Failed to unlock room", log.ErrAttr(err))
				continue
			}

			// Broadcast unlock message
			s.broadcastToRoom(ctx, roomID, ChatMessage{
				Type:      SystemMessage,
				Content:   fmt.Sprintf("Room has been unlocked by %s", nickname),
				RoomId:    roomID,
				Timestamp: time.Now(),
			})
		}

		// Check if user can send message
		if room.LockedBy != "" && room.LockedBy != userID {
			client.mu.Lock()
			wsjson.Write(ctx, conn, ChatMessage{
				Type:      SystemMessage,
				Content:   "Room is locked. Messages cannot be sent.",
				RoomId:    roomID,
				Timestamp: time.Now(),
			})
			client.mu.Unlock()
			continue
		}

		message.Timestamp = time.Now()
		message.SenderId = userID
		message.Nickname = nickname
		message.RoomId = roomID

		// Broadcast message using Redis
		s.broadcastToRoom(ctx, roomID, message)
	}
}

// @summary RegisterUser
// @description Will register a user in a room. If the user is already registered, it will return the room without error.
// @router /api/register-user/ [post]
// @success 200         {object}    RegisterUserResponse			 "Successfully processed chat"
// @failure 400         {object}    Response                         "Invalid request body"
// @failure 500         {object}    Response                         "Internal server error during processing"
func (s *Service) RegisterUser(c context.Context, b io.ReadCloser, db *mongo.Database, roomID string) (interface{}, Error) {
	var body RegisterUserBody
	err := json.NewDecoder(b).Decode(&body)
	if err != nil {
		log.Error(c, "Failed to decode RegisterUserBody", log.ErrAttr(err))
		if svcErr := NewServiceError(constants.FailedToDecodeBody); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_decode_body")
	}
	defer b.Close()

	// Check if user exists
	var user *repositories.User
	if body.UserID != "" {
		user, err = repositories.GetUser(c, db, repositories.GetUserData{
			UserID: body.UserID,
		})
	}

	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_get_user")
	}

	var userID string

	// Set userID from existing user or create a new one
	if user != nil {
		// Use existing user's ID
		userID = body.UserID
	} else {
		// Create new user
		newUser, err := repositories.CreateUser(c, db, repositories.CreateUserData{
			Nickname: body.Nickname,
		})

		if err != nil {
			log.Error(c, "Failed to create user", log.ErrAttr(err))
			return nil, newError("failed_to_create_user")
		}

		// Safely convert ObjectID to string
		if oid, ok := newUser.InsertedID.(primitive.ObjectID); ok {
			userID = oid.Hex()
		} else {
			log.Error(c, "Invalid InsertedID type", log.AnyAttr("type", fmt.Sprintf("%T", newUser.InsertedID)))
			return nil, newError("failed_to_create_user")
		}
	}

	// Check if user is already registered in the room
	existingRoom, err := repositories.GetRooms(c, db, repositories.GetRoomData{
		RoomID: roomID,
	})

	if err != nil {
		log.Error(c, constants.ErrorMessages[constants.FailedToCheckExistingRoom].Message, log.ErrAttr(err))
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_check_existing_room")
	}

	if existingRoom != nil {
		for _, user := range existingRoom.Users {
			if user.ID == body.UserID {
				// User is already registered, return the room without error
				log.Info(c, "User rejoining existing room",
					log.AnyAttr("room_id", roomID),
					log.AnyAttr("user_id", body.UserID))
				return existingRoom, Error{}
			}
		}
	}

	// Register new user in room
	_, err = repositories.CreateRoom(c, db, repositories.CreateRoomData{
		UserID:   userID,
		RoomID:   roomID,
		Nickname: body.Nickname,
	})

	if err != nil {
		log.Error(c, constants.ErrorMessages[constants.FailedToCreateOrUpdateRoom].Message, log.ErrAttr(err))
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_create_or_update_room")
	}

	// Get the updated room to return
	updatedRoom, err := repositories.GetRooms(c, db, repositories.GetRoomData{
		RoomID: roomID,
	})
	if err != nil {
		log.Error(c, "Failed to get updated room", log.ErrAttr(err))
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_get_updated_room")
	}

	return updatedRoom, Error{}
}

// @summary LockRoom
// @description Will lock a room for the user. If the room is already locked by the user, it will unlock the room.
// @router /api/lock-room/ [post]
// @success 200         {object}    Response                         "Successfully processed chat"
// @failure 400         {object}    Response                         "Invalid request body"
// @failure 500         {object}    Response                         "Internal server error during processing"
func (s *Service) LockRoom(c context.Context, b io.ReadCloser, roomID string) (interface{}, Error) {
	var body LockRoomBody
	err := json.NewDecoder(b).Decode(&body)
	if err != nil {
		log.Error(c, "Failed to decode LockRoomBody", log.ErrAttr(err))
		if svcErr := NewServiceError(constants.FailedToDecodeBody); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_decode_body")
	}
	defer b.Close()

	if body.UserID == "" {
		if svcErr := NewServiceError(constants.UserIDRequired); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("user_id_required")
	}

	room, err := repositories.GetRooms(c, s.Mongo, repositories.GetRoomData{
		RoomID: roomID,
	})
	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_get_room")
	}

	if room == nil {
		if svcErr := NewServiceError(constants.RoomNotFound); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("room_not_found")
	}

	userAuthorized := false
	for _, user := range room.Users {
		if user.ID == body.UserID {
			userAuthorized = true
			break
		}
	}

	if !userAuthorized {
		if svcErr := NewServiceError(constants.UserNotAuthorizedToLockRoom); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("user_not_authorized_to_lock_room")
	}

	collection := s.Mongo.Collection(constants.RoomsCollection)
	_, err = collection.UpdateOne(c,
		bson.M{"_id": roomID},
		bson.M{"$set": bson.M{"lockedBy": body.UserID}})
	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_lock_room")
	}

	roomToLock, err := repositories.GetRooms(c, s.Mongo, repositories.GetRoomData{
		RoomID: roomID,
	})
	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_get_room_to_lock")
	}

	userNickname := ""
	for _, user := range roomToLock.Users {
		if user.ID == body.UserID {
			userNickname = user.Nickname
		}
	}

	// Check if room is already locked by this user
	if room.LockedBy == body.UserID {
		// Unlock the room
		_, err = collection.UpdateOne(c,
			bson.M{"_id": roomID},
			bson.M{"$set": bson.M{"lockedBy": ""}})
		if err != nil {
			if svcErr := NewServiceError(err.Error()); svcErr != nil {
				if serviceErr, ok := svcErr.(ServiceError); ok {
					return nil, Error{
						ErrorMessage: &serviceErr.Message,
						ErrorID:      &serviceErr.ID,
						ErrorCode:    &serviceErr.Code,
					}
				}
			}

			return nil, newError("failed_to_unlock_room")
		}

		s.broadcastToRoom(c, roomID, ChatMessage{
			Type:      SystemMessage,
			Content:   fmt.Sprintf("Room has been unlocked by %s", userNickname),
			RoomId:    roomID,
			Timestamp: time.Now(),
		})

		return map[string]string{"status": "room unlocked"}, Error{}
	}

	s.broadcastToRoom(c, roomID, ChatMessage{
		Type:      SystemMessage,
		Content:   fmt.Sprintf("Room has been locked by %s", userNickname),
		RoomId:    roomID,
		Timestamp: time.Now(),
	})

	return map[string]string{"status": "room locked"}, Error{}
}

// @summary GetMessages
// @description Will return the messages of a room. It receives a room_id, page and limit by query params.
// @router /api/get-messages/ [get]
// @success 200         {object}    []ChatMessage			 "Successfully processed chat"
// @failure 400         {object}    Response                         "Invalid request body"
// @failure 500         {object}    Response                         "Internal server error during processing"
func (s *Service) GetMessages(ctx context.Context, query GetMessagesQuery) ([]ChatMessage, Error) {
	if query.RoomID == "" {
		if svcErr := NewServiceError(constants.RoomIDRequired); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}
	}

	room, err := repositories.GetRoom(ctx, s.Mongo, repositories.GetRoomData{
		RoomID: query.RoomID,
	})
	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}
	}

	if room == nil {
		return nil, newError("room_not_found")
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
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_get_messages")
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
			RoomId:    msg.RoomID,
			Nickname:  msg.Nickname,
			SenderId:  msg.FromUserID,
			Timestamp: msg.CreatedAt,
		})
	}

	return messages, Error{}
}

// broadcastToRoom sends a message to all clients in a room by:
// 1. Saving the message to MongoDB for persistence
// 2. Publishing the message to Redis for real-time distribution
func (s *Service) broadcastToRoom(ctx context.Context, roomID string, message ChatMessage) {
	// Save message to MongoDB
	_, err := repositories.CreateMessage(ctx, s.Mongo, repositories.CreateMessageData{
		RoomID:     message.RoomId,
		Message:    message.Content,
		FromUserID: message.SenderId,
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

func (s *Service) GetUsers(ctx context.Context, query GetUsersQuery) (interface{}, error) {
	cursor, err := repositories.GetUsers(ctx, s.Mongo, repositories.GetUserData{})

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	users := []repositories.User{}
	for cursor.Next(ctx) {
		var user repositories.User
		if err := cursor.Decode(&user); err != nil {
			log.Error(ctx, "Failed to decode user", log.ErrAttr(err))
			continue
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (interface{}, error) {
	user, err := repositories.GetUser(ctx, s.Mongo, repositories.GetUserData{
		UserID: userID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	rooms, err := repositories.GetAllRoomsWhereUserIsRegistered(ctx, s.Mongo, repositories.GetUserData{
		UserID: userID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get rooms: %v", err)
	}

	return map[string]interface{}{
		"user":  user,
		"rooms": rooms,
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, body io.ReadCloser) (interface{}, error) {
	defer body.Close()

	var user repositories.CreateUserData
	err := json.NewDecoder(body).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode user: %v", err)
	}

	result, err := repositories.CreateUser(ctx, s.Mongo, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return result, nil
}

func (s *Service) UpdateUser(ctx context.Context, ID string, body io.ReadCloser) (interface{}, error) {
	defer body.Close()

	var user repositories.UpdateUserData
	err := json.NewDecoder(body).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode user: %v", err)
	}

	user.UserID = ID

	result, err := repositories.UpdateUser(ctx, s.Mongo, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	return result, nil
}

func (s *Service) GetRoom(ctx context.Context, roomID string) (RoomDetails, Error) {
	room, err := repositories.GetRoom(ctx, s.Mongo, repositories.GetRoomData{
		RoomID: roomID,
	})

	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return RoomDetails{}, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return RoomDetails{}, newError("failed_to_get_room")

	}

	if room == nil {
		return RoomDetails{}, newError("room_not_found")
	}

	return RoomDetails{
		RoomId:    room.ID,
		Users:     room.Users,
		LockedBy:  &room.LockedBy,
		CreatedAt: room.CreatedAt,
		UpdatedAt: room.UpdatedAt,
	}, Error{}
}

// @summary GetRooms
// @description Returns a paginated list of all chat rooms
// @router /api/get-rooms/ [get]
// @param   page   query    integer  false  "Page number (default: 1)"  minimum(1)
// @param   limit  query    integer  false  "Items per page (default: 10)"  minimum(1) maximum(100)
// @success 200    {object} RoomList "List of chat rooms retrieved successfully"
// @failure 404    {object} Error    "Room not found"
// @failure 500    {object} Error    "Internal server error during processing"
func (s *Service) GetRooms(ctx context.Context, query GetRoomsQuery) (RoomsList, Error) {
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

	cursor, err := repositories.GetRoomsCursor(ctx, s.Mongo, repositories.GetRoomsCursorData{
		Limit: int64(limit),
		Skip:  int64((page - 1) * limit),
	})
	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return RoomsList{}, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return RoomsList{}, newError("failed_to_get_rooms")
	}

	var rooms []repositories.Room
	for cursor.Next(ctx) {
		var room repositories.Room
		if err := cursor.Decode(&room); err != nil {
			log.Error(ctx, "Failed to decode room", log.ErrAttr(err))
			continue
		}

		rooms = append(rooms, room)
	}

	responseRooms := []RoomListDetails{}
	for _, room := range rooms {
		responseUsers := []RoomListUser{}
		for _, user := range room.Users {
			responseUsers = append(responseUsers, RoomListUser{
				Id:       user.ID,
				Nickname: user.Nickname,
			})
		}

		responseRooms = append(responseRooms, RoomListDetails{
			RoomID:    room.ID,
			Users:     responseUsers,
			LockedBy:  &room.LockedBy,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
		})
	}

	return RoomsList{
		Rooms: responseRooms,
	}, Error{}
}

// GetOnlineUsers returns a list of online users in a room
func (s *Service) GetOnlineUsersFromARoom(ctx context.Context, roomID string) ([]repositories.User, Error) {
	users, err := repositories.GetAllOnlineUsersFromARoom(ctx, s.Mongo, repositories.GetAllOnlineUsersFromARoomData{
		RoomID: roomID,
	})

	if err != nil {
		if svcErr := NewServiceError(err.Error()); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError(constants.ErrorMessages[constants.FailedToGetUsers].ID)
	}

	return users, Error{}
}

func (s *Service) GetOnlineUsersFromAllRooms(ctx context.Context, query GetOnlineUsersFromAllRoomsQuery) ([]repositories.User, error) {
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
	cursor, err := repositories.GetAllOnlineUsers(ctx, s.Mongo, repositories.GetAllOnlineUsersData{
		Limit: int64(limit),
		Skip:  skip,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get online users: %v", err)
	}
	defer cursor.Close(ctx)

	var users []repositories.User
	for cursor.Next(ctx) {
		var user repositories.User
		if err := cursor.Decode(&user); err != nil {
			log.Error(ctx, "Failed to decode user", log.ErrAttr(err))
			continue
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *Service) GetTotalMessagesSent(ctx context.Context) (int64, error) {
	total, err := repositories.TotalMessagesSent(ctx, s.Mongo)
	if err != nil {
		return 0, fmt.Errorf("failed to get total messages sent: %v", err)

	}

	return total, nil
}

func (s *Service) GetTotalMessagesSentInARoom(ctx context.Context, roomID string) (int64, error) {
	room, err := repositories.GetRooms(ctx, s.Mongo, repositories.GetRoomData{
		RoomID: roomID,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get room: %v", err)
	}

	if room == nil {
		return 0, fmt.Errorf("room not found")
	}

	total, err := repositories.TotalMessagesSentInARoom(ctx, s.Mongo, repositories.GetTotalMessagesSentInARoomData{
		RoomID: roomID,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get total messages sent in a room: %v", err)
	}

	return total, nil
}

func (s *Service) GetUsersWhoSentMessagesInTheLastDays(ctx context.Context, query GetUsersWhoSentMessagesInTheLastDaysQuery) (interface{}, error) {
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
	cursor, err := repositories.UsersWhoSentMessagesInTheLastDays(ctx, s.Mongo, repositories.UsersWhoSentMessagesInTheLastDaysData{
		Limit: int64(limit),
		Skip:  skip,
		Days:  query.Days,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get users who sent messages in the last %d days: %v", query.Days, err)
	}

	var users []repositories.User
	for cursor.Next(ctx) {
		var user repositories.User
		if err := cursor.Decode(&user); err != nil {
			log.Error(ctx, "Failed to decode user", log.ErrAttr(err))
			continue
		}

		users = append(users, user)
	}

	return users, nil

}

func (s *Service) GetUserContacts(ctx context.Context, query GetUserContactsQuery) (interface{}, error) {
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
	cursor, err := repositories.GetUserContacts(ctx, s.Mongo, repositories.GetUserContactsData{
		UserID: query.ID,
		Limit:  int64(limit),
		Skip:   skip,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user contacts: %v", err)
	}

	var users []repositories.User
	for cursor.Next(ctx) {
		var user repositories.User
		if err := cursor.Decode(&user); err != nil {
			log.Error(ctx, "Failed to decode user", log.ErrAttr(err))
			continue
		}

		users = append(users, user)
	}

	return users, nil
}

func newError(errKey string) Error {
	errMsg := constants.ErrorMessages[errKey]
	return Error{
		ErrorMessage: &errMsg.Message,
		ErrorID:      &errMsg.ID,
		ErrorCode:    &errMsg.Code,
	}
}
