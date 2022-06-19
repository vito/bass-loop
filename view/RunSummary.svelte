<script>
  export let run = {};
</script>

<ul class="summary">
  <li>
    <strong>run</strong> <a class="name" href="/run/{run.id}">{run.id}</a>
    by <a class="username" href="https://github.com/{run.user.login}">@{run.user.login}</a>
    {#if run.end_time}
      {#if run.succeeded}
        <span class="succeeded">succeeded</span>
      {:else}
        <span class="failed">failed</span>
      {/if}
      in {run.duration}
    {:else}
      unfinished
    {/if}
  </li>
  <li><strong>thunk</strong> <a class="name" href="/thunk/{run.thunk.digest}">{run.thunk.digest}</a></li>
</ul>

<style>
  .summary {
    font-size: 16px;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  a.name {
    font-family: var(--monospace-font);
    color: var(--base05);
    font-weight: bold;
    word-break: break-all;
    text-decoration: none;
  }

  a.username {
    color: var(--base04);
    font-weight: bold;
    text-decoration: none;
  }

  a:hover {
    text-decoration: underline;
  }

  .succeeded {
    color: var(--base0B);
  }

  .failed {
    color: var(--base09);
  }
</style>
