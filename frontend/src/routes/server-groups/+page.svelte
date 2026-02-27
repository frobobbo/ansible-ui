<script lang="ts">
	import { onMount } from 'svelte';
	import { serverGroups as sgApi, ApiError } from '$lib/api';
	import { confirmDialog, toast } from '$lib/toast';
	import type { ServerGroup } from '$lib/types';

	let list = $state<ServerGroup[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try { list = await sgApi.list() ?? []; }
		finally { loading = false; }
	});

	async function remove(g: ServerGroup) {
		const ok = await confirmDialog(`Delete server group "${g.name}"? Forms using this group will lose their server target.`);
		if (!ok) return;
		try {
			await sgApi.delete(g.id);
			list = list.filter(x => x.id !== g.id);
			toast.success('Server group deleted');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed');
		}
	}
</script>

<div class="page-header">
	<h1>Server Groups</h1>
	<a href="/server-groups/new" class="btn btn-primary">+ New Group</a>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No server groups yet. Create one to run playbooks across multiple servers.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Name</th><th>Description</th><th>Created</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as g}
					<tr>
						<td><strong>{g.name}</strong></td>
						<td>{g.description || '—'}</td>
						<td>{new Date(g.created_at).toLocaleDateString()}</td>
						<td class="actions">
							<a href="/server-groups/{g.id}" class="btn btn-sm btn-secondary">Edit</a>
							<button class="btn btn-sm btn-danger" onclick={() => remove(g)}>Delete</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
