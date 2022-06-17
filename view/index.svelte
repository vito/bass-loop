<script>
  import Switcher from './Switcher.svelte';

  import Run from './Run.svelte'
  export let home = {
    runs: [],
  };
</script>

<h2>thunk runs</h2>

<ul class="thunks">
  {#if home.runs.length == 0}
    <li class="none">none</li>
  {/if}
  {#each home.runs as run}
    <li class="thunk">
      <Run {run} />
    </li>
  {/each}
</ul>

<Switcher />

<style>
  table {
    border-collapse: collapse;
  }
</style>
