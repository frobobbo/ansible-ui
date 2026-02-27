<script lang="ts">
	import { onMount } from 'svelte';
	import { forms as formsApi } from '$lib/api';
	import { toast, confirmDialog } from '$lib/toast';
	import type { Form } from '$lib/types';

	let list = $state<Form[]>([]);
	let loading = $state(true);
	let filter = $state('');

	let filtered = $derived(
		filter.trim()
			? list.filter(
					(f) =>
						f.name.toLowerCase().includes(filter.toLowerCase()) ||
						(f.description ?? '').toLowerCase().includes(filter.toLowerCase())
				)
			: list
	);

	onMount(async () => {
		loading = true;
		try { list = await formsApi.list(); }
		catch { toast.error('Failed to load forms'); }
		finally { loading = false; }
	});

	async function remove(id: string) {
		if (!(await confirmDialog('Delete this form? This cannot be undone.'))) return;
		try {
			await formsApi.delete(id);
			list = list.filter((f) => f.id !== id);
			toast.success('Form deleted');
		} catch {
			toast.error('Delete failed');
		}
	}
</script>

<div class="page-header">
	<h1>Forms</h1>
	<div class="header-right">
		<input class="form-control search" placeholder="Search forms..." bind:value={filter} />
		<a href="/forms/new" class="btn btn-primary">+ New Form</a>
	</div>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No forms yet. Create a form to run playbooks with variables.</div>
{:else if filtered.length === 0}
	<div class="empty-state">No forms match "{filter}".</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Schedule</th><th>Updated</th><th>Actions</th></tr></thead>
			<tbody>
				{#each filtered as f}
					<tr>
						<td><strong>{f.name}</strong></td>
						<td>{f.description || '—'}</td>
						<td>
							{#if f.schedule_enabled && f.schedule_cron}
								<code class="sched-badge">{f.schedule_cron}</code>
								{#if f.next_run_at}
									<br /><small class="muted">next: {new Date(f.next_run_at).toLocaleString()}</small>
								{/if}
							{:else}—{/if}
						</td>
						<td>{new Date(f.updated_at).toLocaleDateString()}</td>
						<td>
							<div class="actions">
								<a href="/forms/{f.id}/run" class="btn btn-sm btn-primary">Run</a>
								<a href="/forms/{f.id}" class="btn btn-sm btn-secondary">Edit</a>
								<button class="btn btn-sm btn-danger" onclick={() => remove(f.id)}>Delete</button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.header-right { display: flex; gap: 0.75rem; align-items: center; }
	.search { width: 220px; }
	.sched-badge { background: #ede9fe; color: #5b21b6; border-radius: 4px; padding: 0.1rem 0.4rem; font-size: 0.78rem; }
	.muted { color: var(--text-muted, #64748b); font-size: 0.75rem; }
</style>
