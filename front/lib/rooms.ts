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