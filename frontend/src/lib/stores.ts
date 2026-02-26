import { writable, derived } from 'svelte/store';
import type { User } from './types';

function createAuthStore() {
	const stored = typeof localStorage !== 'undefined'
		? { token: localStorage.getItem('token'), user: JSON.parse(localStorage.getItem('user') || 'null') }
		: { token: null, user: null };

	const { subscribe, set } = writable<{ token: string | null; user: User | null }>(stored);

	return {
		subscribe,
		login(token: string, user: User) {
			localStorage.setItem('token', token);
			localStorage.setItem('user', JSON.stringify(user));
			set({ token, user });
		},
		logout() {
			localStorage.removeItem('token');
			localStorage.removeItem('user');
			set({ token: null, user: null });
		}
	};
}

export const authStore = createAuthStore();
export const isAuthenticated = derived(authStore, ($a) => !!$a.token);
export const currentUser = derived(authStore, ($a) => $a.user);
export const isAdmin = derived(authStore, ($a) => $a.user?.role === 'admin');
export const isEditor = derived(authStore, ($a) => $a.user?.role === 'admin' || $a.user?.role === 'editor');
