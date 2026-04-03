import { get } from 'svelte/store';
import { authStore } from './stores';
import type { AuditLog, AppSettings, AuthResponse, EEFiles, EmailSettings, GitHubSettings, Form, FormField, Host, Playbook, Run, Server, ServerGroup, SSHCert, User, Vault, VarSuggestion } from './types';

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
	forgotPassword: (username: string) =>
		request<{ message: string }>('/auth/forgot-password', { method: 'POST', body: JSON.stringify({ username }) }),
	resetPassword: (token: string, password: string) =>
		request<{ message: string }>('/auth/reset-password', { method: 'POST', body: JSON.stringify({ token, password }) }),
};

export const users = {
	list: () => request<User[]>('/users'),
	create: (data: { username: string; password: string; role: string; email?: string }) =>
		request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { username: string; password?: string; role: string; email?: string }) =>
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
	create: (data: { name: string; description: string; repo_url: string; branch: string; token?: string }) =>
		request<Playbook>('/playbooks', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { name: string; description: string; repo_url: string; branch: string; token?: string }) =>
		request<Playbook>(`/playbooks/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/playbooks/${id}`, { method: 'DELETE' }),
	listFiles: (id: string) => request<string[]>(`/playbooks/${id}/files`),
	scanVars: (id: string, path: string) =>
		request<VarSuggestion[]>(`/playbooks/${id}/scan?path=${encodeURIComponent(path)}`),
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
	regenerateWebhookToken: (id: string) => request<Form>(`/forms/${id}/webhook-token`, { method: 'POST' }),
	revokeWebhookToken: (id: string) => request<Form>(`/forms/${id}/webhook-token`, { method: 'DELETE' }),
};

export const serverGroups = {
	list: () => request<ServerGroup[]>('/server-groups'),
	get: (id: string) => request<ServerGroup>(`/server-groups/${id}`),
	create: (data: { name: string; description: string }) =>
		request<ServerGroup>('/server-groups', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { name: string; description: string }) =>
		request<ServerGroup>(`/server-groups/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/server-groups/${id}`, { method: 'DELETE' }),
	getMembers: (id: string) => request<Server[]>(`/server-groups/${id}/members`),
	setMembers: (id: string, serverIds: string[]) =>
		request<void>(`/server-groups/${id}/members`, { method: 'PUT', body: JSON.stringify({ server_ids: serverIds }) }),
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

export const hosts = {
	list: () => request<Host[]>('/hosts'),
	get: (id: string) => request<Host>(`/hosts/${id}`),
	create: (data: { name: string; address: string; description: string; ssh_cert_id?: string | null; vars: Record<string, string> }) =>
		request<Host>('/hosts', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { name: string; address: string; description: string; ssh_cert_id?: string | null; vars: Record<string, string> }) =>
		request<Host>(`/hosts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/hosts/${id}`, { method: 'DELETE' }),
	importFile: (file: File) => {
		const fd = new FormData();
		fd.append('file', file);
		return request<{ created: string[]; skipped: string[]; errors: string[] }>('/hosts/import', { method: 'POST', body: fd });
	},
};

export const sshCerts = {
	list: () => request<SSHCert[]>('/ssh-certs'),
	get: (id: string) => request<SSHCert>(`/ssh-certs/${id}`),
	create: (data: { name: string; description: string }) =>
		request<SSHCert>('/ssh-certs', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { name: string; description: string }) =>
		request<SSHCert>(`/ssh-certs/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	delete: (id: string) => request<void>(`/ssh-certs/${id}`, { method: 'DELETE' }),
	uploadFile: (id: string, file: File) => {
		const fd = new FormData();
		fd.append('file', file);
		return request<SSHCert>(`/ssh-certs/${id}/upload`, { method: 'POST', body: fd });
	},
	deleteFile: (id: string) => request<SSHCert>(`/ssh-certs/${id}/file`, { method: 'DELETE' }),
};

export const settings = {
	getApp: () => request<AppSettings>('/settings/app'),
	updateApp: (data: AppSettings) =>
		request<AppSettings>('/settings/app', { method: 'PUT', body: JSON.stringify(data) }),
	getEmail: () => request<EmailSettings>('/settings/email'),
	updateEmail: (data: EmailSettings) =>
		request<EmailSettings>('/settings/email', { method: 'PUT', body: JSON.stringify(data) }),
	testEmail: (to: string, config?: Partial<EmailSettings>) =>
		request<{ message: string }>('/settings/email/test', {
			method: 'POST',
			body: JSON.stringify({ to, config }),
		}),
	getGitHub: () => request<GitHubSettings>('/settings/github'),
	updateGitHub: (data: GitHubSettings) =>
		request<GitHubSettings>('/settings/github', { method: 'PUT', body: JSON.stringify(data) }),
};

export const ee = {
	get: () => request<EEFiles>('/ee'),
	update: (data: { message: string; files: EEFiles }) =>
		request<{ status: string }>('/ee', { method: 'PUT', body: JSON.stringify(data) }),
};

export const audit = {
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
		return requestPaged<AuditLog[]>(`/audit${qs}`);
	},
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
	cancel: (id: string) => request<void>(`/runs/${id}/cancel`, { method: 'POST' }),
};
