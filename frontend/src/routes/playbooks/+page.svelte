<script lang="ts">
	import { onMount } from 'svelte';
	import { playbooks as playbooksApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast, confirmDialog } from '$lib/toast';
	import type { Playbook } from '$lib/types';

	let list = $state<Playbook[]>([]);
	let loading = $state(true);
	let error = $state('');
	let filter = $state('');

	let filtered = $derived(
		filter.trim()
			? list.filter(
					(p) =>
						p.name.toLowerCase().includes(filter.toLowerCase()) ||
						(p.description ?? '').toLowerCase().includes(filter.toLowerCase()) ||
						p.repo_url.toLowerCase().includes(filter.toLowerCase())
				)
			: list
	);

	const emptyForm = () => ({
		name: '', description: '', repo_url: '', branch: 'main', token: ''
	});

	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state(emptyForm());
	let saving = $state(false);
	let formError = $state('');

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await playbooksApi.list(); }
		catch { error = 'Failed to load playbook sources'; }
		finally { loading = false; }
	}

	function openCreate() {
		editingId = null;
		form = emptyForm();
		formError = '';
		showModal = true;
	}

	function openEdit(p: Playbook) {
		editingId = p.id;
		form = { name: p.name, description: p.description, repo_url: p.repo_url, branch: p.branch, token: '' };
		formError = '';
		showModal = true;
	}

	async function save() {
		if (!form.name || !form.repo_url) {
			formError = 'Name and Repo URL are required.';
			return;
		}
		saving = true;
		formError = '';
		try {
			const payload = { ...form };
			if (editingId) {
				await playbooksApi.update(editingId, payload);
				toast.success('Playbook source updated');
			} else {
				await playbooksApi.create(payload);
				toast.success('Playbook source added');
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
		if (!(await confirmDialog('Delete this playbook source? Forms using it will be affected.'))) return;
		try {
			await playbooksApi.delete(id);
			await load();
			toast.success('Playbook source deleted');
		} catch {
			toast.error('Delete failed');
		}
	}
</script>

<div class="page-header">
	<h1>Playbook Source</h1>
	<div class="header-right">
		<input class="form-control search" placeholder="Search..." bind:value={filter} />
		{#if $isAdmin}
			<button class="btn btn-primary" onclick={openCreate}>+ Add Source</button>
		{/if}
	</div>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No playbook sources configured yet.{#if $isAdmin} Add a Git repository to get started.{/if}</div>
{:else if filtered.length === 0}
	<div class="empty-state">No sources match "{filter}".</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead>
				<tr>
					<th>Name</th>
					<th>Repository</th>
					<th>Branch</th>
					{#if $isAdmin}<th>Actions</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each filtered as pb}
					<tr>
						<td>
							<strong>{pb.name}</strong>
							{#if pb.description}<div class="sub">{pb.description}</div>{/if}
						</td>
						<td class="repo-cell">
							<span class="repo-url" title={pb.repo_url}>{pb.repo_url}</span>
						</td>
						<td><span class="badge badge-info">{pb.branch}</span></td>
						{#if $isAdmin}
							<td>
								<div class="actions">
									<button class="btn btn-sm btn-secondary" onclick={() => openEdit(pb)}>Edit</button>
									<button class="btn btn-sm btn-danger" onclick={() => remove(pb.id)}>Delete</button>
								</div>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

{#if showModal}
	<div class="modal-overlay" onclick={() => showModal = false} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<h2>{editingId ? 'Edit' : 'Add'} Playbook Source</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }} autocomplete="off">
				<div class="form-group">
					<label>Name <span class="req">*</span></label>
					<input class="form-control" bind:value={form.name} required placeholder="e.g. Production Playbooks" />
				</div>
				<div class="form-group">
					<label>Description</label>
					<input class="form-control" bind:value={form.description} placeholder="Optional description" />
				</div>
				<div class="form-group">
					<label>Repository URL <span class="req">*</span></label>
					<input class="form-control" bind:value={form.repo_url} required placeholder="https://github.com/org/repo.git" />
					<span class="hint">HTTPS URL. For private repos, enter a token below.</span>
				</div>
				<div class="form-group">
					<label>Branch <span class="req">*</span></label>
					<input class="form-control" bind:value={form.branch} required placeholder="main" />
					<span class="hint">The specific playbook file is chosen per form.</span>
				</div>
				<div class="form-group">
					<label>Access Token</label>
					<input class="form-control" type="password" bind:value={form.token} placeholder={editingId ? 'Leave blank to keep existing token' : 'GitHub/GitLab PAT for private repos'} autocomplete="new-password" />
					<span class="hint">Stored securely. Leave blank for public repositories.</span>
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
	.sub { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem; }
	.repo-cell { max-width: 280px; }
	.repo-url { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.8rem; color: var(--text-muted); font-family: monospace; }
	.path { font-size: 0.8rem; background: var(--bg); padding: 0.1rem 0.35rem; border-radius: 4px; }
	.req { color: var(--danger); }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: var(--surface); border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 540px; border: 1px solid var(--border); }
</style>
