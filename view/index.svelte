<script>
  import Header from './Header.svelte';
  import Footer from './Footer.svelte';

  import Title from './Title.svelte';
  import Runs from './Runs.svelte';

  export let home = {
    runs: [],
  };
</script>

<svelte:head>
  <title>bass loop</title>
</svelte:head>

<main>
  <Header />
  <Title text="Thunk Runs" />
  <Runs runs={home.runs} />
  <Footer />
</main>

<style>
  @import "/css/global.css";
</style>
