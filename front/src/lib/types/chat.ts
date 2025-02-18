export type RegisterUserRequest = {
    user_id: string;
    room_id: string;
    nickname: string;
};

export type User = {
    userId: string;
    nickname: string;
    joinedAt: string;
};

export type Room = {
    id: string;
    users: User[];
    lockedBy?: string;
    createdAt: string;
    updatedAt: string;
};