<script lang="ts">
	import { onMount } from 'svelte';
	import { sshCerts as api, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast, confirmDialog } from '$lib/toast';
	import type { SSHCert } from '$lib/types';

	let list = $state<SSHCert[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Modal state
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ name: '', description: '' });
	let saving = $state(false);
	let formError = $state('');

	// Upload state
	let uploadingId = $state<string | null>(null);
	let uploadError = $state<Record<string, string>>({});
	let fileInputs = $state<Record<string, HTMLInputElement | null>>({});

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await api.list(); }
		catch { error = 'Failed to load SSH certs'; }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = { name: '', description: '' };
		formError = '';
		showModal = true;
	}

	function openEdit(cert: SSHCert) {
		editingId = cert.id;
		form = { name: cert.name, description: cert.description };
		formError = '';
		showModal = true;
	}

	async function save() {
		saving = true;
		formError = '';
		try {
			if (editingId) {
				await api.update(editingId, form);
			} else {
				await api.create(form);
			}
			showModal = false;
			toast.success(editingId ? 'SSH cert updated' : 'SSH cert created');
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function remove(id: string, name: string) {
		if (!(await confirmDialog(`Delete SSH cert "${name}"?`))) return;
		try {
			await api.delete(id);
			await load();
			toast.success('SSH cert deleted');
		} catch {
			toast.error('Delete failed');
		}
	}

	async function uploadFile(cert: SSHCert, file: File) {
		uploadingId = cert.id;
		uploadError[cert.id] = '';
		try {
			await api.uploadFile(cert.id, file);
			toast.success('Certificate uploaded');
			await load();
		} catch (err) {
			uploadError[cert.id] = err instanceof ApiError ? err.message : 'Upload failed';
		} finally {
			uploadingId = null;
			// Reset the file input so the same file can be re-selected if needed
			if (fileInputs[cert.id]) fileInputs[cert.id]!.value = '';
		}
	}

	async function removeFile(cert: SSHCert) {
		if (!(await confirmDialog(`Remove the certificate file from "${cert.name}"?`))) return;
		try {
			await api.deleteFile(cert.id);
			toast.success('Certificate removed');
			await load();
		} catch {
			toast.error('Failed to remove certificate');
		}
	}
</script>

<div class="page-header">
	<h1>SSH Certs</h1>
	{#if $isAdmin}
		<button class="btn btn-primary" onclick={openCreate}>+ Add SSH Cert</button>
	{/if}
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No SSH certs stored. {#if $isAdmin}Add one to get started.{/if}</div>
{:else}
	<div class="cert-list">
		{#each list as cert}
			<div class="card cert-card">
				<div class="cert-header">
					<div class="cert-meta">
						<strong>{cert.name}</strong>
						{#if cert.description}
							<span class="cert-desc">{cert.description}</span>
						{/if}
					</div>
					{#if $isAdmin}
						<div class="actions">
							<button class="btn btn-sm btn-secondary" onclick={() => openEdit(cert)}>Edit</button>
							<button class="btn btn-sm btn-danger" onclick={() => remove(cert.id, cert.name)}>Delete</button>
						</div>
					{/if}
				</div>

				<div class="cert-file-row">
					{#if cert.file_name}
						<div class="cert-file-present">
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
								<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
								<polyline points="14,2 14,8 20,8"/>
							</svg>
							<span class="file-name">{cert.file_name}</span>
							<span class="badge badge-ok">Stored</span>
						</div>
						{#if $isAdmin}
							<button class="btn btn-sm btn-danger-ghost" onclick={() => removeFile(cert)}>Remove</button>
						{/if}
					{:else}
						<span class="no-file">No certificate uploaded</span>
						{#if $isAdmin}
							<label class="btn btn-sm btn-secondary upload-label">
								{uploadingId === cert.id ? 'Uploading...' : 'Upload Certificate'}
								<input
									type="file"
									class="hidden-file-input"
									bind:this={fileInputs[cert.id]}
									disabled={uploadingId === cert.id}
									onchange={(e) => {
										const f = (e.target as HTMLInputElement).files?.[0];
										if (f) uploadFile(cert, f);
									}}
								/>
							</label>
						{/if}
					{/if}
				</div>

				{#if cert.file_name && $isAdmin}
					<div class="cert-reupload">
						<label class="btn btn-sm btn-secondary upload-label">
							{uploadingId === cert.id ? 'Uploading...' : 'Replace Certificate'}
							<input
								type="file"
								class="hidden-file-input"
								bind:this={fileInputs[cert.id + '-replace']}
								disabled={uploadingId === cert.id}
								onchange={(e) => {
									const f = (e.target as HTMLInputElement).files?.[0];
									if (f) uploadFile(cert, f);
								}}
							/>
						</label>
					</div>
				{/if}

				{#if uploadError[cert.id]}
					<div class="alert alert-error" style="margin-top:0.5rem">{uploadError[cert.id]}</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

{#if showModal}
	<div class="modal-overlay" onclick={() => showModal = false} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<h2>{editingId ? 'Edit SSH Cert' : 'Add SSH Cert'}</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }} autocomplete="off">
				<div class="form-group">
					<label>Name</label>
					<input class="form-control" bind:value={form.name} required />
				</div>
				<div class="form-group">
					<label>Description <span class="hint-inline">(optional)</span></label>
					<input class="form-control" bind:value={form.description} placeholder="e.g. Production deploy key" />
				</div>
				<small class="hint">
					After saving, upload the certificate file from the main list. Certificate bytes are encrypted with AES-256-GCM before being stored in the database — the plaintext cert never touches the filesystem.
				</small>
				<div class="actions" style="justify-content:flex-end; margin-top:1rem">
					<button type="button" class="btn btn-secondary" onclick={() => showModal = false}>Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
	.cert-list { display: flex; flex-direction: column; gap: 0.75rem; }
	.cert-card { padding: 1rem 1.25rem; display: flex; flex-direction: column; gap: 0.75rem; }
	.cert-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; }
	.cert-meta { display: flex; flex-direction: column; gap: 0.2rem; }
	.cert-desc { font-size: 0.85rem; color: var(--text-muted); }
	.cert-file-row { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
	.cert-file-present { display: flex; align-items: center; gap: 0.4rem; font-size: 0.85rem; }
	.file-name { font-family: monospace; font-size: 0.8rem; color: var(--text-muted); }
	.no-file { font-size: 0.85rem; color: var(--text-muted); }
	.cert-reupload { display: flex; }
	.badge { display: inline-block; padding: 0.1rem 0.45rem; border-radius: 9999px; font-size: 0.7rem; font-weight: 600; }
	.badge-ok { background: #dcfce7; color: #15803d; }
	.upload-label { cursor: pointer; position: relative; }
	.hidden-file-input { position: absolute; inset: 0; opacity: 0; cursor: pointer; width: 100%; }
	.btn-danger-ghost {
		background: none; border: 1px solid var(--danger);
		color: var(--danger); padding: 0.25rem 0.6rem;
		border-radius: var(--radius); font-size: 0.8rem; cursor: pointer;
		transition: background 0.15s;
	}
	.btn-danger-ghost:hover { background: color-mix(in srgb, var(--danger) 10%, transparent); }
	.hint-inline { font-weight: normal; font-size: 0.8rem; color: var(--text-muted); }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 480px; }
</style>
