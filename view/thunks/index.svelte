<script>
  export let thunks = []
</script>

<h1>Thunk Index</h1>

<table border="1" cellpadding="10">
  {#if thunks.length > 0}
    <thead>
      {#each Object.keys(thunks[0]) as key}
        <th>{key}</th>
      {/each}
    </thead>
  {/if}
  {#each thunks as thunk}
    <tr>
      {#each Object.keys(thunk) as key}
        {#if key.toLowerCase() === "id"}
          <td><a href={`/thunks/${thunk[key]}`}>{thunk[key]}</a></td>
        {:else}
          <td>{thunk[key]}</td>
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
