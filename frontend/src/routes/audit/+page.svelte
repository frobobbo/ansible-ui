<script lang="ts">
	import { onMount } from 'svelte';
	import { audit as auditApi } from '$lib/api';
	import type { AuditLog } from '$lib/types';

	const PAGE_SIZE = 50;

	let logs = $state<AuditLog[]>([]);
	let total = $state(0);
	let page = $state(0);
	let loading = $state(true);

	let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try {
			const res = await auditApi.list({ limit: PAGE_SIZE, offset: page * PAGE_SIZE });
			logs = res.data;
			total = res.total;
		} finally {
			loading = false;
		}
	}

	async function goTo(p: number) {
		page = p;
		await load();
	}

	function actionBadge(action: string) {
		if (action === 'delete') return 'badge-danger';
		if (action === 'create' || action === 'upload') return 'badge-success';
		if (action === 'update') return 'badge-info';
		return 'badge-muted';
	}
</script>

<div class="page-header">
	<h1>Audit Log</h1>
	<span class="total-label">{total} entries</span>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if logs.length === 0}
	<p class="empty-state">No audit entries yet.</p>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead>
				<tr>
					<th>Time</th>
					<th>User</th>
					<th>Action</th>
					<th>Resource</th>
					<th>Resource ID</th>
					<th>IP</th>
				</tr>
			</thead>
			<tbody>
				{#each logs as log}
					<tr>
						<td class="mono">{new Date(log.created_at).toLocaleString()}</td>
						<td>{log.username || '—'}</td>
						<td><span class="badge {actionBadge(log.action)}">{log.action}</span></td>
						<td>{log.resource}</td>
						<td class="mono resource-id" title={log.resource_id}>{log.resource_id ? log.resource_id.slice(0, 8) + '…' : '—'}</td>
						<td class="mono">{log.ip || '—'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if totalPages > 1}
		<div class="pagination">
			<button class="btn btn-sm btn-secondary" disabled={page === 0} onclick={() => goTo(page - 1)}>← Prev</button>
			<span>Page {page + 1} of {totalPages}</span>
			<button class="btn btn-sm btn-secondary" disabled={page >= totalPages - 1} onclick={() => goTo(page + 1)}>Next →</button>
		</div>
	{/if}
{/if}

<style>
	.total-label { font-size: 0.875rem; color: var(--text-muted); align-self: center; }
	.mono { font-family: monospace; font-size: 0.8rem; }
	.resource-id { max-width: 100px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.pagination { display: flex; align-items: center; gap: 0.75rem; justify-content: center; margin-top: 1.5rem; }
	.pagination span { font-size: 0.875rem; color: var(--text-muted); }
</style>
