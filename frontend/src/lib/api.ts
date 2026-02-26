import { get } from 'svelte/store';
import { authStore } from './stores';
import type { AuthResponse, Form, FormField, Playbook, Run, Server, User, Vault } from './types';

export class ApiError extends Error {
	constructor(public status: number, message: string) {
		super(message);
	}
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const { token } = get(authStore);
	const headers: Record<string, string> = {};

	if (!(options.body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}
	if (options.headers) {
		Object.assign(headers, options.headers);
	}

	const res = await fetch(`/api${path}`, { ...options, headers });

	if (res.status === 401) {
		authStore.logout();
		throw new ApiError(401, 'Session expired');
	}
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: 'Request failed' }));
		throw new ApiError(res.status, body.error || 'Request failed');
	}
	if (res.status === 204) return null as T;
	return res.json();
}

// Like request<T> but also returns the X-Total-Count response header.
async function requestPaged<T>(path: string, options: RequestInit = {}): Promise<{ data: T; total: number }> {
	const { token } = get(authStore);
	const headers: Record<string, string> = {};

	if (!(options.body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}
	if (options.headers) {
		Object.assign(headers, options.headers);
	}

	const res = await fetch(`/api${path}`, { ...options, headers });

	if (res.status === 401) {
		authStore.logout();
		throw new ApiError(401, 'Session expired');
	}
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: 'Request failed' }));
		throw new ApiError(res.status, body.error || 'Request failed');
	}

	const total = parseInt(res.headers.get('X-Total-Count') ?? '0', 10);
	const data = (await res.json()) as T;
	return { data, total };
}

export const auth = {
	login: (username: string, password: string) =>
		request<AuthResponse>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
	logout: () => request('/auth/logout', { method: 'POST' }),
};

export const users = {
	list: () => request<User[]>('/users'),
	create: (data: { username: string; password: string; role: string }) =>
		request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { username: string; password?: string; role: string }) =>
		request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/users/${id}`, { method: 'DELETE' }),
};

export const servers = {
	list: () => request<Server[]>('/servers'),
	get: (id: string) => request<Server>(`/servers/${id}`),
	create: (data: Partial<Server>) =>
		request<Server>('/servers', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: Partial<Server>) =>
		request<Server>(`/servers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/servers/${id}`, { method: 'DELETE' }),
	test: (id: string) =>
		request<{ success: boolean; message: string }>(`/servers/${id}/test`, { method: 'POST' }),
};

export const playbooks = {
	list: () => request<Playbook[]>('/playbooks'),
	get: (id: string) => request<Playbook>(`/playbooks/${id}`),
	upload: (name: string, description: string, file: File) => {
		const fd = new FormData();
		fd.append('name', name);
		fd.append('description', description);
		fd.append('file', file);
		return request<Playbook>('/playbooks', { method: 'POST', body: fd });
	},
	delete: (id: string) => request<void>(`/playbooks/${id}`, { method: 'DELETE' }),
};

export const forms = {
	list: () => request<Form[]>('/forms'),
	get: (id: string) => request<Form>(`/forms/${id}`),
	getFields: (id: string) => request<FormField[]>(`/forms/${id}/fields`),
	quickActions: () => request<Form[]>('/quick-actions'),
	create: (data: Partial<Form> & { fields?: Partial<FormField>[] }) =>
		request<Form>('/forms', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: Partial<Form> & { fields?: Partial<FormField>[] }) =>
		request<Form>(`/forms/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/forms/${id}`, { method: 'DELETE' }),
	uploadImage: (id: string, file: File) => {
		const fd = new FormData();
		fd.append('file', file);
		return request<Form>(`/forms/${id}/image`, { method: 'POST', body: fd });
	},
	deleteImage: (id: string) => request<Form>(`/forms/${id}/image`, { method: 'DELETE' }),
};

export const vaults = {
	list: () => request<Vault[]>('/vaults'),
	get: (id: string) => request<Vault>(`/vaults/${id}`),
	create: (data: { name: string; description: string; password: string }) =>
		request<Vault>('/vaults', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { name: string; description: string; password?: string }) =>
		request<Vault>(`/vaults/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/vaults/${id}`, { method: 'DELETE' }),
	uploadFile: (id: string, file: File) => {
		const fd = new FormData();
		fd.append('file', file);
		return request<Vault>(`/vaults/${id}/upload`, { method: 'POST', body: fd });
	},
	deleteFile: (id: string) => request<Vault>(`/vaults/${id}/file`, { method: 'DELETE' }),
};

export const runs = {
	/** Returns a page of runs plus the total count across all pages. */
	list: (params?: { limit?: number; offset?: number }) => {
		const qs = params
			? '?' +
				new URLSearchParams(
					Object.fromEntries(
						Object.entries(params)
							.filter(([, v]) => v !== undefined)
							.map(([k, v]) => [k, String(v)])
					)
				).toString()
			: '';
		return requestPaged<Run[]>(`/runs${qs}`);
	},
	get: (id: string) => request<Run>(`/runs/${id}`),
	create: (formId: string, variables: Record<string, unknown>) =>
		request<{ run_id: string; status: string }>('/runs', {
			method: 'POST',
			body: JSON.stringify({ form_id: formId, variables }),
		}),
};
