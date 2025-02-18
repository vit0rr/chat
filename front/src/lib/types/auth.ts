export type LoginRequest = {
    email: string;
    password: string;
}

export type LoginResponse = {
    token: string;
    user: {
        id: string;
        email: string;
        nickname: string;
    };
}

export type AuthStore = {
    token: string | null;
    user: {
        id: string;
        email: string;
        nickname: string;
    } | null;
} 