<script>
  import Header from './Header.svelte';
  import Footer from '../Footer.svelte';

  import Run from '../Run.svelte'
  import Vertex from './Vertex.svelte'

  export let run = {}
</script>

<main>
  <Header {run} />

  {#each run.vertexes as vertex}
    <Vertex {vertex} />
  {/each}


  <Footer />
</main>

<style>
  @import "/css/global.css";
</style>
