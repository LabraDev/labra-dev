<script lang="ts">
	import { onMount } from 'svelte';
	import { apiGET, type ProfileResponse, type ServiceStatus } from '$lib/api';

	let loading = true;
	let error = '';
	let profile: ProfileResponse | null = null;
	let services: ServiceStatus[] = [];

	onMount(async () => {
		try {
			profile = await apiGET<ProfileResponse>('/v1/profile');
			const system = await apiGET<{ services: ServiceStatus[] }>('/v1/system/services');
			services = system.services;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load dashboard';
		} finally {
			loading = false;
		}
	});
</script>

<section class="page">
	<h1>Dashboard</h1>

	{#if loading}
		<p>Loading dashboard...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else}
		<div class="card">
			<h2>Profile</h2>
			<p>User ID: {profile?.user.id}</p>
			<p>Email: {profile?.user.email ?? 'n/a'}</p>
			<p>Roles: {profile?.principal.roles.join(', ') || 'none'}</p>
			<p>AWS Connections: {profile?.aws_connection_count}</p>
		</div>

		<div class="card">
			<h2>Control-Plane Services</h2>
			<ul>
				{#each services as service}
					<li>
						<strong>{service.name}</strong> ({service.tier}) - {service.status}
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</section>

<style>
	.page {
		max-width: 820px;
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
	}
	ul {
		display: grid;
		gap: 0.5rem;
		padding-left: 1.2rem;
	}
	.error {
		color: #ffb4b4;
	}
</style>
