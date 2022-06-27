<script>
  import Octicons from '@primer/octicons';

  export let icon;

  let octicon = Octicons[icon];
  if (!octicon) {
    throw "unknown icon: " + icon;
  }
</script>

{@html octicon.toSVG({ width: 16 })}

<style>
  :global(.octicon) {
    vertical-align: text-bottom;
  }

  :global(.octicon) :global(path) {
    fill: currentColor;
  }
</style>
