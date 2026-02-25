<script lang="ts">
	import { onMount } from 'svelte';
	import { servers, playbooks, forms, runs } from '$lib/api';

	let stats = $state({ servers: 0, playbooks: 0, forms: 0, runs: 0, recent: [] as any[] });
	let loading = $state(true);

	onMount(async () => {
		try {
			const [svList, pbList, fmList, rnList] = await Promise.all([
				servers.list(), playbooks.list(), forms.list(), runs.list()
			]);
			stats = {
				servers: svList.length,
				playbooks: pbList.length,
				forms: fmList.length,
				runs: rnList.length,
				recent: rnList.slice(0, 5),
			};
		} finally {
			loading = false;
		}
	});

	function statusBadge(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}
</script>

<h1>Dashboard</h1>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else}
	<div class="stat-grid">
		<a href="/servers" class="stat-card">
			<div class="stat-value">{stats.servers}</div>
			<div class="stat-label">Servers</div>
		</a>
		<a href="/playbooks" class="stat-card">
			<div class="stat-value">{stats.playbooks}</div>
			<div class="stat-label">Playbooks</div>
		</a>
		<a href="/forms" class="stat-card">
			<div class="stat-value">{stats.forms}</div>
			<div class="stat-label">Forms</div>
		</a>
		<a href="/runs" class="stat-card">
			<div class="stat-value">{stats.runs}</div>
			<div class="stat-label">Total Runs</div>
		</a>
	</div>

	{#if stats.recent.length > 0}
		<div class="card" style="margin-top:1.5rem">
			<h2>Recent Runs</h2>
			<table class="table">
				<thead><tr><th>Run ID</th><th>Status</th><th>Started</th></tr></thead>
				<tbody>
					{#each stats.recent as run}
						<tr>
							<td><a href="/runs/{run.id}">{run.id.slice(0, 8)}...</a></td>
							<td><span class="badge {statusBadge(run.status)}">{run.status}</span></td>
							<td>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	.stat-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; }
	@media (max-width: 768px) { .stat-grid { grid-template-columns: repeat(2, 1fr); } }
	.stat-card { background: white; border: 1px solid var(--border); border-radius: var(--radius); padding: 1.5rem; text-decoration: none; color: inherit; text-align: center; transition: box-shadow 0.15s; }
	.stat-card:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.1); text-decoration: none; }
	.stat-value { font-size: 2.5rem; font-weight: 700; color: var(--primary); }
	.stat-label { font-size: 0.875rem; color: var(--text-muted); margin-top: 0.25rem; }
</style>
