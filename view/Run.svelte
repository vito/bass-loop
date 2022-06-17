<script>
  import RunSummary from './RunSummary.svelte'
  export let run = {};
</script>

<span class="logo-and-text">
  <a href="/thunks/{run.thunk.digest}">{@html run.avatar}</a>
  <div class="heading-and-name">
    <ul>
      <li><RunSummary {run} /></li>
      <li><strong>thunk</strong> <a class="name" href="/thunks/{run.thunk.digest}">{run.thunk.digest}</a></li>
    </ul>
  </div>
</span>
