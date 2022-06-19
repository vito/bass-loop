<script>
  import Header from './Header.svelte';
  import Footer from './Footer.svelte';

  import Title from './Title.svelte';
  import Run from './Run.svelte'

  export let home = {
    runs: [],
  };
</script>

<main>
  <Header />

  <Title text="Thunk Runs" />

  <ul class="thunk-runs">
    {#if home.runs.length == 0}
      <li class="none">none</li>
    {/if}
    {#each home.runs as run}
      <li class="thunk">
        <Run {run} />
      </li>
    {/each}
  </ul>

  <Footer />
</main>

<style>
  @import "css/global.css";

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
