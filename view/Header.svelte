<script>
  import Logo from "./Logo.svelte";
</script>

<div class="header">
  <div class="logo-and-text">
    <a class="logo" href="/"><Logo /></a>
    <a class="main-name" href="/">bass loop</a>
  </div>
</div>

<style>
  .header {
    display: flex;
    flex-direction: row;
    padding-bottom: 33px;
    border-bottom: 2px solid var(--border-color);
    margin-bottom: 35px;
    justify-content: space-between;
  }

  .logo-and-text {
    display: flex;
    flex-direction: row;
    height: 70px;
  }

  .logo :global(svg) {
    height: 70px;
    width: auto;
  }

  .main-name {
    font-size: 70px;
    font-weight: bold;
    text-transform: lowercase;
    text-decoration: none;
    color: var(--base05);
    margin-left: 25px;
    line-height: 70px;
  }

  .logo, .main-name {
    line-height: 70px;
  }

  .main-name:hover {
    text-decoration: underline;
  }
</style>
