<script lang="ts">
	import { goto } from '$app/navigation';
	import { serverGroups as sgApi, ApiError } from '$lib/api';

	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let error = $state('');

	async function save() {
		saving = true; error = '';
		try {
			await sgApi.create({ name, description });
			goto('/server-groups');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}
</script>

<div class="page-header">
	<h1>New Server Group</h1>
	<a href="/server-groups" class="btn btn-secondary">← Back</a>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

<form onsubmit={(e) => { e.preventDefault(); save(); }}>
	<div class="card">
		<h2>Group Details</h2>
		<div class="grid-2">
			<div class="form-group">
				<label>Name</label>
				<input class="form-control" bind:value={name} required />
			</div>
			<div class="form-group">
				<label>Description</label>
				<input class="form-control" bind:value={description} />
			</div>
		</div>
		<p class="hint" style="margin-top:0.5rem">After creating the group, edit it to add member servers.</p>
	</div>

	<div class="actions" style="justify-content:flex-end">
		<button type="submit" class="btn btn-primary" disabled={saving}>
			{saving ? 'Creating...' : 'Create Group'}
		</button>
	</div>
</form>
