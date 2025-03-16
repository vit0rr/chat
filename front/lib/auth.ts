import axios from 'axios';

const API_URL = process.env.BACKEND_ROOT_URL;

export const deleteUser = async (userId: string, token: string): Promise<void> => {
    await axios.delete(`${API_URL}/api/v1/auth/user`, {
        data: { user_id: userId },
        headers: {
            Authorization: `Bearer ${token}`,
        },
    });
};