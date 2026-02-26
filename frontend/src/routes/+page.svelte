<script lang="ts">
	import { onMount } from 'svelte';
	import { servers, playbooks, forms, runs } from '$lib/api';
	import { currentUser, isAdmin, isEditor } from '$lib/stores';
	import type { Form, Run } from '$lib/types';

	let quickActions = $state<Form[]>([]);
	let formCount = $state(0);
	let runCount = $state(0);
	let serverCount = $state(0);
	let playbookCount = $state(0);
	let recentRuns = $state<Run[]>([]);
	let loading = $state(true);

	onMount(async () => {
		const role = $currentUser?.role;
		try {
			const promises: Promise<unknown>[] = [forms.quickActions().then((r) => (quickActions = r))];

			if (role === 'admin' || role === 'editor') {
				promises.push(
					forms.list().then((r) => (formCount = r.length)),
					runs.list({ limit: 5 }).then(({ data, total }) => {
						runCount = total;
						recentRuns = data ?? [];
					})
				);
			}
			if (role === 'admin') {
				promises.push(
					servers.list().then((r) => (serverCount = r.length)),
					playbooks.list().then((r) => (playbookCount = r.length))
				);
			}
			await Promise.all(promises);
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
	{#if $isAdmin}
		<div class="stat-grid">
			<a href="/servers" class="stat-card">
				<div class="stat-value">{serverCount}</div>
				<div class="stat-label">Servers</div>
			</a>
			<a href="/playbooks" class="stat-card">
				<div class="stat-value">{playbookCount}</div>
				<div class="stat-label">Playbooks</div>
			</a>
			<a href="/forms" class="stat-card">
				<div class="stat-value">{formCount}</div>
				<div class="stat-label">Forms</div>
			</a>
			<a href="/runs" class="stat-card">
				<div class="stat-value">{runCount}</div>
				<div class="stat-label">Total Runs</div>
			</a>
		</div>
	{:else if $isEditor}
		<div class="stat-grid stat-grid-2">
			<a href="/forms" class="stat-card">
				<div class="stat-value">{formCount}</div>
				<div class="stat-label">Forms</div>
			</a>
			<a href="/runs" class="stat-card">
				<div class="stat-value">{runCount}</div>
				<div class="stat-label">Total Runs</div>
			</a>
		</div>
	{/if}

	<div class="section-header">
		<h2>Quick Actions</h2>
	</div>

	{#if quickActions.length === 0}
		<div class="empty-state">
			No quick actions configured.
			{#if $isEditor}
				<a href="/forms">Edit a form</a> and enable "Quick Action" to add it here.
			{/if}
		</div>
	{:else}
		<div class="qa-grid">
			{#each quickActions as qa}
				<a href="/forms/{qa.id}/run" class="qa-card">
					<div class="qa-image">
						{#if qa.image_name}
							<img src="/api/forms/{qa.id}/image" alt={qa.name} />
						{:else}
							<div class="qa-placeholder">▶</div>
						{/if}
					</div>
					<div class="qa-body">
						<div class="qa-name">{qa.name}</div>
						{#if qa.description}<div class="qa-desc">{qa.description}</div>{/if}
					</div>
					<div class="qa-run">Run →</div>
				</a>
			{/each}
		</div>
	{/if}

	{#if $isEditor && recentRuns.length > 0}
		<div class="card" style="margin-top:1.5rem">
			<h2>Recent Runs</h2>
			<table class="table">
				<thead><tr><th>Run ID</th><th>Status</th><th>Started</th></tr></thead>
				<tbody>
					{#each recentRuns as run}
						<tr>
							<td><a href="/runs/{run.id}">{run.id.slice(0, 8)}...</a></td>
							<td><span class="badge {statusBadge(run.status)}">{run.status}</span></td>
							<td>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<div style="margin-top:0.75rem;text-align:right">
				<a href="/runs" class="btn btn-secondary btn-sm">View all {runCount} runs →</a>
			</div>
		</div>
	{/if}
{/if}

<style>
	.stat-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; margin-bottom: 1.5rem; }
	.stat-grid-2 { grid-template-columns: repeat(2, 1fr); max-width: 480px; }
	@media (max-width: 768px) { .stat-grid { grid-template-columns: repeat(2, 1fr); } }
	.stat-card { background: white; border: 1px solid var(--border); border-radius: var(--radius); padding: 1.5rem; text-decoration: none; color: inherit; text-align: center; transition: box-shadow 0.15s; }
	.stat-card:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.1); text-decoration: none; }
	.stat-value { font-size: 2.5rem; font-weight: 700; color: var(--primary); }
	.stat-label { font-size: 0.875rem; color: var(--text-muted); margin-top: 0.25rem; }
	.section-header { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
	.section-header h2 { margin: 0; }
	.qa-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 1rem; }
	.qa-card { background: white; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; text-decoration: none; color: inherit; display: flex; flex-direction: column; transition: box-shadow 0.15s, transform 0.15s; }
	.qa-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.12); transform: translateY(-2px); text-decoration: none; }
	.qa-image { height: 120px; background: #f1f5f9; display: flex; align-items: center; justify-content: center; overflow: hidden; }
	.qa-image img { width: 100%; height: 100%; object-fit: cover; }
	.qa-placeholder { font-size: 2.5rem; color: #94a3b8; }
	.qa-body { padding: 0.75rem 1rem 0.5rem; flex: 1; }
	.qa-name { font-weight: 600; font-size: 0.95rem; }
	.qa-desc { font-size: 0.8rem; color: var(--text-muted); margin-top: 0.25rem; }
	.qa-run { padding: 0.5rem 1rem 0.75rem; font-size: 0.8rem; font-weight: 600; color: var(--primary); }
</style>
