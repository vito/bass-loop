<script>
  export let homes = []
</script>

<h1>Home Index</h1>

<table border="1" cellpadding="10">
  {#if homes.length > 0}
    <thead>
      {#each Object.keys(homes[0]) as key}
        <th>{key}</th>
      {/each}
    </thead>
  {/if}
  {#each homes as home}
    <tr>
      {#each Object.keys(home) as key}
        {#if key.toLowerCase() === "id"}
          <td><a href={`/${home[key]}`}>{home[key]}</a></td>
        {:else}
          <td>{home[key]}</td>
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
