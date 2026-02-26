<script lang="ts">
	import { onMount } from 'svelte';
	import { runs as runsApi } from '$lib/api';
	import type { Run } from '$lib/types';

	const PAGE_SIZE = 25;

	let list = $state<Run[]>([]);
	let loading = $state(true);
	let totalCount = $state(0);
	let currentPage = $state(0);
	let statusFilter = $state('');

	let totalPages = $derived(Math.max(1, Math.ceil(totalCount / PAGE_SIZE)));

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try {
			const { data, total } = await runsApi.list({ limit: PAGE_SIZE, offset: currentPage * PAGE_SIZE });
			list = data ?? [];
			totalCount = total;
		} finally {
			loading = false;
		}
	}

	async function goToPage(p: number) {
		currentPage = p;
		await load();
	}

	let filtered = $derived(
		statusFilter ? list.filter((r) => r.status === statusFilter) : list
	);

	function statusClass(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}

	function duration(r: Run) {
		if (!r.started_at || !r.finished_at) return '—';
		const ms = new Date(r.finished_at).getTime() - new Date(r.started_at).getTime();
		const s = Math.floor(ms / 1000);
		return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
	}
</script>

<div class="page-header">
	<h1>Run History</h1>
	<div class="header-right">
		<select class="form-control status-filter" bind:value={statusFilter}>
			<option value="">All statuses</option>
			<option value="pending">Pending</option>
			<option value="running">Running</option>
			<option value="success">Success</option>
			<option value="failed">Failed</option>
		</select>
	</div>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No runs yet. Run a form from the Forms page.</div>
{:else if filtered.length === 0}
	<div class="empty-state">No runs with status "{statusFilter}" on this page.</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead><tr><th>Run ID</th><th>Status</th><th>Duration</th><th>Started</th><th>Actions</th></tr></thead>
			<tbody>
				{#each filtered as run}
					<tr>
						<td><code>{run.id.slice(0, 8)}...</code></td>
						<td><span class="badge {statusClass(run.status)}">{run.status}</span></td>
						<td>{duration(run)}</td>
						<td>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</td>
						<td><a href="/runs/{run.id}" class="btn btn-sm btn-secondary">View</a></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if totalPages > 1}
		<div class="paginator">
			<button class="btn btn-secondary btn-sm" onclick={() => goToPage(0)} disabled={currentPage === 0}>«</button>
			<button class="btn btn-secondary btn-sm" onclick={() => goToPage(currentPage - 1)} disabled={currentPage === 0}>‹ Prev</button>
			<span class="page-info">Page {currentPage + 1} of {totalPages} &nbsp;·&nbsp; {totalCount} total</span>
			<button class="btn btn-secondary btn-sm" onclick={() => goToPage(currentPage + 1)} disabled={currentPage >= totalPages - 1}>Next ›</button>
			<button class="btn btn-secondary btn-sm" onclick={() => goToPage(totalPages - 1)} disabled={currentPage >= totalPages - 1}>»</button>
		</div>
	{:else}
		<p class="total-label">{totalCount} run{totalCount === 1 ? '' : 's'}</p>
	{/if}
{/if}

<style>
	.header-right { display: flex; gap: 0.75rem; align-items: center; }
	.status-filter { width: 160px; }
	.paginator { display: flex; align-items: center; gap: 0.5rem; justify-content: center; margin-top: 1rem; }
	.page-info { font-size: 0.85rem; color: var(--text-muted); padding: 0 0.5rem; }
	.total-label { text-align: center; color: var(--text-muted); font-size: 0.85rem; margin-top: 0.75rem; }
</style>
