<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { forms as formsApi, runs as runsApi, ApiError } from '$lib/api';
	import type { Form, FormField, Run } from '$lib/types';

	let id = $derived($page.params.id);
	let form = $state<Form | null>(null);
	let loading = $state(true);
	let variables = $state<Record<string, string>>({});
	let running = $state(false);
	let runResult = $state<Run | null>(null);
	let error = $state('');
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	onMount(async () => {
		form = await formsApi.get(id);
		if (form?.fields) {
			for (const f of form.fields) {
				variables[f.name] = f.default_value || (f.field_type === 'bool' ? 'false' : '');
			}
		}
		loading = false;
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});

	async function executeRun() {
		if (!form) return;
		running = true;
		error = '';
		runResult = null;

		// Convert typed values
		const typedVars: Record<string, unknown> = {};
		for (const field of form.fields || []) {
			const val = variables[field.name];
			if (field.field_type === 'number') typedVars[field.name] = Number(val);
			else if (field.field_type === 'bool') typedVars[field.name] = val === 'true';
			else typedVars[field.name] = val;
		}

		try {
			const { run_id } = await runsApi.create(id, typedVars);

			// Poll for status
			pollInterval = setInterval(async () => {
				const r = await runsApi.get(run_id);
				runResult = r;
				if (r && (r.status === 'success' || r.status === 'failed')) {
					if (pollInterval) clearInterval(pollInterval);
					pollInterval = null;
					running = false;
				}
			}, 2000);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to start run';
			running = false;
		}
	}

	function statusClass(status: string) {
		return { pending: 'badge-muted', running: 'badge-info', success: 'badge-success', failed: 'badge-danger' }[status] || 'badge-muted';
	}
</script>

<div class="page-header">
	<h1>Run: {form?.name ?? '...'}</h1>
	<a href="/forms/{id}" class="btn btn-secondary">← Edit Form</a>
</div>

{#if loading}
	<p class="empty-state">Loading form...</p>
{:else if form}
	{#if error}<div class="alert alert-error">{error}</div>{/if}

	<form onsubmit={(e) => { e.preventDefault(); executeRun(); }}>
		<div class="card">
			<h2>Variables</h2>
			{#if !form.fields || form.fields.length === 0}
				<p class="empty-state" style="padding:0.5rem 0">This form has no fields. The playbook will run with no extra variables.</p>
			{/if}
			{#each form.fields ?? [] as field}
				<div class="form-group">
					<label>
						{field.label}
						{#if field.required}<span class="required">*</span>{/if}
						<span class="var-name">({field.name})</span>
					</label>
					{#if field.field_type === 'bool'}
						<select class="form-control" bind:value={variables[field.name]}>
							<option value="false">false</option>
							<option value="true">true</option>
						</select>
					{:else if field.field_type === 'select'}
						<select class="form-control" bind:value={variables[field.name]}>
							{#each JSON.parse(field.options || '[]') as opt}
								<option value={opt}>{opt}</option>
							{/each}
						</select>
					{:else if field.field_type === 'number'}
						<input class="form-control" type="number" bind:value={variables[field.name]} required={field.required} />
					{:else}
						<input class="form-control" type="text" bind:value={variables[field.name]} required={field.required} />
					{/if}
				</div>
			{/each}
		</div>

		<div class="actions" style="justify-content:flex-end; margin-bottom:1.5rem">
			<button type="submit" class="btn btn-primary" disabled={running}>
				{running ? 'Running...' : '▶ Run Playbook'}
			</button>
		</div>
	</form>

	{#if runResult}
		<div class="card">
			<div class="run-header">
				<h2>Run Output</h2>
				<div class="run-meta">
					<span class="badge {statusClass(runResult.status)}">{runResult.status}</span>
					{#if runResult.started_at}
						<span class="meta-text">Started: {new Date(runResult.started_at).toLocaleString()}</span>
					{/if}
					{#if runResult.finished_at}
						<span class="meta-text">Finished: {new Date(runResult.finished_at).toLocaleString()}</span>
					{/if}
					<a href="/runs/{runResult.id}" class="btn btn-sm btn-secondary">View Full Run</a>
				</div>
			</div>
			{#if running}
				<p class="polling-notice">Polling for output every 2 seconds...</p>
			{/if}
			<pre class="output">{runResult.output || 'Waiting for output...'}</pre>
		</div>
	{/if}
{/if}

<style>
	.required { color: var(--danger); margin-left: 2px; }
	.var-name { font-size: 0.75rem; color: var(--text-muted); font-weight: normal; font-family: monospace; }
	.run-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem; }
	.run-header h2 { margin-bottom: 0; }
	.run-meta { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
	.meta-text { font-size: 0.8rem; color: var(--text-muted); }
	.polling-notice { font-size: 0.8rem; color: var(--text-muted); margin-bottom: 0.75rem; font-style: italic; }
	.output { background: #0f172a; color: #e2e8f0; padding: 1.25rem; border-radius: var(--radius); font-size: 0.8rem; line-height: 1.6; overflow-x: auto; white-space: pre-wrap; word-break: break-all; max-height: 500px; overflow-y: auto; }
</style>
