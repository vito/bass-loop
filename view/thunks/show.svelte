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
  {@html props.thunk.json_html}
  <Footer />
</main>

<style>
  @import "/css/global.css";
</style>
