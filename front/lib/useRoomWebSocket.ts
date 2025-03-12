import { useEffect, useRef, useState } from 'react';
import { Room } from './rooms';

export function useRoomWebSocket(roomId: string, userId: string, token: string) {
    const [room, setRoom] = useState<Room | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const wsRef = useRef<WebSocket | null>(null);

    useEffect(() => {
        if (!roomId || !userId || !token) {
            return;
        }

        const wsUrl = `wss://chat-solitary-butterfly-9161.fly.dev/api/v1/rooms/${roomId}/watch?user_id=${userId}`;
        console.log('Connecting to Room WebSocket:', wsUrl);

        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('Room WebSocket connected');
            setIsConnected(true);
            setError(null);
        };

        ws.onmessage = (event) => {
            try {
                const updatedRoom = JSON.parse(event.data);
                setRoom(updatedRoom);
            } catch (err) {
                console.error('Error parsing room update:', err);
            }
        };

        ws.onerror = (event) => {
            console.error('Room WebSocket error:', event);
            setError('Failed to connect to room updates');
            setIsConnected(false);
        };

        ws.onclose = () => {
            console.log('Room WebSocket disconnected');
            setIsConnected(false);
        };

        wsRef.current = ws;

        return () => {
            if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
                ws.close();
            }
        };
    }, [roomId, userId, token]);

    return {
        room,
        isConnected,
        error
    };
} 