<script>
  export let thunk = {}
</script>

<h1>Showing Thunk</h1>

<table border="1" cellpadding="10">
  <thead>
    {#each Object.keys(thunk) as key}
      <th>{key}</th>
    {/each}
  </thead>
  <tr>
    {#each Object.keys(thunk) as key}
      <td>{thunk[key]}</td>
    {/each}
  </tr>
</table>

<br />

<a href="/thunk">Back</a>

<style>
  table {
    border-collapse: collapse;
  }
</style>
