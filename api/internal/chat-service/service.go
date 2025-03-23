package chatservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/vit0rr/chat/api/constants"

	"github.com/google/uuid"
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
	conn            *websocket.Conn // WebSocket connection
	roomID          string          // ID of the room client is connected to
	userID          string          // Unique identifier for the client
	nickname        string          // Display name of the client
	mu              sync.Mutex      // Mutex for thread-safe operations
	isOnline        bool            // Online status of the client
	lastMessageTime time.Time       // Timestamp of the last message sent by this client
	connectionID    string          // Unique connection ID
}

// MessageType defines the type of messages that can be sent
type MessageType string

const (
	TextMessage   MessageType = "text"   // Regular chat messages
	SystemMessage MessageType = "system" // System notifications and alerts
	MaxMessageLen             = 5000     // Maximum characters allowed per message
	MessageDelay              = 1500 * time.Millisecond // 1.5 second delay between messages
)

// ChatMessage represents a message in the chat system
type ChatMessage struct {
	Type      MessageType `json:"type"`      // Type of message (text/system)
	Content   string      `json:"content"`   // Actual message content
	RoomId    string      `json:"room_id"`   // Room the message belongs to
	SenderId  string      `json:"sender_id"` // ID of message sender
	Nickname  string      `json:"nickname"`  // Sender's display name
	Timestamp time.Time   `json:"timestamp"` // When message was sent
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Service handles the chat service operations including WebSocket,
// MongoDB, and Redis interactions
type Service struct {
	deps  *deps.Deps
	Mongo *mongo.Database
	redis *redis.Client
}

// RegisterUserBody is the body of the register user
type RegisterUserBody struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

type GetMessagesQuery struct {
	RoomID   string `json:"room_id"`
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
}

type GetRoomsQuery struct {
	PageStr  string `json:"page_str"`
	LimitStr string `json:"limit_str"`
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
	service := &Service{
		deps:  deps,
		Mongo: db,
		redis: redisClient,
	}
	
	go service.monitorConnections()
	
	return service
}

// @summary Real-time Chat WebSocket Connection
// @description Establishes a WebSocket connection for real-time messaging in a chat room
// @tags websocket,rooms
// @router /api/v1/ws [get]
// @param token query string true "Authentication token (required)"
// @param user_id query string true "User ID (required)"
// @param room_id query string true "Room ID (required)"
// @param nickname query string true "User's display name (required)"
// @produce application/json
// @success 101 {object} ChatMessage "WebSocket connection successfully upgraded"
// @failure 400 {string} string "Missing required parameters or invalid request"
// @failure 401 {string} string "Unauthorized - Missing or invalid token"
// @failure 403 {string} string "Forbidden - User not authorized to join room"
// @failure 404 {string} string "Room not found"
// @failure 500 {string} string "Internal server error"
func (s *Service) WebSocket(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	log.Info(ctx, "Token", log.AnyAttr("token", token))
	if token == "" {
		log.Error(ctx, "Missing authentication token", log.AnyAttr("token", token))
		return nil, fmt.Errorf("missing authentication token")
	}
	
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("websocket accept error: %v", err)
	}
	requestedUserID := r.URL.Query().Get("user_id")

	roomID := r.URL.Query().Get("room_id")
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
		if user.ID == requestedUserID {
			userAuthorized = true
			break
		}
	}

	if !userAuthorized {
		log.Error(ctx, "User not authorized to join room",
			log.AnyAttr("room_id", roomID),
			log.AnyAttr("user_id", requestedUserID))
		conn.Close(websocket.StatusInternalError, "User not authorized to join room")
		return nil, fmt.Errorf("user not authorized to join room")
	}

	connectionID := uuid.New().String()
	client := &Client{
		conn:            conn,
		roomID:          roomID,
		userID:          requestedUserID,
		nickname:        nickname,
		connectionID:    connectionID,
		mu:              sync.Mutex{},
		isOnline:        true,
		lastMessageTime: time.Now(),
	}

	if err := registerClient(ctx, s.redis, client); err != nil {
		log.Error(ctx, "Failed to register client", log.ErrAttr(err))
		conn.Close(websocket.StatusInternalError, "Failed to initialize connection")
		return nil, err
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	go startHeartbeat(heartbeatCtx, s.redis, client)

	defer func() {
		cancelHeartbeat()
		unregisterClient(ctx, s.redis, client)
		
		repositories.UpdateUser(ctx, s.Mongo, repositories.UpdateUserData{
			UserID:   requestedUserID,
			Activity: &[]string{"offline"}[0],
		})
	}()

	pubsub := s.redis.Subscribe(ctx, roomID)
	defer pubsub.Close()

	go func() {
		historyKey := fmt.Sprintf("room:%s:history", roomID)
		messages, err := s.redis.ZRevRangeByScore(ctx, historyKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: "+inf",
			Count: 50, 
		}).Result()
		
		if err == nil && len(messages) > 0 {
			for i := len(messages) - 1; i >= 0; i-- {
				var msg ChatMessage
				if err := json.Unmarshal([]byte(messages[i]), &msg); err != nil {
					continue
				}
				
				client.mu.Lock()
				wsjson.Write(ctx, conn, msg)
				client.mu.Unlock()
			}
		}
	}()

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var chatMsg ChatMessage
			if err := json.Unmarshal([]byte(msg.Payload), &chatMsg); err != nil {
				log.Error(ctx, "Failed to unmarshal message", log.ErrAttr(err))
				continue
			}
			
			if chatMsg.SenderId == requestedUserID && 
			   chatMsg.Type != SystemMessage &&
			   chatMsg.Metadata != nil && 
			   chatMsg.Metadata["connectionID"] == connectionID {
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

		canSend, timeToWait := deps.CheckAndUpdateMessageRateLimit(ctx, s.redis, requestedUserID, MessageDelay)
		if !canSend {
			broadcastMessage(ctx, s.redis, ChatMessage{
				Type:      SystemMessage,
				Content:   fmt.Sprintf("Please wait %.1f seconds before sending another message", timeToWait),
				RoomId:    roomID,
				Timestamp: time.Now(),
			})
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
		if room.LockedBy == requestedUserID {
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
		if room.LockedBy != "" && room.LockedBy != requestedUserID {
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
		message.SenderId = requestedUserID
		message.Nickname = nickname
		message.RoomId = roomID

		// Broadcast message using Redis
		s.broadcastToRoom(ctx, roomID, message)
	}
}

// @summary Register User to Room
// @description Adds a user to a chat room. Creates new user if needed. Returns existing room if user already registered.
// @tags rooms,users
// @router /api/v1/rooms/{roomId}/register-user [post]
// @param roomId path string true "Room ID (required)"
// @param body body RegisterUserBody true "User information for registration"
// @produce application/json
// @success 200 {object} repositories.Room "User successfully registered to room"
// @failure 400 {object} Error "Bad request or invalid input"
// @failure 404 {object} Error "Room not found"
// @failure 500 {object} Error "Internal server error"
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

// @summary Lock or Unlock Room
// @description Controls the lock status of a chat room. Locks room for exclusive use by a user or unlocks if already locked by same user.
// @tags rooms
// @router /api/v1/rooms/{roomId}/lock [post]
// @param roomId path string true "Room ID (required)"
// @param body body LockRoomBody true "User information for locking the room"
// @produce application/json
// @success 200 {object} map[string]string "Room lock status updated successfully"
// @failure 400 {object} Error "Bad request or missing required fields"
// @failure 403 {object} Error "User not authorized to lock room"
// @failure 404 {object} Error "Room not found"
// @failure 500 {object} Error "Internal server error"
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

// @summary Retrieve Room Messages
// @description Fetches paginated messages for a specific chat room
// @tags messages,rooms
// @router /api/v1/rooms/{roomId}/messages [get]
// @param roomId path string true "Room ID (required)"
// @param page query integer false "Page number (default: 1)" minimum(1)
// @param limit query integer false "Items per page (default: 50)" minimum(1) maximum(100)
// @produce application/json
// @success 200 {array} ChatMessage "Messages retrieved successfully"
// @failure 400 {object} Error "Bad request or missing room ID"
// @failure 404 {object} Error "Room not found"
// @failure 500 {object} Error "Internal server error"
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

func (s *Service) UpdateUser(ctx context.Context, ID string, body io.ReadCloser) (interface{}, Error) {
	defer body.Close()

	var user repositories.UpdateUserData
	err := json.NewDecoder(body).Decode(&user)
	if err != nil {
		log.Error(ctx, constants.ErrorMessages[constants.FailedToDecodeBody].Message, log.ErrAttr(err))
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

	user.UserID = ID

	result, err := repositories.UpdateUser(ctx, s.Mongo, user)
	if err != nil {
		log.Error(ctx, constants.ErrorMessages[constants.FailedToUpdateUser].Message, log.ErrAttr(err))
		if svcErr := NewServiceError(constants.FailedToUpdateUser); svcErr != nil {
			if serviceErr, ok := svcErr.(ServiceError); ok {
				return nil, Error{
					ErrorMessage: &serviceErr.Message,
					ErrorID:      &serviceErr.ID,
					ErrorCode:    &serviceErr.Code,
				}
			}
		}

		return nil, newError("failed_to_update_user")
	}

	return result, Error{}
}

// @summary Get Room Details
// @description Returns detailed information about a specific chat room by ID
// @tags rooms
// @router /api/v1/rooms/{roomId} [get]
// @param roomId path string true "Room ID (required)"
// @produce application/json
// @success 200 {object} RoomDetails "Room details retrieved successfully"
// @failure 400 {object} Error "Bad request"
// @failure 404 {object} Error "Room not found"
// @failure 500 {object} Error "Internal server error"
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

// @summary List All Chat Rooms
// @description Returns a paginated list of all available chat rooms with their users and status
// @tags rooms
// @router /api/v1/rooms [get]
// @param page query integer false "Page number (default: 1)" minimum(1)
// @param limit query integer false "Items per page (default: 50)" minimum(1) maximum(100)
// @produce application/json
// @success 200 {object} RoomsList "List of chat rooms retrieved successfully"
// @failure 400 {object} Error "Bad request"
// @failure 401 {object} Error "Unauthorized"
// @failure 500 {object} Error "Internal server error"
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

func newError(errKey string) Error {
	errMsg := constants.ErrorMessages[errKey]
	return Error{
		ErrorMessage: &errMsg.Message,
		ErrorID:      &errMsg.ID,
		ErrorCode:    &errMsg.Code,
	}
}

func registerClient(ctx context.Context, redis *redis.Client, client *Client) error {
	clientKey := fmt.Sprintf("client:%s", client.userID)
	roomKey := fmt.Sprintf("room:%s:members", client.roomID)
	
	pipe := redis.Pipeline()
	
	pipe.HSet(ctx, clientKey, map[string]interface{}{
		"roomID": client.roomID,
		"nickname": client.nickname,
		"connectionID": client.connectionID,
		"lastSeen": time.Now().Unix(),
	})
	pipe.Expire(ctx, clientKey, 24*time.Hour)
	
	pipe.SAdd(ctx, roomKey, client.userID)
	pipe.Expire(ctx, roomKey, 24*time.Hour)
	
	pipe.SAdd(ctx, "users:online", client.userID)
	
	_, err := pipe.Exec(ctx)
	return err
}

func unregisterClient(ctx context.Context, redis *redis.Client, client *Client) error {
	clientKey := fmt.Sprintf("client:%s", client.userID)
	roomKey := fmt.Sprintf("room:%s:members", client.roomID)
	
	pipe := redis.Pipeline()
	pipe.Del(ctx, clientKey)
	pipe.SRem(ctx, roomKey, client.userID)
	pipe.SRem(ctx, "users:online", client.userID)
	
	_, err := pipe.Exec(ctx)
	return err
}

func heartbeat(ctx context.Context, redis *redis.Client, userID string) error {
	clientKey := fmt.Sprintf("client:%s", userID)
	return redis.HSet(ctx, clientKey, "lastSeen", time.Now().Unix()).Err()
}

func startHeartbeat(ctx context.Context, redis *redis.Client, client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			heartbeat(ctx, redis, client.userID)
		case <-ctx.Done():
			return
		}
	}
}

func broadcastMessage(ctx context.Context, redisClient *redis.Client, message ChatMessage) error {
	message.Metadata = map[string]interface{}{
		"timestamp": time.Now().Unix(),
	}
	
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	if err := redisClient.Publish(ctx, message.RoomId, payload).Err(); err != nil {
		return err
	}
	
	historyKey := fmt.Sprintf("room:%s:history", message.RoomId)
	return redisClient.ZAdd(ctx, historyKey, redis.Z{
		Score:  float64(message.Timestamp.Unix()),
		Member: payload,
	}).Err()
}

func (s *Service) monitorConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		now := time.Now().Unix()
		
		iter := s.redis.Scan(ctx, 0, "client:*", 1000).Iterator()
		for iter.Next(ctx) {
			clientKey := iter.Val()
			clientData, err := s.redis.HGetAll(ctx, clientKey).Result()
			if err != nil {
				continue
			}
			
			lastSeen, _ := strconv.ParseInt(clientData["lastSeen"], 10, 64)
			if now - lastSeen > 120 { 
				userID := strings.TrimPrefix(clientKey, "client:")
				roomID := clientData["roomID"]
				
				s.redis.Del(ctx, clientKey)
				s.redis.SRem(ctx, fmt.Sprintf("room:%s:members", roomID), userID)
				s.redis.SRem(ctx, "users:online", userID)
				
				broadcastMessage(ctx, s.redis, ChatMessage{
					Type:      SystemMessage,
					Content:   fmt.Sprintf("%s has disconnected (timeout)", clientData["nickname"]),
					RoomId:    roomID,
					Timestamp: time.Now(),
				})
			}
		}
	}
}