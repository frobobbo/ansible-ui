<script lang="ts">
	import { onMount } from 'svelte';
	import { servers as serversApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast, confirmDialog } from '$lib/toast';
	import type { Server } from '$lib/types';

	let list = $state<Server[]>([]);
	let loading = $state(true);
	let error = $state('');
	let filter = $state('');
	let testResults = $state<Record<string, { success: boolean; message: string }>>({});
	let testing = $state<Record<string, boolean>>({});

	let filtered = $derived(
		filter.trim()
			? list.filter(
					(s) =>
						s.name.toLowerCase().includes(filter.toLowerCase()) ||
						s.host.toLowerCase().includes(filter.toLowerCase()) ||
						s.username.toLowerCase().includes(filter.toLowerCase()) ||
						s.execution_environment.toLowerCase().includes(filter.toLowerCase())
				)
			: list
	);

	// Modal state
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ name: '', host: '', port: 22, username: '', ssh_private_key: '', pre_command: '', execution_environment: '' });
	let saving = $state(false);
	let formError = $state('');

	let isEE = $derived(form.execution_environment.trim() !== '');

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await serversApi.list(); }
		catch { error = 'Failed to load servers'; }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = { name: '', host: '', port: 22, username: '', ssh_private_key: '', pre_command: '', execution_environment: '' };
		formError = '';
		showModal = true;
	}

	function openEdit(sv: Server) {
		editingId = sv.id;
		form = {
			name: sv.name,
			host: sv.host,
			port: sv.port,
			username: sv.username,
			ssh_private_key: '',
			pre_command: sv.pre_command,
			execution_environment: sv.execution_environment ?? ''
		};
		formError = '';
		showModal = true;
	}

	async function save() {
		saving = true;
		formError = '';
		try {
			if (editingId) {
				await serversApi.update(editingId, form);
			} else {
				await serversApi.create(form);
			}
			showModal = false;
			toast.success(editingId ? 'Server updated' : 'Server added');
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function remove(id: string) {
		if (!(await confirmDialog('Delete this server?'))) return;
		try {
			await serversApi.delete(id);
			await load();
			toast.success('Server deleted');
		} catch {
			toast.error('Delete failed');
		}
	}

	async function testConnection(id: string) {
		testing[id] = true;
		try {
			const result = await serversApi.test(id);
			testResults[id] = result;
		} catch {
			testResults[id] = { success: false, message: 'Test failed' };
		} finally {
			testing[id] = false;
		}
	}
</script>

<div class="page-header">
	<h1>Servers</h1>
	<div class="header-right">
		<input class="form-control search" placeholder="Search servers..." bind:value={filter} />
		{#if $isAdmin}
			<button class="btn btn-primary" onclick={openCreate}>+ Add Server</button>
		{/if}
	</div>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No servers configured. {#if $isAdmin}Add one to get started.{/if}</div>
{:else if filtered.length === 0}
	<div class="empty-state">No servers match "{filter}".</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Type</th><th>Target</th><th>Actions</th></tr></thead>
			<tbody>
				{#each filtered as sv}
					<tr>
						<td><strong>{sv.name}</strong></td>
						<td>
							{#if sv.execution_environment}
								<span class="badge badge-ee">EE</span>
							{:else}
								<span class="badge badge-ssh">SSH</span>
							{/if}
						</td>
						<td class="target-cell">
							{#if sv.execution_environment}
								<span class="ee-image" title={sv.execution_environment}>{sv.execution_environment}</span>
							{:else}
								{sv.username}@{sv.host}:{sv.port}
							{/if}
						</td>
						<td>
							<div class="actions">
								<button class="btn btn-sm btn-secondary" onclick={() => testConnection(sv.id)} disabled={testing[sv.id]}>
									{testing[sv.id] ? 'Testing...' : 'Test'}
								</button>
								{#if $isAdmin}
									<button class="btn btn-sm btn-secondary" onclick={() => openEdit(sv)}>Edit</button>
									<button class="btn btn-sm btn-danger" onclick={() => remove(sv.id)}>Delete</button>
								{/if}
							</div>
							{#if testResults[sv.id]}
								<div class="test-result" class:ok={testResults[sv.id].success}>
									{testResults[sv.id].success ? '✓' : '✗'} {testResults[sv.id].message}
								</div>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

{#if showModal}
	<div class="modal-overlay" onclick={() => showModal = false} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<h2>{editingId ? 'Edit Server' : 'Add Server'}</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }} autocomplete="off">
				<div class="form-group">
					<label>Name</label>
					<input class="form-control" bind:value={form.name} required />
				</div>

				<div class="form-group">
					<label>Execution Environment Image <span class="hint-inline">(optional — leave blank to use SSH)</span></label>
					<input class="form-control" bind:value={form.execution_environment}
						placeholder="ghcr.io/ansible/community-general-ee:latest" />
					<small class="hint">A container image (GitHub Container Registry, Docker Hub, etc.) with ansible-playbook installed. When set, the playbook runs inside a Kubernetes Job using this image instead of connecting via SSH.</small>
				</div>

				{#if !isEE}
					<div class="ssh-section">
						<div class="grid-2">
							<div class="form-group">
								<label>Host / IP</label>
								<input class="form-control" bind:value={form.host} required={!isEE} />
							</div>
							<div class="form-group">
								<label>Port</label>
								<input class="form-control" type="number" bind:value={form.port} min="1" max="65535" />
							</div>
							<div class="form-group">
								<label>SSH Username</label>
								<input class="form-control" bind:value={form.username} required={!isEE} />
							</div>
						</div>
						<div class="form-group">
							<label>SSH Private Key (PEM){editingId ? ' — leave blank to keep existing' : ''}</label>
							<textarea class="form-control" bind:value={form.ssh_private_key} rows="8"
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
								required={!editingId && !isEE}></textarea>
						</div>
					</div>
				{/if}

				<div class="form-group">
					<label>Pre-run Command <span class="hint-inline">(optional)</span></label>
					<input class="form-control" bind:value={form.pre_command}
						placeholder={isEE ? 'e.g. pip install -r requirements.txt' : 'e.g. . /home/user/ansible/bin/activate'} />
					<small class="hint">
						{#if isEE}
							Runs inside the container before ansible-playbook (e.g. install extra collections).
						{:else}
							Runs on the remote host before ansible-playbook (e.g. activate a virtualenv).
						{/if}
					</small>
				</div>

				<div class="actions" style="justify-content:flex-end">
					<button type="button" class="btn btn-secondary" onclick={() => showModal = false}>Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.header-right { display: flex; gap: 0.75rem; align-items: center; }
	.search { width: 220px; }
	.test-result { font-size: 0.75rem; margin-top: 0.25rem; }
	.test-result.ok { color: var(--success); }
	.test-result:not(.ok) { color: var(--danger); }
	.target-cell { font-size: 0.85rem; max-width: 280px; }
	.ee-image { font-family: monospace; font-size: 0.8rem; word-break: break-all; }
	.badge { display: inline-block; padding: 0.15rem 0.5rem; border-radius: 9999px; font-size: 0.7rem; font-weight: 600; letter-spacing: 0.05em; }
	.badge-ssh { background: var(--border); color: var(--text-muted); }
	.badge-ee  { background: #dbeafe; color: #1d4ed8; }
	.hint-inline { font-weight: normal; font-size: 0.8rem; color: var(--text-muted); }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 600px; max-height: 90vh; overflow-y: auto; }
</style>
