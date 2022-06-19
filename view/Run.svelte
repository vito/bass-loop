<script>
  import RunSummary from './RunSummary.svelte'
  export let run = {};
</script>

<div class="thunk-run">
  <a class="avatar" href="/thunk/{run.thunk.digest}">{@html run.avatar}</a>
  <RunSummary {run} />
</div>

<style>
  a.name {
    font-family: var(--monospace-font);
    color: var(--base05);
    font-weight: bold;
    word-break: break-all;
    text-decoration: none;
  }

  a:hover {
    text-decoration: underline;
  }

  .thunk-run {
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  .avatar {
    margin-right: 15px;
  }

  .avatar :global(svg) {
    width: 44px;
    height: 44px;
    vertical-align: middle;
  }
</style>
