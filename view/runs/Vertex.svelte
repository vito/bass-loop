<script>
  export let vertex = {}
</script>

<div class="vertex" class:cached={vertex.cached} class:error={vertex.error}>
  <div class="vertex-info">
    <div class="vertex-name"><code>{vertex.name}</code></div>
    {#if vertex.cached}
    <div class="vertex-status"><code>CACHED</code></div>
    {:else}
    <div class="vertex-duration"><code>[{vertex.duration}]</code></div>
    {/if}
  </div>
  {#if vertex.lines.length > 0 || vertex.error}
  <table class="vertex-logs">
    {#each vertex.lines as line, lnum}
      <tr>
        <td id="V{vertex.num}L{lnum+1}" class="number">
          <a href="#V{vertex.num}L{lnum+1}" data-line-number="{lnum+1}"></a>
        </td>
        <td class="content">{@html line.content}</td>
      </tr>
    {/each}
    {#if vertex.error}
      <tr>
        <td id="V{vertex.num}ERR" class="number">
          <a href="#V{vertex.num}ERR" data-line-number="ERR"></a>
        </td>
        <td class="content">
          <span class="ansi-line"><span class="fg-red">{vertex.error}</span></span>
        </td>
      </tr>
    {/if}
  </table>
  {/if}
</div>

<svelte:head>
  <link rel="stylesheet" href="/css/ansi.css" />
</svelte:head>

<style>
  .vertex {
    line-height: 20px;
    font-size: 16px;
  }

  .vertex-info {
    display: flex;
    flex-direction: row;
    color: var(--base0B);
  }

  .vertex-info:before {
    content: "=> ";
    white-space: pre;
    font-family: var(--monospace-font);
  }

  .vertex-name {
    margin-right: 1ch;
  }

  .vertex.cached .vertex-info {
    color: var(--base03);
  }

  .vertex.error .vertex-info {
    color: var(--base08);
  }

  .vertex-logs {
    margin: 0;
    font-size: 16px;
    white-space: pre-wrap;
    border-collapse: collapse;
    border-spacing: 0;
  }

  .vertex-logs tr {
    position: relative;
  }

  .vertex-logs td.number {
    width: 100px;
    text-align: right;
    position: absolute;
    left: -100px;
  }

  .vertex-logs a {
    text-decoration: none;
    color: var(--base04);
  }

  .vertex-logs a:before { /* prevent text being selectable */
    content: attr(data-line-number);
    font-family: var(--monospace-font);
    text-align: right;
  }

  .vertex-logs a:hover {
    cursor: pointer;
    color: var(--base05);
  }

  .vertex-logs td.number:target {
    background: var(--base01);
  }

  .vertex-logs td.number:target a {
    color: var(--base0A);
  }

  td.content {
    font-family: var(--monospace-font);
    white-space: pre;
    margin: 0;
    padding: 0;
    border: 0;
    tab-size: 8;
  }

  td.content:empty {
    content: "x";
  }

  td.content :global(.ansi-line:before) {
    font-family: var(--monospace-font);
    content: "  │ ";
    color: var(--base03);
  }
</style>
