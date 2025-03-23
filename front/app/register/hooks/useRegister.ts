import { useAuth } from "@/lib/auth-context";
import { useRouter } from "next/navigation";
import { useState } from "react";


type RegisterRequest = {
    email: string;
    password: string;
    nickname: string;
};

export const useRegister = () => {
    const [formData, setFormData] = useState<RegisterRequest>({
        email: "",
        password: "",
        nickname: "",
    });
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const router = useRouter();
    const { login: authLogin } = useAuth();

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setFormData({
            ...formData,
            [e.target.name]: e.target.value,
        });
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");
        setLoading(true);

        try {
            const response = await fetch(`/api/auth/register`, {
                method: "POST",
                body: JSON.stringify(formData),
            });
            const data = await response.json();

            if (data.error) {
                setError(data.error);
                setLoading(false);
                return;
            }

            const { token, user_id, nickname } = data;
            authLogin(token, user_id, nickname);
            router.push("/rooms");
        } catch (err: unknown) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError("An unknown error occurred");
            }
        } finally {
            setLoading(false);
        }
    };

    return {
        error,
        handleSubmit,
        formData,
        handleChange,
        loading
    }
}