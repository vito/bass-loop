<script>
  import StyleSwitcher from "./StyleSwitcher.svelte";
</script>

<footer>
  <StyleSwitcher />
</footer>

<style>
  footer {
    display: flex;
    flex-direction: row;
    justify-content: flex-end;
    align-items: flex-start;
    margin-top: var(--page-margin);
    height: var(--bottom-height);
    border-top: 2px solid var(--border-color);
    padding-top: 33px;
    color: var(--base03);
  }

  a {
    color: var(--base04);
  }

  a:hover, a:active, a:focus {
    color: var(--base05);
  }
</style>
