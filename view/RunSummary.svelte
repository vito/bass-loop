<script>
  export let run = {};
</script>

<strong>run</strong> <a class="name" href="/runs/{run.id}">{run.id}</a> by <a class="username" href="https://github.com/{run.user.login}">@{run.user.login}</a> {#if run.end_time}{#if run.succeeded.Int64}<span class="succeeded">succeeded</span>{:else}<span class="failed">failed</span>{/if} in {run.Duration}{:else}unfinished{/if}

<style>
  .succeeded {
    color: var(--base0B);
  }

  .failed {
    color: var(--base09);
  }
</style>
