import axios from 'axios';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'https://chat-solitary-butterfly-9161.fly.dev/api/v1';

export type Room = {
    room_id: string;
    users: {
        id: string;
        nickname: string;
    }[];
    locked_by?: string;
    created_at: string;
    updated_at: string;
}

export type RoomsList = {
    rooms: Room[];
}

export type RegisterUserResponse = {
    user_id: string;
    room_id: string;
    nickname: string;
}

export type Message = {
    type: 'text' | 'system';
    content: string;
    room_id: string;
    sender_id: string;
    nickname: string;
    timestamp: string;
}

export const getRooms = async (token: string): Promise<Room[]> => {
    const response = await axios.get<RoomsList>(`${API_URL}/rooms`, {
        headers: {
            Authorization: `Bearer ${token}`,
        },
    });
    return response.data.rooms;
};

export const registerUserInRoom = async (
    id: string,
    users: {
        id: string;
        nickname: string;
    }[],
    token: string
): Promise<RegisterUserResponse> => {
    if (!token) {
        throw new Error('Authentication token is required');
    }

    if (!id) {
        throw new Error('Room ID is required');
    }

    try {
        const response = await axios.post<RoomDetails>(
            `${API_URL}/rooms/${id}/register-user`,
            {
                user_id: users[0].id,
                nickname: users[0].nickname,
            },
            {
                headers: {
                    Authorization: `Bearer ${token}`,
                    'Content-Type': 'application/json',
                },
            }
        );

        return {
            user_id: users[0].id,
            room_id: response.data.id,
            nickname: users[0].nickname
        };
    } catch (error: unknown) {
        console.error('Register user in room error:', error instanceof Error ? error.message : String(error));
        throw error;
    }
};

// Add this type to match backend response
type RoomDetails = {
    id: string;
    users: Array<{
        id: string;
        nickname: string;
    }>;
    locked_by?: string;
    created_at: string;
    updated_at: string;
};

export const getRoom = async (roomId: string, token: string): Promise<Room> => {
    const response = await axios.get(`${API_URL}/rooms/${roomId}`, {
        headers: {
            Authorization: `Bearer ${token}`,
        },
    });
    return response.data;
};

export const getMessages = async (
    roomId: string,
    token: string,
    page: number = 1,
    limit: number = 50
): Promise<Message[]> => {
    if (!token) {
        throw new Error('Authentication token is required');
    }

    try {
        const response = await axios.get<Message[]>(
            `${API_URL}/rooms/${roomId}/messages`,
            {
                params: {
                    page,
                    limit,
                },
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            }
        );

        return response.data;
    } catch (error: unknown) {
        console.error('Error fetching messages:', error instanceof Error ? error.message : String(error));
        throw error;
    }
};

export async function joinRoom(roomId: string, userId: string, nickname: string, token: string) {
    if (!token) {
        throw new Error('Authentication required');
    }

    const response = await fetch(`${API_URL}/rooms/${roomId}/register-user`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
            user_id: userId,
            nickname: nickname,
        }),
    });

    if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to join room');
    }

    const data = await response.json();
    return data;
} 