import { useAuth } from "@/lib/auth-context";
import { Room } from "@/lib/rooms";
import { AxiosError } from "axios";
import { useRouter } from "next/navigation";
import { use, useEffect, useState } from "react";

type RoomIdParams = Promise<{ roomId: string }>

export const useRoomId = (params: RoomIdParams) => {
    const resolvedParams = use(params);
    const roomId = resolvedParams.roomId;

    const { user, token, isAuthenticated, logout } = useAuth();
    const [room, setRoom] = useState<Room | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [isJoining, setIsJoining] = useState(false);
    const router = useRouter();

    useEffect(() => {
        if (!isAuthenticated) {
            router.push("/login");
            return;
        }

        const fetchRoom = async () => {
            if (!token || !roomId) return;

            try {
                setLoading(true);
                setError("");
                const response = await fetch(`/api/room?roomId=${roomId}`, {
                    method: "GET",
                    headers: {
                        Authorization: `Bearer ${token}`,
                    },
                });

                if (response.status === 401) {
                    logout();
                    return;
                }
                const roomData = await response.json();
                setRoom(roomData);
            } catch (err: unknown) {
                console.error("Error fetching room:", err);
                if (err instanceof Error) {
                    setError(err.message);
                } else {
                    setError("Failed to load room");
                }
                if (err instanceof AxiosError && err.response?.status === 404) {
                    router.push("/rooms");
                }
            } finally {
                setLoading(false);
            }
        };

        fetchRoom();
    }, [isAuthenticated, token, roomId, router, logout]);

    const isUserInRoom = room?.users?.some(
        (roomUser) => roomUser.id === user?.id
    );

    const handleJoinRoom = async () => {
        if (!user || !token) {
            router.push("/login");
            return;
        }

        try {
            setIsJoining(true);
            setError("");
            await fetch(
                `/api/room?roomId=${roomId}&userId=${user.id}&nickname=${user.nickname}`,
                {
                    method: "POST",
                    headers: {
                        Authorization: `Bearer ${token}`,
                    },
                }
            );
            // Fetch the updated room data
            const response = await fetch(`/api/room?roomId=${roomId}`, {
                method: "GET",
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            });

            if (response.status === 401) {
                logout();
                return;
            }

            const updatedRoom = await response.json();
            setRoom(updatedRoom);
        } catch (err) {
            console.error("Error joining room:", err);
            setError(err instanceof Error ? err.message : "Failed to join room");
        } finally {
            setIsJoining(false);
        }
    };

    return {
        isAuthenticated,
        loading,
        error,
        room,
        roomId,
        isUserInRoom,
        handleJoinRoom,
        isJoining,
        user,
        token,
    }

}