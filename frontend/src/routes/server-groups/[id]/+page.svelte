<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { serverGroups as sgApi, servers as serversApi, ApiError } from '$lib/api';
	import { toast } from '$lib/toast';
	import type { Server, ServerGroup } from '$lib/types';

	let id = $derived($page.params.id);
	let group = $state<ServerGroup | null>(null);
	let name = $state('');
	let description = $state('');
	let allServers = $state<Server[]>([]);
	let members = $state<Server[]>([]);
	let saving = $state(false);
	let savingMembers = $state(false);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		const [g, svList, memberList] = await Promise.all([
			sgApi.get(id),
			serversApi.list(),
			sgApi.getMembers(id),
		]);
		group = g;
		name = g?.name ?? '';
		description = g?.description ?? '';
		allServers = svList;
		members = memberList;
		loading = false;
	});

	let memberIds = $derived(new Set(members.map(m => m.id)));

	function toggleMember(sv: Server) {
		if (memberIds.has(sv.id)) {
			members = members.filter(m => m.id !== sv.id);
		} else {
			members = [...members, sv];
		}
	}

	async function save() {
		saving = true; error = '';
		try {
			await sgApi.update(id, { name, description });
			toast.success('Group saved');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function saveMembers() {
		savingMembers = true;
		try {
			await sgApi.setMembers(id, members.map(m => m.id));
			toast.success('Members saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to save members');
		} finally {
			savingMembers = false;
		}
	}
</script>

<div class="page-header">
	<h1>Edit Server Group</h1>
	<a href="/server-groups" class="btn btn-secondary">← Back</a>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else}
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
		</div>
		<div class="actions" style="justify-content:flex-end; margin-bottom:1.5rem">
			<button type="submit" class="btn btn-primary" disabled={saving}>
				{saving ? 'Saving...' : 'Save Details'}
			</button>
		</div>
	</form>

	<div class="card">
		<div class="card-header-row">
			<h2>Member Servers</h2>
			<button class="btn btn-primary btn-sm" onclick={saveMembers} disabled={savingMembers}>
				{savingMembers ? 'Saving...' : 'Save Members'}
			</button>
		</div>
		{#if allServers.length === 0}
			<p class="empty-state" style="padding:0.5rem 0">No servers configured yet.</p>
		{:else}
			<div class="member-list">
				{#each allServers as sv}
					<label class="member-row">
						<input type="checkbox" checked={memberIds.has(sv.id)} onchange={() => toggleMember(sv)} />
						<span class="member-name">{sv.name}</span>
						<span class="member-host">{sv.host}:{sv.port}</span>
					</label>
				{/each}
			</div>
			<p class="hint" style="margin-top:0.75rem">
				{members.length} of {allServers.length} servers selected. Running a form with this group will create one run per selected server.
			</p>
		{/if}
	</div>
{/if}

<style>
	.card-header-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
	.card-header-row h2 { margin-bottom: 0; }
	.member-list { display: flex; flex-direction: column; gap: 0.25rem; }
	.member-row {
		display: flex; align-items: center; gap: 0.75rem;
		padding: 0.5rem 0.75rem; border-radius: var(--radius);
		cursor: pointer; transition: background 0.12s;
	}
	.member-row:hover { background: var(--surface); }
	.member-name { font-weight: 500; flex: 1; }
	.member-host { font-size: 0.8rem; color: var(--text-muted); font-family: monospace; }
</style>
