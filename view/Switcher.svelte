<script>
  import Base16Options from './Base16Options.svelte';
</script>

<div>
  <select id="styleswitcher">
    <Base16Options />
  </select>

  <script type="module">
    import { switchStyle } from "./js/switcher.js";
    document.getElementById("styleswitcher").onchange = switchStyle;
  </script>
</div>
