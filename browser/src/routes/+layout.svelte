<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { v4 as uuidv4 } from 'uuid';
  import { getDb } from '$lib/db/sqlite';
  import { maxSeq, subscribe as subscribeRecords } from '$lib/db/records';
  import { RelayClient } from '$lib/relay/client';
  import { SyncEngine } from '$lib/sync/engine';
  import {
    connState,
    deviceId as deviceIdStore,
    lastSeq,
    records,
    userId as userIdStore
  } from '$lib/stores';
  import {
    deviceId,
    deviceToken,
    email as storedEmail,
    logout,
    registerDevice,
    sessionToken,
    setDeviceId,
    userId
  } from '$lib/auth';
  import { setEngine } from './engine-context';

  $: onLoginPage = $page.route.id?.startsWith('/login') ?? false;

  let booted = false;
  let bootError = '';

  onMount(() => {
    let relay: RelayClient | null = null;
    let unsubRecords: (() => void) | null = null;
    let unsubState: (() => void) | null = null;

    (async () => {
      try {
        // Login page bypasses the authenticated boot path.
        if (onLoginPage) {
          booted = true;
          return;
        }

        // Stable per-browser device_id.
        let devId = deviceId();
        if (!devId) {
          devId = uuidv4();
          setDeviceId(devId);
        }
        deviceIdStore.set(devId);

        // Must be logged in; otherwise redirect to /login.
        if (!sessionToken() || !userId()) {
          await goto('/login');
          return;
        }

        // Acquire a device token if we don't have one yet (e.g. first boot
        // after signup, or after a logout/login cycle cleared it).
        let devTok = deviceToken();
        if (!devTok) {
          const reg = await registerDevice();
          devTok = reg.token;
        }
        userIdStore.set(userId());

        // DB init + initial last_seq.
        await getDb();
        const seq = await maxSeq();
        lastSeq.set(seq);

        unsubRecords = subscribeRecords((snap) => records.set(snap));

        relay = new RelayClient();
        const engine = new SyncEngine(relay);
        setEngine(engine);

        unsubState = relay.state.subscribe((s) => connState.set(s));
        relay.connect(devTok, seq);

        booted = true;
      } catch (e) {
        bootError = e instanceof Error ? e.message : String(e);
        console.error('boot failed', e);
      }
    })();

    return () => {
      unsubRecords?.();
      unsubState?.();
      relay?.stop();
    };
  });

  async function onSignOut() {
    logout();
    await goto('/login');
  }

  // Nav definitions. Using top tabs; simpler to keep legible without CSS
  // framework. Each entry's `match` drives the active-link highlight.
  const navItems = [
    { href: '/records', label: 'Records' },
    { href: '/tables', label: 'Tables' },
    { href: '/keys', label: 'API Keys' },
    { href: '/billing', label: 'Billing' },
    { href: '/devices', label: 'Devices' }
  ];

  function isActive(pathname: string | undefined, href: string): boolean {
    if (!pathname) return false;
    return pathname === href || pathname.startsWith(href + '/');
  }
</script>

<main>
  {#if onLoginPage}
    <slot />
  {:else if bootError}
    <p style="color: red">Boot error: {bootError}</p>
  {:else if !booted}
    <p>Initializing…</p>
  {:else}
    <header
      style="display:flex;justify-content:space-between;align-items:center;padding:0.5rem 1rem;border-bottom:1px solid #ddd;gap:1rem;flex-wrap:wrap;"
    >
      <strong>appunvs</strong>
      <nav aria-label="primary" style="display:flex;gap:0.75rem;flex:1;">
        {#each navItems as item}
          <a
            href={item.href}
            data-testid={`nav-${item.href.slice(1)}`}
            aria-current={isActive($page.url?.pathname, item.href) ? 'page' : undefined}
            style="text-decoration:none;padding:0.25rem 0.5rem;border-bottom:2px solid {isActive(
              $page.url?.pathname,
              item.href
            )
              ? '#333'
              : 'transparent'};color:#333;"
          >
            {item.label}
          </a>
        {/each}
      </nav>
      <div style="display:flex;gap:0.75rem;align-items:center;">
        <span data-testid="header-email">{storedEmail()}</span>
        <button type="button" on:click={onSignOut} data-testid="sign-out">
          sign out
        </button>
      </div>
    </header>
    <slot />
  {/if}
</main>
