<script lang="ts">
	import { createAuthSession } from '$lib/api';

	let externalJWT = '';
	let loading = false;
	let error = '';
	let success = '';

	async function handleLogin() {
		error = '';
		success = '';
		loading = true;
		try {
			const session = await createAuthSession(externalJWT.trim());
			success = `Signed in as ${session.principal.email ?? session.principal.sub}`;
			setTimeout(() => {
				window.location.href = '/dashboard';
			}, 350);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<section class="page">
	<h1>Login</h1>
	<p>Paste your Cognito JWT for Sprint 2 auth bootstrap.</p>

	<label for="jwt">Cognito JWT</label>
	<textarea id="jwt" bind:value={externalJWT} rows="8" placeholder="eyJhbGciOi..."></textarea>
	<button on:click={handleLogin} disabled={loading || externalJWT.trim().length === 0}>
		{loading ? 'Signing in...' : 'Create Session'}
	</button>

	{#if error}<p class="error">{error}</p>{/if}
	{#if success}<p class="success">{success}</p>{/if}
</section>

<style>
	.page {
		max-width: 760px;
		margin: 0 auto;
		padding: 2rem 1rem;
		display: grid;
		gap: 0.8rem;
	}

	textarea {
		width: 100%;
		border-radius: 10px;
		padding: 0.7rem;
		background: #111523;
		color: #d9e1ff;
		border: 1px solid #3a4058;
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

	.error { color: #ffb4b4; }
	.success { color: #98e5b0; }
</style>
