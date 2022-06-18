<script>
  import Header from './Header.svelte';
  import Footer from './Footer.svelte';
  import { style } from './DefaultStyle.js';

  import Run from './Run.svelte'

  export let home = {
    runs: [],
  };
</script>

<main class="page">
  {@html `<style id="default-theme">:root {${style.vars.join(";")}}</style>`}

  <Header />

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

  <Footer />
</main>

<style>
  @import "css/global.css";

  main {
    display: flex;
    flex-direction: column;
    justify-content: center;
    justify-self: stretch;
    font-weight: 400;
    line-height: 1.6;
    padding: var(--page-margin);
  }
</style>
