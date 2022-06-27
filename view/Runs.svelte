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
    list-style-type: none;
    margin: 0;
    padding: 0;
  }

  .thunk-runs li {
    display: flex;
    flex-direction: column;
    margin-bottom: 22px;
  }
</style>
