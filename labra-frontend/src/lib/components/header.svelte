<script>
    import GithubLoginButton from '$lib/components/githublogin.svelte';
</script>

<div class="header">
    <h1 id="logo">Labra</h1>
    <div class="login">
        <GithubLoginButton />
    </div>
</div>
<hr>
<slot />

<style>
    .header {
        background-color: var(--crust);
        min-height: 7vh;
        color: var(--text-color);
        width: 100%;
        display: flex;
        padding-top: 2vh;
        padding-bottom: 2vh;
        justify-content: space-between;
        align-items: center;
    }
    #logo {
        margin-left: 3vh;
        font-size: 5vh;
    }
    .login {
        margin-left: auto;
        margin-right: 3vh;
        display: flex;
        justify-content: center;
    }
    hr {
        height: 0.2vh;
        border-width: 0;
        color: var(--hr-color);
        background-color: var(--hr-color);
    }
</style>