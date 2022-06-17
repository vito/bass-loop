<script>
  export let runs = []
</script>

<h1>Run Index</h1>

<table border="1" cellpadding="10">
  {#if runs.length > 0}
    <thead>
      {#each Object.keys(runs[0]) as key}
        <th>{key}</th>
      {/each}
    </thead>
  {/if}
  {#each runs as run}
    <tr>
      {#each Object.keys(run) as key}
        {#if key.toLowerCase() === "id"}
          <td><a href={`/run/${run[key]}`}>{run[key]}</a></td>
        {:else}
          <td>{run[key]}</td>
        {/if}
      {/each}
    </tr>
  {/each}
</table>

<style>
  table {
    border-collapse: collapse;
  }
</style>
