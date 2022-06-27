<script>
  import Header from '../Header.svelte';
  import Footer from '../Footer.svelte';

  import Title from '../Title.svelte';
  import Runs from '../Runs.svelte';

  export let props = {
    thunk: {},
    runs: [],
  };

  export let thunk = props.thunk;
  export let runs = props.runs;
</script>

<svelte:head>
  <title>thunk {thunk.digest} ; bass loop</title>
</svelte:head>

<main>
  <Header />
  <Title text="Runs" />
  <Runs runs={runs} />
  <Title text="JSON" />
  <div class="highlight wrap" contenteditable="true">
    {@html props.json_html}
  </div>
  <Footer />
</main>

<style>
  @import "/css/global.css";
  @import "/css/highlight.css";

  .highlight.wrap :global(pre) {
    word-break: break-all;
    white-space: pre-wrap;
  }
</style>
