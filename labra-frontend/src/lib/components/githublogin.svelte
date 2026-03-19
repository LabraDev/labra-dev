<script>
    import github_logo from '$lib/assets/github-logo.svg';
</script>

<a href="http://localhost:8080/login" id="logintxt">
    <img src={github_logo} alt="github logo" height="40" width="40" />
    ‎ Login with Github
</a>

<style>
    #logintxt {
        display: flex;
        justify-content: center;
        align-items: center;
        padding: 1vh;
        border: 1px solid var(--border-color);
        border-radius: 1vh;
        margin: 1vh;
        font-size: 20px;
        background-color: var(--crust);
    }
    #logintxt:link, #logintxt:hover, #logintxt:visited, #logintxt:active {
        text-decoration: none;
        color: var(--text-color);
    }
</style>