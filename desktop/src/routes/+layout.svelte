<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import type { UnlistenFn } from '@tauri-apps/api/event';
  import {
    getSyncStatus,
    onConnState,
    onRecordsChanged,
    queryRecords
  } from '$lib/tauri';
  import { connState, lastSeq, records, role } from '$lib/stores';
  import type { ConnState, Role } from '$lib/types';

  let unlistenConn: UnlistenFn | null = null;
  let unlistenRecords: UnlistenFn | null = null;

  onMount(async () => {
    try {
      const list = await queryRecords();
      records.set(list);
    } catch (err) {
      console.error('query_records failed', err);
    }

    try {
      const status = await getSyncStatus();
      connState.set(status.conn_state as ConnState);
      role.set(status.role as Role);
      lastSeq.set(status.last_seq);
    } catch (err) {
      console.error('get_sync_status failed', err);
    }

    unlistenConn = await onConnState((s) => {
      connState.set(s as ConnState);
    });

    unlistenRecords = await onRecordsChanged((evt) => {
      records.update((list) => {
        if (evt.op === 'delete') {
          return list.filter((r) => r.id !== evt.record.id);
        }
        const idx = list.findIndex((r) => r.id === evt.record.id);
        if (idx === -1) return [...list, evt.record];
        const next = list.slice();
        next[idx] = evt.record;
        return next;
      });
      lastSeq.update((cur) => Math.max(cur, evt.record.seq));
    });
  });

  onDestroy(() => {
    unlistenConn?.();
    unlistenRecords?.();
  });
</script>

<slot />
