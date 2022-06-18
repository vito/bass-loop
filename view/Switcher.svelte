<script>
  import Base16Options from './Base16Options.svelte';
</script>

<div class="style-switcher">
  <select id="styleswitcher">
    <Base16Options />
  </select>

  <script type="module">
    import { switchStyle } from "./js/switcher.js";
    document.getElementById("styleswitcher").onchange = switchStyle;
  </script>
</div>
