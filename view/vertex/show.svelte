<script>
  export let vertex = {}
</script>

<h1>Showing Vertex</h1>

<table border="1" cellpadding="10">
  <thead>
    {#each Object.keys(vertex) as key}
      <th>{key}</th>
    {/each}
  </thead>
  <tr>
    {#each Object.keys(vertex) as key}
      <td>{vertex[key]}</td>
    {/each}
  </tr>
</table>

<br />

<a href="/vertex">Back</a>

<style>
  table {
    border-collapse: collapse;
  }
</style>
