<script>
  import RunHeader from './RunHeader.svelte';
  import Footer from '../Footer.svelte';

  import Run from '../Run.svelte'
  import Vertex from './Vertex.svelte'

  export let props = {
    run: {},
    vertexes: [],
  }

  export let run = props.run;
  export let vertexes = props.vertexes;
</script>

<svelte:head>
  <title>run {run.id} ; bass loop</title>
</svelte:head>

<main>
  <RunHeader {run} />

  {#each vertexes as vertex}
    <Vertex {vertex} />
  {/each}


  <Footer />
</main>

<style>
  @import "/css/global.css";
</style>
