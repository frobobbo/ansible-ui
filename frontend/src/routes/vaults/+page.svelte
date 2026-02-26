<script lang="ts">
	import { onMount } from 'svelte';
	import { vaults as vaultsApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast, confirmDialog } from '$lib/toast';
	import { goto } from '$app/navigation';
	import type { Vault } from '$lib/types';

	let list = $state<Vault[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Create/edit modal
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let editingVaultFileName = $state('');
	let form = $state({ name: '', description: '', password: '' });
	let saving = $state(false);
	let formError = $state('');
	let fileUploading = $state(false);
	let stagedFile = $state<File | null>(null);

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
		editingVaultFileName = '';
		stagedFile = null;
		form = { name: '', description: '', password: '' };
		formError = '';
		showModal = true;
	}

	function openEdit(v: Vault) {
		editingId = v.id;
		editingVaultFileName = v.vault_file_name;
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
				toast.success('Vault updated');
			} else {
				const created = await vaultsApi.create(form);
				if (stagedFile) {
					await vaultsApi.uploadFile(created.id, stagedFile);
				}
				toast.success('Vault created');
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
		if (!(await confirmDialog('Delete this vault? Any forms using it will lose their vault reference.'))) return;
		try {
			await vaultsApi.delete(id);
			await load();
			toast.success('Vault deleted');
		} catch {
			toast.error('Delete failed');
		}
	}

	async function handleFileUpload(id: string, input: HTMLInputElement) {
		const file = input.files?.[0];
		if (!file) return;
		fileUploading = true;
		try {
			const updated = await vaultsApi.uploadFile(id, file);
			editingVaultFileName = updated.vault_file_name;
			list = list.map((v) => (v.id === id ? updated : v));
			toast.success('File uploaded');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Upload failed');
		} finally {
			fileUploading = false;
			input.value = '';
		}
	}

	async function removeFile(id: string) {
		if (!(await confirmDialog('Remove the vault file from this vault?', { confirmText: 'Remove' }))) return;
		try {
			const updated = await vaultsApi.deleteFile(id);
			editingVaultFileName = '';
			list = list.map((v) => (v.id === id ? updated : v));
			toast.success('File removed');
		} catch {
			toast.error('Failed to remove file');
		}
	}
</script>

<div class="page-header">
	<h1>Vaults</h1>
	<button class="btn btn-primary" onclick={openCreate}>+ Add Vault</button>
</div>

<div class="alert-info">
	Vault passwords are encrypted at rest (AES-256-GCM). Passwords are never returned by the API — only used at run time to pass <code>--vault-password-file</code> to ansible-playbook. If you upload a vault YAML file, it will be passed as <code>--extra-vars "@file"</code> so its variables are decrypted automatically.
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No vaults configured. Add one to use ansible-vault encrypted variables.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Vault File</th><th>Created</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as v}
					<tr>
						<td><strong>{v.name}</strong></td>
						<td>{v.description || '—'}</td>
						<td>
							{#if v.vault_file_name}
								<span class="file-badge">{v.vault_file_name}</span>
							{:else}
								<span class="no-file">None</span>
							{/if}
						</td>
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
				<div class="form-group">
					<label>Vault YAML File (optional)</label>
					{#if editingId}
						{#if editingVaultFileName}
							<div class="file-current">
								<span class="file-badge">{editingVaultFileName}</span>
								<button type="button" class="btn btn-sm btn-danger" style="margin-left:0.5rem"
									onclick={() => removeFile(editingId!)}>Remove</button>
							</div>
						{:else}
							<label class="btn btn-secondary file-label" class:disabled={fileUploading}>
								{fileUploading ? 'Uploading…' : 'Choose File…'}
								<input
									type="file"
									accept=".yml,.yaml"
									style="display:none"
									disabled={fileUploading}
									onchange={(e) => handleFileUpload(editingId!, e.currentTarget as HTMLInputElement)}
								/>
							</label>
						{/if}
					{:else}
						{#if stagedFile}
							<div class="file-current">
								<span class="file-badge">{stagedFile.name}</span>
								<button type="button" class="btn btn-sm btn-danger" style="margin-left:0.5rem"
									onclick={() => stagedFile = null}>Remove</button>
							</div>
						{:else}
							<label class="btn btn-secondary file-label">
								Choose File…
								<input
									type="file"
									accept=".yml,.yaml"
									style="display:none"
									onchange={(e) => {
										const input = e.currentTarget as HTMLInputElement;
										stagedFile = input.files?.[0] ?? null;
									}}
								/>
							</label>
						{/if}
					{/if}
					<small class="hint">Passed as <code>--extra-vars "@file"</code> to ansible-playbook.</small>
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
	.file-badge { display: inline-flex; align-items: center; background: #f0fdf4; border: 1px solid #bbf7d0; color: #166534; border-radius: 4px; padding: 0.15rem 0.5rem; font-size: 0.8rem; }
	.no-file { color: #94a3b8; font-size: 0.85rem; }
	.file-current { display: flex; align-items: center; }
	.file-label { cursor: pointer; display: inline-flex; align-items: center; }
	.file-label.disabled { opacity: 0.6; cursor: not-allowed; }
	.hint { display: block; margin-top: 0.25rem; font-size: 0.8rem; color: #64748b; }
	.hint code { background: #f1f5f9; padding: 0.1em 0.3em; border-radius: 3px; }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 480px; }
</style>
