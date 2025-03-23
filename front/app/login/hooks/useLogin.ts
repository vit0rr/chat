import { useAuth } from "@/lib/auth-context";
import { useRouter } from "next/navigation";
import { useState } from "react";

export const useLogin = () => {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    const router = useRouter();
    const { login } = useAuth();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");
        setLoading(true);

        const response = await fetch(`/api/auth/login`, {
            method: "POST",
            body: JSON.stringify({ email, password }),
        });

        const data = await response.json();

        if (data.error) {
            setError(data.error);
            setLoading(false);
            return;
        }

        const { token, user_id, nickname } = data;
        login(token, user_id, nickname);
        setEmail("");
        setPassword("");
        router.push("/rooms");
    };

    return {
        error,
        handleSubmit,
        email,
        setEmail,
        password,
        setPassword,
        loading,
    }
}