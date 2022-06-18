<script>
  import Header from './Header.svelte';
  import Footer from './Footer.svelte';
  import { style } from './DefaultStyle.js';

  import Run from './Run.svelte'

  export let home = {
    runs: [],
  };
</script>

<main>
  <Header />

  <h2><span class="header-text">thunk runs</span></h2>

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

  main {
    display: flex;
    flex-direction: column;
    justify-content: center;
    justify-self: stretch;
    font-weight: 400;
    line-height: 1.6;
    padding: var(--page-margin);
  }

  h2 {
    font-family: var(--monospace-font);
    margin-top: 0;
  }

  .header-text {
    background: var(--highlight-gradient);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }

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
