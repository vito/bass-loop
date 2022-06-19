<script>
  import RunHeader from './RunHeader.svelte';
  import Footer from '../Footer.svelte';

  import Run from '../Run.svelte'
  import Vertex from './Vertex.svelte'

  export let run = {}
</script>

<svelte:head>
  <title>run {run.id} ; bass loop</title>
</svelte:head>

<main>
  <RunHeader {run} />

  {#each run.vertexes as vertex}
    <Vertex {vertex} />
  {/each}


  <Footer />
</main>

<style>
  @import "/css/global.css";
</style>
