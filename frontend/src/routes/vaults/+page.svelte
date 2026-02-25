<script lang="ts">
	import { onMount } from 'svelte';
	import { vaults as vaultsApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { goto } from '$app/navigation';
	import type { Vault } from '$lib/types';

	let list = $state<Vault[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Modal state
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ name: '', description: '', password: '' });
	let saving = $state(false);
	let formError = $state('');

	onMount(async () => {
		if (!$isAdmin) { goto('/'); return; }
		await load();
	});

	async function load() {
		loading = true;
		try { list = await vaultsApi.list(); }
		catch { error = 'Failed to load vaults'; }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = { name: '', description: '', password: '' };
		formError = '';
		showModal = true;
	}

	function openEdit(v: Vault) {
		editingId = v.id;
		form = { name: v.name, description: v.description, password: '' };
		formError = '';
		showModal = true;
	}

	async function save() {
		saving = true;
		formError = '';
		try {
			if (editingId) {
				await vaultsApi.update(editingId, form);
			} else {
				await vaultsApi.create(form);
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
		if (!confirm('Delete this vault? Any forms using it will lose their vault reference.')) return;
		try { await vaultsApi.delete(id); await load(); }
		catch { alert('Delete failed'); }
	}
</script>

<div class="page-header">
	<h1>Vaults</h1>
	<button class="btn btn-primary" onclick={openCreate}>+ Add Vault</button>
</div>

<div class="alert alert-info">
	Vault passwords are encrypted at rest (AES-256-GCM). Passwords are never returned by the API — only used at run time to pass <code>--vault-password-file</code> to ansible-playbook.
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No vaults configured. Add one to use ansible-vault encrypted variables.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Created</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as v}
					<tr>
						<td><strong>{v.name}</strong></td>
						<td>{v.description || '—'}</td>
						<td>{new Date(v.created_at).toLocaleDateString()}</td>
						<td>
							<div class="actions">
								<button class="btn btn-sm btn-secondary" onclick={() => openEdit(v)}>Edit</button>
								<button class="btn btn-sm btn-danger" onclick={() => remove(v.id)}>Delete</button>
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
			<h2>{editingId ? 'Edit Vault' : 'Add Vault'}</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }}>
				<div class="form-group">
					<label>Name</label>
					<input class="form-control" bind:value={form.name} required />
				</div>
				<div class="form-group">
					<label>Description (optional)</label>
					<input class="form-control" bind:value={form.description} />
				</div>
				<div class="form-group">
					<label>Vault Password{editingId ? ' — leave blank to keep existing' : ''}</label>
					<input class="form-control" type="password" bind:value={form.password}
						required={!editingId}
						placeholder={editingId ? '••••••••' : 'Enter vault password'} />
					<small class="hint">Stored encrypted (AES-256-GCM). Never exposed via the API.</small>
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
	.alert-info { background: #eff6ff; border: 1px solid #bfdbfe; color: #1e40af; border-radius: var(--radius); padding: 0.75rem 1rem; margin-bottom: 1rem; font-size: 0.875rem; }
	.alert-info code { background: #dbeafe; padding: 0.1em 0.3em; border-radius: 3px; font-size: 0.85em; }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 480px; }
</style>
