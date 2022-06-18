<script>
  import RunSummary from './RunSummary.svelte'
  export let run = {};
</script>

<div class="thunk-run">
  <a class="avatar" href="/thunks/{run.thunk.digest}">{@html run.avatar}</a>
  <ul class="summary">
    <li><RunSummary {run} /></li>
    <li><strong>thunk</strong> <a class="name" href="/thunks/{run.thunk.digest}">{run.thunk.digest}</a></li>
  </ul>
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
    font-size: 16px;
    align-items: center;
  }

  .summary {
    list-style: none;
    margin-left: 15px;
    padding: 0;
  }

  .avatar :global(svg) {
    width: 44px;
    height: 44px;
    vertical-align: middle;
  }
</style>
