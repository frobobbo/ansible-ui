<script lang="ts">
	import { onMount } from 'svelte';
	import { users as usersApi, ApiError } from '$lib/api';
	import { currentUser } from '$lib/stores';
	import type { User } from '$lib/types';

	let list = $state<User[]>([]);
	let loading = $state(true);
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ username: '', password: '', role: 'viewer' as string });
	let saving = $state(false);
	let formError = $state('');

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await usersApi.list(); }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = { username: '', password: '', role: 'viewer' };
		formError = '';
		showModal = true;
	}

	function openEdit(u: User) {
		editingId = u.id;
		form = { username: u.username, password: '', role: u.role };
		formError = '';
		showModal = true;
	}

	async function save() {
		saving = true;
		formError = '';
		try {
			if (editingId) {
				await usersApi.update(editingId, form);
			} else {
				await usersApi.create({ username: form.username, password: form.password, role: form.role });
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
		if (id === $currentUser?.id) { alert('Cannot delete your own account'); return; }
		if (!confirm('Delete this user?')) return;
		try { await usersApi.delete(id); await load(); }
		catch { alert('Delete failed'); }
	}
</script>

<div class="page-header">
	<h1>Users</h1>
	<button class="btn btn-primary" onclick={openCreate}>+ Add User</button>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Username</th><th>Role</th><th>Created</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as u}
					<tr>
						<td><strong>{u.username}</strong>{#if u.id === $currentUser?.id} <span class="badge badge-info">You</span>{/if}</td>
						<td><span class="badge {u.role === 'admin' ? 'badge-warning' : u.role === 'editor' ? 'badge-info' : 'badge-muted'}">{u.role}</span></td>
						<td>{new Date(u.created_at).toLocaleDateString()}</td>
						<td>
							<div class="actions">
								<button class="btn btn-sm btn-secondary" onclick={() => openEdit(u)}>Edit</button>
								<button class="btn btn-sm btn-danger" onclick={() => remove(u.id)} disabled={u.id === $currentUser?.id}>Delete</button>
							</div>
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
			<h2>{editingId ? 'Edit User' : 'Create User'}</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }}>
				<div class="form-group">
					<label>Username</label>
					<input class="form-control" bind:value={form.username} required />
				</div>
				<div class="form-group">
					<label>Password{editingId ? ' — leave blank to keep existing' : ''}</label>
					<input class="form-control" type="password" bind:value={form.password} required={!editingId} />
				</div>
				<div class="form-group">
					<label>Role</label>
					<select class="form-control" bind:value={form.role}>
						<option value="viewer">Viewer — dashboard only</option>
						<option value="editor">Editor — dashboard + forms + run history</option>
						<option value="admin">Admin — full access</option>
					</select>
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
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 400px; }
</style>
