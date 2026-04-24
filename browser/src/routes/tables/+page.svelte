<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '$lib/api';
  import { ApiError } from '$lib/api';
  import { ColumnType, type SchemaTable } from '$lib/pb/http';

  let tables: SchemaTable[] = [];
  let loading = true;
  let loadError = '';

  let newTableName = '';
  let createError = '';
  let creating = false;

  // Per-table inline "add column" inputs. Keyed by table name so each card
  // has its own form state.
  const addColState = new Map<
    string,
    { name: string; type: string; required: boolean; error: string; busy: boolean }
  >();

  async function refresh() {
    loading = true;
    loadError = '';
    try {
      const resp = await api.listTables();
      tables = resp.tables ?? [];
    } catch (e) {
      loadError = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  async function onCreateTable() {
    const name = newTableName.trim();
    if (!name) return;
    createError = '';
    creating = true;
    try {
      await api.createTable(name);
      newTableName = '';
      await refresh();
    } catch (e) {
      createError = errorMessage(e);
    } finally {
      creating = false;
    }
  }

  async function onDeleteTable(name: string) {
    try {
      await api.deleteTable(name);
      await refresh();
    } catch (e) {
      loadError = errorMessage(e);
    }
  }

  function colState(tableName: string) {
    let s = addColState.get(tableName);
    if (!s) {
      s = { name: '', type: ColumnType.TEXT, required: false, error: '', busy: false };
      addColState.set(tableName, s);
    }
    return s;
  }

  async function onAddColumn(tableName: string) {
    const s = colState(tableName);
    const name = s.name.trim();
    if (!name) return;
    s.error = '';
    s.busy = true;
    try {
      await api.addColumn(tableName, {
        name,
        type: s.type as ColumnType,
        required: s.required
      });
      s.name = '';
      s.required = false;
      s.type = ColumnType.TEXT;
      await refresh();
    } catch (e) {
      s.error = errorMessage(e);
    } finally {
      s.busy = false;
      // force Svelte to re-render the bound state map
      addColState.set(tableName, s);
      tables = tables;
    }
  }

  async function onDeleteColumn(table: string, column: string) {
    try {
      await api.deleteColumn(table, column);
      await refresh();
    } catch (e) {
      loadError = errorMessage(e);
    }
  }

  function errorMessage(e: unknown): string {
    if (e instanceof ApiError) return `${e.status}: ${e.message}`;
    return e instanceof Error ? e.message : String(e);
  }

  function fmtDate(ms: number): string {
    if (!ms) return '—';
    return new Date(ms).toLocaleString();
  }
</script>

<section style="padding:1rem;max-width:70rem;">
  <h1>Tables</h1>

  <form on:submit|preventDefault={onCreateTable} style="margin-bottom:1.5rem;">
    <label>
      new table name
      <input
        type="text"
        bind:value={newTableName}
        placeholder="my_table"
        data-testid="new-table-name"
      />
    </label>
    <button type="submit" disabled={creating} data-testid="create-table">
      {creating ? '…' : 'create table'}
    </button>
    {#if createError}
      <span role="alert" style="color:red;margin-left:0.5rem;">{createError}</span>
    {/if}
  </form>

  {#if loading}
    <p>Loading…</p>
  {:else if loadError}
    <p role="alert" style="color:red">{loadError}</p>
  {:else if tables.length === 0}
    <p><em>No tables yet.</em></p>
  {:else}
    {#each tables as t (t.name)}
      {@const s = colState(t.name)}
      <article
        data-testid="table-card"
        data-table-name={t.name}
        style="border:1px solid #ccc;padding:1rem;margin-bottom:1rem;border-radius:4px;"
      >
        <header style="display:flex;justify-content:space-between;align-items:baseline;">
          <h2 style="margin:0">{t.name}</h2>
          <button on:click={() => onDeleteTable(t.name)} data-testid="delete-table">
            delete table
          </button>
        </header>
        <p style="color:#666;font-size:0.85rem;">created {fmtDate(t.created_at)}</p>

        <table border="1" cellpadding="4" style="width:100%;border-collapse:collapse;">
          <thead>
            <tr><th>column</th><th>type</th><th>required</th><th></th></tr>
          </thead>
          <tbody>
            {#each t.columns ?? [] as c (c.name)}
              <tr>
                <td><code>{c.name}</code></td>
                <td>{c.type}</td>
                <td>{c.required ? 'yes' : 'no'}</td>
                <td>
                  <button on:click={() => onDeleteColumn(t.name, c.name)}>
                    delete
                  </button>
                </td>
              </tr>
            {:else}
              <tr><td colspan="4"><em>no columns</em></td></tr>
            {/each}
          </tbody>
        </table>

        <form
          on:submit|preventDefault={() => onAddColumn(t.name)}
          style="margin-top:0.5rem;display:flex;gap:0.5rem;align-items:center;flex-wrap:wrap;"
        >
          <input
            type="text"
            bind:value={s.name}
            on:input={() => (tables = tables)}
            placeholder="column name"
          />
          <select bind:value={s.type} on:change={() => (tables = tables)}>
            <option value={ColumnType.TEXT}>text</option>
            <option value={ColumnType.NUMBER}>number</option>
            <option value={ColumnType.BOOL}>bool</option>
            <option value={ColumnType.JSON}>json</option>
          </select>
          <label>
            <input
              type="checkbox"
              bind:checked={s.required}
              on:change={() => (tables = tables)}
            />
            required
          </label>
          <button type="submit" disabled={s.busy}>
            {s.busy ? '…' : 'add column'}
          </button>
          {#if s.error}
            <span role="alert" style="color:red">{s.error}</span>
          {/if}
        </form>
      </article>
    {/each}
  {/if}
</section>
