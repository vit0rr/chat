import { User } from "@/lib/auth-context";
import { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useEffect, useState } from "react";

type CreateRoomParams = {
    user: User | null;
    token: string | null;
    router: AppRouterInstance
    isAuthenticated: boolean;
}

export const useCreate = ({ user, token, router, isAuthenticated }: CreateRoomParams) => {
    const [nickname, setNickname] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (user?.nickname) {
            setNickname(user.nickname);
        }
    }, [user]);

    useEffect(() => {
        if (!isAuthenticated) {
            router.push("/login");
        }
    }, [isAuthenticated, router]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!nickname.trim()) {
            setError("Nickname is required");
            return;
        }

        if (!user?.id || !token) {
            setError("Please log in again");
            router.push("/login");
            return;
        }

        try {
            setLoading(true);
            setError("");

            const newRoomId = crypto.randomUUID();

            const response = await fetch(`/api/rooms`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({
                    room_id: newRoomId,
                    user_id: user.id,
                    nickname: nickname,
                }),
            });

            const data = await response.json();

            if (data && data.room_id) {
                router.push(`/rooms/${data.room_id}`);
            } else {
                throw new Error("Invalid response from server");
            }
        } catch (err: unknown) {
            console.error("Error creating room:", err);
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError("Failed to create room. Please try again.");
            }
        } finally {
            setLoading(false);
        }
    };

    return {
        error,
        handleSubmit,
        nickname,
        setNickname,
        loading,
    }
}