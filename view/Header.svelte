<script>
  import Logo from "./Logo.svelte";
</script>

<header>
  <a class="logo" href="/"><Logo /></a>
  <a class="main-name" href="/">bass loop</a>
</header>

<style>
  header {
    box-sizing: content-box;
    display: flex;
    flex-direction: row;
    padding-bottom: 33px;
    border-bottom: 2px solid var(--border-color);
    margin-bottom: 35px;
    height: 70px;
  }

  .logo {
    margin-right: 25px;
  }

  .main-name {
    font-size: 70px;
    font-weight: bold;
    text-transform: lowercase;
    text-decoration: none;
    color: var(--base05);
    line-height: 70px;
  }

  .logo, .main-name {
    line-height: 70px;
  }

  .main-name:hover {
    text-decoration: underline;
  }
</style>
