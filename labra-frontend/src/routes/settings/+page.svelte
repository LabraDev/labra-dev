<script lang="ts">
	import { onMount } from 'svelte';
	import { apiGET, apiPOST } from '$lib/api';

	type AWSConnection = {
		id: number;
		role_arn: string;
		external_id: string;
		region: string;
		account_id: string;
		status: string;
		updated_at: number;
	};

	let roleARN = '';
	let externalID = '';
	let region = 'us-west-2';
	let loading = false;
	let error = '';
	let success = '';
	let connections: AWSConnection[] = [];

	async function refreshConnections() {
		const result = await apiGET<{ aws_connections: AWSConnection[] }>('/v1/aws-connections');
		connections = result.aws_connections;
	}

	async function connectAWS() {
		error = '';
		success = '';
		loading = true;
		try {
			await apiPOST('/v1/aws-connections', {
				role_arn: roleARN,
				external_id: externalID,
				region
			});
			success = 'AWS connection validated and saved.';
			await refreshConnections();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to connect AWS';
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		try {
			await refreshConnections();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load AWS connections';
		}
	});
</script>

<section class="page">
	<h1>Settings</h1>
	<p>Connect your AWS account using AssumeRole metadata.</p>

	<div class="card">
		<h2>Connect AWS</h2>
		<label for="role-arn">Role ARN</label>
		<input id="role-arn" bind:value={roleARN} placeholder="arn:aws:iam::123456789012:role/labra-access" />
		<label for="external-id">External ID</label>
		<input id="external-id" bind:value={externalID} placeholder="external-id-123" />
		<label for="region">Region</label>
		<input id="region" bind:value={region} placeholder="us-west-2" />
		<button on:click={connectAWS} disabled={loading}> {loading ? 'Saving...' : 'Validate + Save'} </button>
	</div>

	{#if error}<p class="error">{error}</p>{/if}
	{#if success}<p class="success">{success}</p>{/if}

	<div class="card">
		<h2>Existing Connections</h2>
		{#if connections.length === 0}
			<p>No connections yet.</p>
		{:else}
			<ul>
				{#each connections as c}
					<li>
						<strong>{c.account_id}</strong> - {c.region} - {c.status}
						<div class="small">{c.role_arn}</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</section>

<style>
	.page {
		max-width: 840px;
		margin: 0 auto;
		padding: 2rem 1rem;
		display: grid;
		gap: 1rem;
	}
	.card {
		border: 1px solid #32384f;
		background: #1a1f33;
		border-radius: 12px;
		padding: 1rem;
		display: grid;
		gap: 0.5rem;
	}
	input {
		background: #101423;
		border: 1px solid #3a4058;
		border-radius: 8px;
		color: #d9e1ff;
		padding: 0.5rem;
	}
	button {
		width: fit-content;
		background: #c9d2ff;
		color: #11111b;
		border: none;
		border-radius: 10px;
		padding: 0.6rem 1rem;
		font-weight: 600;
		cursor: pointer;
	}
	.small {
		font-size: 0.82rem;
		color: #b3bddf;
	}
	.error {
		color: #ffb4b4;
	}
	.success {
		color: #98e5b0;
	}
	ul {
		display: grid;
		gap: 0.6rem;
		padding-left: 1.1rem;
	}
</style>
