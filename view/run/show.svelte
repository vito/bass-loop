<script>
  export let run = {}
</script>

<h1>Showing Run</h1>

<table border="1" cellpadding="10">
  <thead>
    {#each Object.keys(run) as key}
      <th>{key}</th>
    {/each}
  </thead>
  <tr>
    {#each Object.keys(run) as key}
      <td>{run[key]}</td>
    {/each}
  </tr>
</table>

<br />

<a href="/run">Back</a>

<style>
  table {
    border-collapse: collapse;
  }
</style>
