<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '$lib/api';
  import { ApiError } from '$lib/api';
  import type { BillingStatusResponse, Plan } from '$lib/pb/http';

  let status: BillingStatusResponse | null = null;
  let plans: Plan[] = [];
  let loading = true;
  let loadError = '';

  let busyPlanId = '';
  let checkoutError = '';
  let lastMode = '';

  async function refresh() {
    loading = true;
    loadError = '';
    try {
      const [s, p] = await Promise.all([api.billingStatus(), api.listPlans()]);
      status = s;
      plans = p.plans;
    } catch (e) {
      loadError = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  async function onUpgrade(planId: string) {
    checkoutError = '';
    busyPlanId = planId;
    try {
      const resp = await api.billingCheckout(planId);
      lastMode = resp.mode;
      if (resp.mode === 'mock') {
        // Mock mode: relay has no real Stripe. Show the URL so the dev
        // flow has something visible; in real mode we'd just redirect.
        window.alert(
          `Stripe is not configured on this relay. Mock checkout URL:\n\n${resp.url}\n\n` +
            `In live mode this browser would redirect to Stripe Checkout.`
        );
      } else {
        window.location.href = resp.url;
      }
    } catch (e) {
      checkoutError = errorMessage(e);
    } finally {
      busyPlanId = '';
    }
  }

  function errorMessage(e: unknown): string {
    if (e instanceof ApiError) return `${e.message} (${e.status})`;
    if (e instanceof Error) return e.message;
    return String(e);
  }

  function fmtBytes(n: number): string {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
    if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MiB`;
    return `${(n / 1024 / 1024 / 1024).toFixed(2)} GiB`;
  }

  function fmtDate(ms: number): string {
    if (!ms) return '—';
    return new Date(ms).toLocaleString();
  }

  function pct(used: number, limit: number): string {
    if (limit <= 0) return '0%';
    const p = Math.min(100, Math.round((used / limit) * 100));
    return `${p}%`;
  }

  onMount(refresh);
</script>

<section style="padding:1rem 1.5rem;">
  <h1>Billing</h1>

  {#if loading}
    <p>Loading…</p>
  {:else if loadError}
    <p role="alert" style="color:red">{loadError}</p>
  {:else if status}
    <article
      style="border:1px solid #ddd;border-radius:6px;padding:1rem;margin-bottom:1.5rem;"
      data-testid="status-card"
    >
      <h2 style="margin-top:0;">
        Current plan: <code>{status.plan}</code>
        {#if status.plan_name}<small>({status.plan_name})</small>{/if}
      </h2>
      <p>
        Status: <strong>{status.status}</strong> · renews {fmtDate(status.period_end)}
      </p>
      <dl style="display:grid;grid-template-columns:auto 1fr auto;gap:0.25rem 1rem;">
        <dt>messages / day</dt>
        <dd>
          {status.messages_used} / {status.limits.messages_per_day}
          <span style="color:#666;">({pct(status.messages_used, status.limits.messages_per_day)})</span>
        </dd>
        <dd></dd>

        <dt>bytes / period</dt>
        <dd>
          {fmtBytes(status.storage_bytes)} / {fmtBytes(status.limits.storage_bytes)}
          <span style="color:#666;">({pct(status.storage_bytes, status.limits.storage_bytes)})</span>
        </dd>
        <dd></dd>

        <dt>devices</dt>
        <dd>up to {status.limits.max_devices}</dd>
        <dd></dd>

        <dt>api keys</dt>
        <dd>up to {status.limits.max_api_keys}</dd>
        <dd></dd>
      </dl>
    </article>

    <h2>Plans</h2>
    {#if checkoutError}
      <p role="alert" style="color:red">{checkoutError}</p>
    {/if}
    <div
      style="display:grid;grid-template-columns:repeat(auto-fit,minmax(14rem,1fr));gap:1rem;"
      data-testid="plans-grid"
    >
      {#each plans as plan (plan.id)}
        {@const current = status && plan.id === status.plan}
        <article
          style="border:1px solid {current ? '#333' : '#ddd'};border-radius:6px;padding:1rem;"
          data-testid={`plan-${plan.id}`}
        >
          <h3 style="margin-top:0;">{plan.name}</h3>
          <p style="font-size:1.25rem;margin:0.25rem 0;">
            {#if plan.price_cents_monthly === 0}
              Free
            {:else}
              ${(plan.price_cents_monthly / 100).toFixed(0)} / mo
            {/if}
          </p>
          <ul style="padding-left:1.25rem;line-height:1.6;">
            <li>{plan.messages_per_day.toLocaleString()} messages / day</li>
            <li>{fmtBytes(plan.storage_bytes)} bytes / period</li>
            <li>{plan.max_devices} devices</li>
            <li>{plan.max_api_keys} api keys</li>
          </ul>
          {#if current}
            <p><strong>Current</strong></p>
          {:else}
            <button
              type="button"
              disabled={busyPlanId === plan.id}
              on:click={() => onUpgrade(plan.id)}
              data-testid={`upgrade-${plan.id}`}
            >
              {busyPlanId === plan.id ? '…' : 'Upgrade'}
            </button>
          {/if}
        </article>
      {/each}
    </div>

    {#if lastMode === 'mock'}
      <p style="color:#666;margin-top:1rem;">
        <small>
          Stripe is not configured on this relay; checkout runs in mock mode.
          Set <code>APPUNVS_BILLING_STRIPE_SECRET_KEY</code> on the relay to
          enable live payments.
        </small>
      </p>
    {/if}
  {/if}
</section>
