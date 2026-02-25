<script lang="ts">
	import { onMount } from 'svelte';
	import { forms as formsApi, ApiError } from '$lib/api';
	import type { Form } from '$lib/types';

	let list = $state<Form[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		loading = true;
		try { list = await formsApi.list(); }
		catch { error = 'Failed to load forms'; }
		finally { loading = false; }
	});

	async function remove(id: string) {
		if (!confirm('Delete this form?')) return;
		try { await formsApi.delete(id); list = list.filter(f => f.id !== id); }
		catch { alert('Delete failed'); }
	}
</script>

<div class="page-header">
	<h1>Forms</h1>
	<a href="/forms/new" class="btn btn-primary">+ New Form</a>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No forms yet. Create a form to run playbooks with variables.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Updated</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as f}
					<tr>
						<td><strong>{f.name}</strong></td>
						<td>{f.description || '—'}</td>
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
