<script lang="ts">
	import { onMount } from 'svelte';
	import { runs as runsApi } from '$lib/api';
	import type { Run } from '$lib/types';

	let list = $state<Run[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try { list = await runsApi.list(); }
		finally { loading = false; }
	});

	function statusClass(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}

	function duration(r: Run) {
		if (!r.started_at || !r.finished_at) return '—';
		const ms = new Date(r.finished_at).getTime() - new Date(r.started_at).getTime();
		const s = Math.floor(ms / 1000);
		return s < 60 ? `${s}s` : `${Math.floor(s/60)}m ${s%60}s`;
	}
</script>

<div class="page-header">
	<h1>Run History</h1>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No runs yet. Run a form from the Forms page.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Run ID</th><th>Status</th><th>Duration</th><th>Started</th><th>Actions</th></tr></thead>
			<tbody>
				{#each list as run}
					<tr>
						<td><code>{run.id.slice(0,8)}...</code></td>
						<td><span class="badge {statusClass(run.status)}">{run.status}</span></td>
						<td>{duration(run)}</td>
						<td>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</td>
						<td><a href="/runs/{run.id}" class="btn btn-sm btn-secondary">View</a></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
