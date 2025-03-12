import axios from 'axios';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'https://chat-solitary-butterfly-9161.fly.dev/api/v1';

export type RegisterRequest = {
    email: string;
    password: string;
    nickname: string;
}

export type LoginRequest = {
    email: string;
    password: string;
}

export type AuthResponse = {
    token: string;
    user_id: string;
    nickname: string;
}

export const register = async (data: RegisterRequest): Promise<AuthResponse> => {
    const response = await axios.post(`${API_URL}/auth/register`, data);
    return response.data;
};

export const login = async (data: LoginRequest): Promise<AuthResponse> => {
    const response = await axios.post(`${API_URL}/auth/login`, data);
    return response.data;
};

export const deleteUser = async (userId: string, token: string): Promise<void> => {
    await axios.delete(`${API_URL}/auth/user`, {
        data: { user_id: userId },
        headers: {
            Authorization: `Bearer ${token}`,
        },
    });
};