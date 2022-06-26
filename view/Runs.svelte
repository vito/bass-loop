<script>
  import Run from './Run.svelte'

  export let runs = [];
</script>

<ul class="thunk-runs">
  {#if runs.length == 0}
    <li class="none">none</li>
  {/if}
  {#each runs as run}
    <li class="thunk">
      <Run {run} />
    </li>
  {/each}
</ul>

<style>
  .thunk-runs {
    color: var(--base04);
  }

  .thunk-runs {
    list-style-type: none;
    margin: 0;
    padding: 0;
  }

  .thunk-runs li {
    margin-bottom: 10px;

    display: flex;
    flex-direction: column;
  }
</style>
