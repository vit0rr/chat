import axios from 'axios';

const API_URL = process.env.BACKEND_ROOT_URL;

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
            `${API_URL}/api/v1/rooms/${roomId}/messages`,
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

    const response = await fetch(`${API_URL}/api/v1/rooms/${roomId}/register-user`, {
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