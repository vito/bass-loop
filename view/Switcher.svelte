<script>
  import { onMount } from 'svelte';

  import * as style from './style';
  import Base16Options from './Base16Options.svelte';

  onMount(() => {
    style.init();
  });
</script>

<select>
  <Base16Options />
</select>
