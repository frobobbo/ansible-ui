<script lang="ts">
	import { onMount } from 'svelte';
	import { playbooks as playbooksApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import type { Playbook } from '$lib/types';

	let list = $state<Playbook[]>([]);
	let loading = $state(true);
	let error = $state('');

	let showUpload = $state(false);
	let uploadForm = $state({ name: '', description: '', file: null as File | null });
	let uploading = $state(false);
	let uploadError = $state('');

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await playbooksApi.list(); }
		catch { error = 'Failed to load playbooks'; }
		finally { loading = false; }
	}

	async function upload() {
		if (!uploadForm.file) return;
		uploading = true;
		uploadError = '';
		try {
			await playbooksApi.upload(uploadForm.name, uploadForm.description, uploadForm.file);
			showUpload = false;
			uploadForm = { name: '', description: '', file: null };
			await load();
		} catch (err) {
			uploadError = err instanceof ApiError ? err.message : 'Upload failed';
		} finally {
			uploading = false;
		}
	}

	async function remove(id: string) {
		if (!confirm('Delete this playbook? Forms using it will be affected.')) return;
		try { await playbooksApi.delete(id); await load(); }
		catch { alert('Delete failed'); }
	}

	function onFileChange(e: Event) {
		const input = e.target as HTMLInputElement;
		uploadForm.file = input.files?.[0] ?? null;
		if (uploadForm.file && !uploadForm.name) {
			uploadForm.name = uploadForm.file.name.replace(/\.ya?ml$/, '');
		}
	}
</script>

<div class="page-header">
	<h1>Playbooks</h1>
	{#if $isAdmin}
		<button class="btn btn-primary" onclick={() => showUpload = true}>+ Upload Playbook</button>
	{/if}
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No playbooks uploaded yet.{#if $isAdmin} Upload a YAML playbook to get started.{/if}</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Uploaded</th>{#if $isAdmin}<th>Actions</th>{/if}</tr></thead>
			<tbody>
				{#each list as pb}
					<tr>
						<td><strong>{pb.name}</strong></td>
						<td>{pb.description || '—'}</td>
						<td>{new Date(pb.created_at).toLocaleDateString()}</td>
						{#if $isAdmin}
							<td><button class="btn btn-sm btn-danger" onclick={() => remove(pb.id)}>Delete</button></td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

{#if showUpload}
	<div class="modal-overlay" onclick={() => showUpload = false} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<h2>Upload Playbook</h2>
			{#if uploadError}<div class="alert alert-error">{uploadError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); upload(); }}>
				<div class="form-group">
					<label>YAML File</label>
					<input class="form-control" type="file" accept=".yml,.yaml" onchange={onFileChange} required />
				</div>
				<div class="form-group">
					<label>Name</label>
					<input class="form-control" bind:value={uploadForm.name} required />
				</div>
				<div class="form-group">
					<label>Description (optional)</label>
					<input class="form-control" bind:value={uploadForm.description} />
				</div>
				<div class="actions" style="justify-content:flex-end">
					<button type="button" class="btn btn-secondary" onclick={() => showUpload = false}>Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={uploading}>{uploading ? 'Uploading...' : 'Upload'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 480px; }
</style>
