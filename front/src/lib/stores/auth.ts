import type { AuthStore } from '$lib/types/auth';
import { writable } from 'svelte/store';

const createAuthStore = () => {
    // Initialize from localStorage if available
    const storedAuth = typeof localStorage !== 'undefined'
        ? localStorage.getItem('auth')
        : null;
    const initial: AuthStore = storedAuth ? JSON.parse(storedAuth) : { token: null, user: null };

    const { subscribe, set, update } = writable<AuthStore>(initial);

    return {
        subscribe,
        login: (token: string, user: AuthStore['user']) => {
            const auth = { token, user };
            localStorage.setItem('auth', JSON.stringify(auth));
            set(auth);
        },
        logout: () => {
            localStorage.removeItem('auth');
            set({ token: null, user: null });
        }
    };
};

export const auth = createAuthStore(); 