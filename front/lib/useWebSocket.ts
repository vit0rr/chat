import { AxiosError } from 'axios';
import { useEffect, useRef, useState } from 'react';

const WS_URL = process.env.BACKEND_WS_ROOT_URL;

export type Message = {
    type: 'text' | 'system';
    content: string;
    room_id: string;
    sender_id: string;
    nickname: string;
    timestamp: string;
}

export function useWebSocket(roomId: string, userId: string, nickname: string, token: string) {
    const [messages, setMessages] = useState<Message[]>([]);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isLoadingHistory, setIsLoadingHistory] = useState(true);
    const wsRef = useRef<WebSocket | null>(null);
    const [page, setPage] = useState(1);
    const [hasMore, setHasMore] = useState(true);
    const [isPageLoaded, setIsPageLoaded] = useState(false);

    // First effect to check if page is loaded
    useEffect(() => {
        if (document.readyState === 'complete') {
            setIsPageLoaded(true);
        } else {
            window.addEventListener('load', () => setIsPageLoaded(true));
            return () => window.removeEventListener('load', () => setIsPageLoaded(true));
        }
    }, []);

    // Load message history
    useEffect(() => {
        const loadMessageHistory = async () => {
            try {
                setIsLoadingHistory(true);
                const historicMessagesResponse = await fetch(`/api/messages?roomId=${roomId}&page=${page}&limit=50`, {
                    headers: {
                        Authorization: `Bearer ${token}`,
                    },
                });

                const historicMessages = await historicMessagesResponse.json();

                // Handle case where historicMessages is null or empty
                if (!historicMessages || historicMessages.length === 0) {
                    setHasMore(false);
                    if (page === 1) {
                        setMessages([]); // Reset messages if it's the first page
                    }
                    return;
                }

                setMessages(prev => {
                    // Filter out duplicates and sort by timestamp
                    const allMessages = [...prev, ...historicMessages];
                    const uniqueMessages = Array.from(
                        new Map(allMessages.map(m => [m.timestamp, m])).values()
                    );
                    return uniqueMessages.sort(
                        (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
                    );
                });
                setHasMore(historicMessages.length === 50); // Assuming 50 is the limit
            } catch (err) {
                console.error('Error loading message history:', err);
                // Don't set error if it's just empty messages
                if (err instanceof AxiosError && err.response?.status !== 404) {
                    setError('Failed to load message history');
                }
                setHasMore(false);
            } finally {
                setIsLoadingHistory(false);
            }
        };

        if (token) {
            loadMessageHistory();
        }
    }, [roomId, token, page]);

    // Second effect to handle WebSocket connection
    useEffect(() => {
        // Only connect if the page is fully loaded and we have all required params
        if (!isPageLoaded || !roomId || !userId || !nickname) {
            return;
        }

        const wsUrl = `${WS_URL}/api/v1/ws?room_id=${roomId}&user_id=${userId}&nickname=${encodeURIComponent(nickname)}&token=${token}`;
        console.log('Connecting to WebSocket:', wsUrl);

        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('WebSocket connected');
            setIsConnected(true);
            setError(null);
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                setMessages((prev) => [...prev, message]);
            } catch (err) {
                console.error('Error parsing message:', err);
            }
        };

        ws.onerror = (event) => {
            console.error('WebSocket error:', event);
            setError('Failed to connect to chat');
            setIsConnected(false);
        };

        ws.onclose = (event) => {
            console.log('WebSocket closed:', event);
            setError('Disconnected from chat');
            setIsConnected(false);
        };

        wsRef.current = ws;

        return () => {
            if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
                ws.close();
            }
        };
    }, [isPageLoaded, roomId, userId, nickname, token]);

    const sendMessage = (content: string) => {
        if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
            setError('Not connected to chat');
            return;
        }

        const message = {
            type: 'text',
            content,
            room_id: roomId,
        };

        wsRef.current.send(JSON.stringify(message));
    };

    const loadMoreMessages = () => {
        if (!isLoadingHistory && hasMore) {
            setPage(prev => prev + 1);
        }
    };

    return {
        messages,
        sendMessage,
        isConnected,
        error,
        isLoadingHistory,
        hasMore,
        loadMoreMessages
    };
} 