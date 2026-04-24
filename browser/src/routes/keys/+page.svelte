<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '$lib/api';
  import { ApiError } from '$lib/api';
  import type { APIKeyCreateResponse, APIKeySummary } from '$lib/pb/http';

  let keys: APIKeySummary[] = [];
  let loading = true;
  let loadError = '';

  let newName = '';
  let creating = false;
  let createError = '';

  // Newly-minted key (prefix + secret). Present until the user dismisses.
  let fresh: APIKeyCreateResponse | null = null;

  async function refresh() {
    loading = true;
    loadError = '';
    try {
      keys = await api.listKeys();
    } catch (e) {
      loadError = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  async function onCreate() {
    const name = newName.trim();
    if (!name) return;
    createError = '';
    creating = true;
    try {
      fresh = await api.createKey(name);
      newName = '';
      await refresh();
    } catch (e) {
      createError = errorMessage(e);
    } finally {
      creating = false;
    }
  }

  async function onRevoke(id: string) {
    try {
      await api.revokeKey(id);
      await refresh();
    } catch (e) {
      loadError = errorMessage(e);
    }
  }

  function dismissFresh() {
    fresh = null;
  }

  function errorMessage(e: unknown): string {
    if (e instanceof ApiError) return `${e.status}: ${e.message}`;
    return e instanceof Error ? e.message : String(e);
  }

  function fmtDate(ms: number): string {
    if (!ms) return '—';
    return new Date(ms).toLocaleString();
  }

  function status(k: APIKeySummary): string {
    return k.revoked_at && k.revoked_at > 0 ? 'revoked' : 'active';
  }
</script>

<section style="padding:1rem;max-width:70rem;">
  <h1>API keys</h1>

  {#if fresh}
    <div
      data-testid="fresh-key-banner"
      role="alert"
      style="border:2px solid #f5c518;background:#fff8dc;padding:1rem;border-radius:4px;margin-bottom:1rem;"
    >
      <strong>Copy this key now — we don't store it unhashed.</strong>
      <p>Name: <code>{fresh.name}</code></p>
      <p>Prefix: <code>{fresh.prefix}</code></p>
      <p>Secret: <code data-testid="fresh-secret">{fresh.secret}</code></p>
      <button on:click={dismissFresh} data-testid="dismiss-fresh-key">
        I've copied it
      </button>
    </div>
  {/if}

  <form on:submit|preventDefault={onCreate} style="margin-bottom:1.5rem;">
    <label>
      new key name
      <input
        type="text"
        bind:value={newName}
        placeholder="my-agent"
        data-testid="new-key-name"
      />
    </label>
    <button type="submit" disabled={creating} data-testid="create-key">
      {creating ? '…' : 'create key'}
    </button>
    {#if createError}
      <span role="alert" style="color:red;margin-left:0.5rem;">{createError}</span>
    {/if}
  </form>

  {#if loading}
    <p>Loading…</p>
  {:else if loadError}
    <p role="alert" style="color:red">{loadError}</p>
  {:else if keys.length === 0}
    <p><em>No API keys yet.</em></p>
  {:else}
    <table border="1" cellpadding="4" style="border-collapse:collapse;width:100%;">
      <thead>
        <tr>
          <th>prefix</th>
          <th>name</th>
          <th>created_at</th>
          <th>last_used_at</th>
          <th>status</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each keys as k (k.id)}
          <tr data-testid="key-row" data-key-id={k.id}>
            <td><code>{k.prefix}</code></td>
            <td>{k.name}</td>
            <td>{fmtDate(k.created_at)}</td>
            <td>{fmtDate(k.last_used_at)}</td>
            <td>{status(k)}</td>
            <td>
              {#if status(k) === 'active'}
                <button on:click={() => onRevoke(k.id)} data-testid="revoke-key">
                  revoke
                </button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>
