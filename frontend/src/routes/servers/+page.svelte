<script lang="ts">
	import { onMount } from 'svelte';
	import { servers as serversApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import type { Server } from '$lib/types';

	let list = $state<Server[]>([]);
	let loading = $state(true);
	let error = $state('');
	let testResults = $state<Record<string, { success: boolean; message: string }>>({});
	let testing = $state<Record<string, boolean>>({});

	// Modal state
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ name: '', host: '', port: 22, username: '', ssh_private_key: '', pre_command: '' });
	let saving = $state(false);
	let formError = $state('');

	onMount(async () => {
		await load();
	});

	async function load() {
		loading = true;
		try { list = await serversApi.list(); }
		catch { error = 'Failed to load servers'; }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = { name: '', host: '', port: 22, username: '', ssh_private_key: '', pre_command: '' };
		formError = '';
		showModal = true;
	}

	function openEdit(sv: Server) {
		editingId = sv.id;
		form = { name: sv.name, host: sv.host, port: sv.port, username: sv.username, ssh_private_key: '', pre_command: sv.pre_command };
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
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function remove(id: string) {
		if (!confirm('Delete this server?')) return;
		try { await serversApi.delete(id); await load(); }
		catch { alert('Delete failed'); }
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
	{#if $isAdmin}
		<button class="btn btn-primary" onclick={openCreate}>+ Add Server</button>
	{/if}
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No servers configured. {#if $isAdmin}Add one to get started.{/if}</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Host</th><th>Port</th><th>Username</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as sv}
					<tr>
						<td><strong>{sv.name}</strong></td>
						<td>{sv.host}</td>
						<td>{sv.port}</td>
						<td>{sv.username}</td>
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
			<form onsubmit={(e) => { e.preventDefault(); save(); }}>
				<div class="grid-2">
					<div class="form-group">
						<label>Name</label>
						<input class="form-control" bind:value={form.name} required />
					</div>
					<div class="form-group">
						<label>Host / IP</label>
						<input class="form-control" bind:value={form.host} required />
					</div>
					<div class="form-group">
						<label>Port</label>
						<input class="form-control" type="number" bind:value={form.port} min="1" max="65535" required />
					</div>
					<div class="form-group">
						<label>SSH Username</label>
						<input class="form-control" bind:value={form.username} required />
					</div>
				</div>
				<div class="form-group">
					<label>SSH Private Key (PEM){editingId ? ' — leave blank to keep existing' : ''}</label>
					<textarea class="form-control" bind:value={form.ssh_private_key} rows="8"
						placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
						required={!editingId}></textarea>
				</div>
				<div class="form-group">
					<label>Pre-run Command (optional)</label>
					<input class="form-control" bind:value={form.pre_command}
						placeholder="e.g. . /home/brett/ansible/bin/activate" />
					<small class="hint">Run before ansible-playbook to activate a virtualenv or set PATH.</small>
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
	.test-result { font-size: 0.75rem; margin-top: 0.25rem; }
	.test-result.ok { color: var(--success); }
	.test-result:not(.ok) { color: var(--danger); }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 600px; max-height: 90vh; overflow-y: auto; }
</style>
