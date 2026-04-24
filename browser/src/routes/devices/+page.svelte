<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '$lib/api';
  import { ApiError } from '$lib/api';
  import { deviceId } from '$lib/auth';
  import type { MeResponse } from '$lib/pb/http';

  let me: MeResponse | null = null;
  let loading = true;
  let loadError = '';
  let thisDevice = '';

  async function refresh() {
    loading = true;
    loadError = '';
    try {
      me = await api.me();
    } catch (e) {
      loadError = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  function errorMessage(e: unknown): string {
    if (e instanceof ApiError) return `${e.message} (${e.status})`;
    if (e instanceof Error) return e.message;
    return String(e);
  }

  function fmtDate(ms: number): string {
    if (!ms) return '—';
    return new Date(ms).toLocaleString();
  }

  onMount(async () => {
    thisDevice = deviceId();
    await refresh();
  });
</script>

<section style="padding:1rem 1.5rem;">
  <h1>Devices</h1>

  {#if loading}
    <p>Loading…</p>
  {:else if loadError}
    <p role="alert" style="color:red">{loadError}</p>
  {:else if me}
    <p>Signed in as <code>{me.email}</code> (<code>{me.user_id}</code>).</p>
    <p>
      {me.devices.length} device{me.devices.length === 1 ? '' : 's'} registered.
    </p>

    <table
      border="1"
      cellpadding="4"
      style="border-collapse:collapse;width:100%;"
      data-testid="devices-table"
    >
      <thead>
        <tr>
          <th>id</th>
          <th>platform</th>
          <th>created</th>
          <th>last seen</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each me.devices as d (d.id)}
          <tr>
            <td><code>{d.id}</code></td>
            <td>{d.platform}</td>
            <td>{fmtDate(d.created_at)}</td>
            <td>{fmtDate(d.last_seen)}</td>
            <td>
              {#if d.id === thisDevice}
                <small>(this browser)</small>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>
