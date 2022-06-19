<script>
  import Logo from "../Logo.svelte";

  import Run from '../Run.svelte'
  import RunSummary from '../RunSummary.svelte'

  export let run = {};
</script>

<header>
  <a class="logo" href="/"><Logo /></a>
  <div class="run-info">
    <span class="title">run log</span>
    <RunSummary {run} />
  </div>
  <a class="avatar" href="/thunk/{run.thunk.digest}">{@html run.thunk.avatar}</a>
</header>

<style>
  header {
    box-sizing: content-box;
    display: flex;
    flex-direction: row;
    padding-bottom: 33px;
    border-bottom: 2px solid var(--border-color);
    margin-bottom: 35px;
    align-items: center;
  }

  .logo {
    margin-right: 25px;
  }

  .avatar {
    margin-left: auto;
  }

  .title {
    font-size: 32px;
    line-height: 1.6;
    font-weight: bold;
    text-transform: lowercase;
    text-decoration: none;
    color: var(--base05);
  }
</style>
