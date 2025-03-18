import { useAuth } from "@/lib/auth-context";
import { Room } from "@/lib/rooms";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export const useRooms = () => {
    const { token, logout } = useAuth();
    const [rooms, setRooms] = useState<Room[]>([]);
    const [loading, setLoading] = useState(true);
    const router = useRouter();

    useEffect(() => {
        const fetchRooms = async () => {
            if (!token) return;

            try {
                setLoading(true);
                const roomsResponse = await fetch(`/api/rooms`, {
                    headers: {
                        Authorization: `Bearer ${token}`,
                    },
                });

                if (roomsResponse.status === 401) {
                    logout();
                    return;
                }

                const roomsData = await roomsResponse.json();
                const newToken = roomsResponse.headers.get("X-New-Token");
                if (newToken) {
                    localStorage.setItem("token", newToken);
                }
                setRooms(roomsData);
            } catch (error) {
                console.error("Error fetching rooms:", error);
            } finally {
                setLoading(false);
            }
        };

        fetchRooms();
    }, [token, logout]);

    return {
        rooms,
        loading,
        router,
    }
}