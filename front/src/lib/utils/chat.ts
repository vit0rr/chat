export async function fetchMessages(roomId: string, page: number, token: string) {
    const response = await fetch(`/api/messages?room_id=${roomId}&page=${page}`, {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    });
    if (!response.ok) {
        throw new Error('Failed to fetch messages');
    }
    return response.json();
} 