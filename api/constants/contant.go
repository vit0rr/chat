package constants

type ErrorMessage struct {
	Message string `json:"message" validate:"required"`
	ID      string `json:"id" validate:"required"`
	Code    int    `json:"code" validate:"required,min=100,max=599"`
}

const (
	// HeaderRequestID - request ID header
	HeaderRequestID = "X-Request-Id"

	RoomsCollection    = "rooms"
	MessagesCollection = "messages"
	UsersCollection    = "users"
	ClientsCollection  = "clients"
	// @TODO: it will change in production, probably move to env
	DatabaseName = "db_chat"
)

const (
	// Room errors
	RoomNotFound               = "Room not found"
	FailedToGetRooms           = "Failed to get rooms"
	RoomIDRequired             = "Room ID is required"
	FailedToGetMessages        = "Failed to get messages"
	FailedToCheckExistingRoom  = "Failed to check existing room"
	FailedToCreateOrUpdateRoom = "Failed to create or update room"

	// User errors
	FailedToGetUsers            = "Failed to get users"
	UserNotFound                = "User not found"
	FailedToCreateUser          = "Failed to create user"
	UserIDRequired              = "User ID is required"
	UserNotAuthorizedToLockRoom = "User not authorized to lock room"

	// General errors
	FailedToDecodeBody = "Failed to decode body"
)

var ErrorMessages = map[string]ErrorMessage{
	// Room errors
	RoomNotFound: {
		Message: RoomNotFound,
		ID:      "room_not_found",
		Code:    404,
	},
	FailedToGetRooms: {
		Message: FailedToGetRooms,
		ID:      "failed_get_rooms",
		Code:    500,
	},
	RoomIDRequired: {
		Message: RoomIDRequired,
		ID:      "room_id_required",
		Code:    400,
	},
	FailedToGetMessages: {
		Message: FailedToGetMessages,
		ID:      "failed_get_messages",
		Code:    500,
	},
	FailedToCheckExistingRoom: {
		Message: FailedToCheckExistingRoom,
		ID:      "failed_check_existing_room",
		Code:    500,
	},
	FailedToCreateOrUpdateRoom: {
		Message: FailedToCreateOrUpdateRoom,
		ID:      "failed_create_or_update_room",
		Code:    500,
	},

	// User errors
	FailedToGetUsers: {
		Message: FailedToGetUsers,
		ID:      "failed_get_users",
		Code:    500,
	},
	UserNotFound: {
		Message: UserNotFound,
		ID:      "user_not_found",
		Code:    404,
	},
	FailedToCreateUser: {
		Message: FailedToCreateUser,
		ID:      "failed_create_user",
		Code:    500,
	},
	UserIDRequired: {
		Message: UserIDRequired,
		ID:      "user_id_required",
		Code:    400,
	},
	UserNotAuthorizedToLockRoom: {
		Message: UserNotAuthorizedToLockRoom,
		ID:      "user_not_authorized_to_lock_room",
		Code:    403,
	},

	// General errors
	FailedToDecodeBody: {
		Message: FailedToDecodeBody,
		ID:      "failed_decode_body",
		Code:    400,
	},
}
