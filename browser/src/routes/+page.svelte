<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  // `/` exists only to bounce into the dashboard default tab.
  onMount(() => {
    void goto('/records', { replaceState: true });
  });
</script>

<p>Redirecting…</p>
