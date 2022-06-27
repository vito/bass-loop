<script>
  import Time from "svelte-time";
  import Octicon from "./Octicon.svelte";

  export let run = {};

  let {repo, branch, commit} = run.meta?.github || {};
  let check = run.meta?.check;
  let event = run.meta?.event;
</script>

<ul class="summary">
  <li>
    <span class="meta" class:running={!run.completed_at} class:succeeded={run.succeeded} class:failed={!run.succeeded}>
      {#if run.completed_at}
      {#if event}
      <Octicon icon="webhook" />
      {:else}
      <Octicon icon={run.succeeded ? "check-circle-fill" : "x-circle-fill"} />
      {/if}
      {:else}
      <Octicon icon="dot" />
      {/if}
      {#if check}
      <a class="run-id" href="/runs/{run.id}"><strong>{check.name}</strong></a>
      {:else if event}
      <a class="run-id" href="/runs/{run.id}"><strong>{event.name}</strong></a>
      {:else}
      <a class="run-id" href="/runs/{run.id}"><strong>{run.id}</strong></a>
      {/if}
    </span>

    <span class="meta">
      <Octicon icon="person" />
      <a class="subname" href="https://github.com/{run.user.login}">{run.user.login}</a>
    </span>

    <span class="meta">
      <Octicon icon="calendar" />
      <Time live relative timestamp={run.started_at} />
    </span>

    {#if run.completed_at}
    <span class="meta">
      <Octicon icon="stopwatch" />
      {run.duration}
    </span>
    {/if}
  </li>
  <li>
    {#if repo}
    <span class="meta">
      <Octicon icon="repo" />
      <a class="subname" href="{repo.url}">{repo.full_name}</a>
    </span>
    {/if}

    {#if branch}
    <span class="meta">
      <Octicon icon="git-branch" />
      <a class="subname" href="{branch.url}">{branch.name}</a>
    </span>
    {/if}

    {#if commit}
    <span class="meta">
      <Octicon icon="commit" />
      <a class="subname" href="{commit.url}">{commit.sha}</a>
    </span>
    {/if}
  </li>
</ul>

<style>
  .summary {
    font-size: 16px;
    list-style: none;
    padding: 0;
    margin: 0;
    color: var(--base04);
  }

  .meta {
    margin-right: 6px;
  }

  .meta.running :global(.octicon path) {
    fill: var(--running-color) !important;
  }

  .meta.succeeded :global(.octicon path) {
    fill: var(--succeeded-color) !important;
  }

  .meta.failed :global(.octicon path) {
    fill: var(--failed-color) !important;
  }

  .run-id {
    color: var(--base05);
    text-decoration: none;
    word-break: break-all;
  }

  .run-id strong {
    color: var(--base05);
    /* font-family: var(--monospace-font); */
  }

  a {
    text-decoration: none;
  }

  a.subname {
    color: var(--base04);
  }

  a:hover {
    text-decoration: underline;
    color: var(--base05);
  }
</style>
