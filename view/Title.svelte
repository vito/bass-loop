<script>
  export let text = "";
</script>

<h1><span class="header-text">{text}</span></h1>

<style>
  h1 {
    font-family: var(--monospace-font);
    margin-top: 0;
    text-transform: lowercase;
  }

  .header-text::before {
    content: ";; ";
  }

  .header-text {
    background: var(--highlight-gradient);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }
</style>
