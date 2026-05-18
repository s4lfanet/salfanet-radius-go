@extends('layouts.app')

@section('content')
<div class="mb-6 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
    <div>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Provisioned ONUs</h1>
        <p class="text-sm mt-1" style="color: var(--text-muted);">{{ $onus->total() }} perangkat terdaftar</p>
    </div>
    <form action="{{ route('onus.index') }}" method="GET" class="w-full sm:w-auto flex items-center">
        <div class="relative w-full sm:w-72">
            <svg class="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4" style="color: var(--text-muted);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
            <input type="text" id="search" name="search" value="{{ request('search') }}" placeholder="Cari Nama, SN, Target..." class="w-full pl-10 pr-4 py-2.5 rounded-xl text-sm transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-card); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
        </div>
    </form>
</div>

<div class="glass-card overflow-hidden">
    <div class="overflow-x-auto">
        <table class="min-w-full">
            <thead>
                <tr style="border-bottom: 1px solid var(--border-color);">
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Location</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Serial / MAC</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Name</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Status</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Signal</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">RX Power</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">OLT Target</th>
                    <th class="px-5 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Last Seen</th>
                    <th class="px-5 py-3 text-right text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Actions</th>
                </tr>
            </thead>
            <tbody>
                @forelse ($onus as $onu)
                    <tr class="transition-all duration-150" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                        {{-- Location --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="text-sm font-mono font-medium" style="color: var(--text-primary);">{{ $onu->board }}/{{ $onu->slot }}/{{ $onu->port }}:{{ $onu->onu_index }}</span>
                        </td>

                        {{-- Serial / MAC --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="text-sm font-mono font-semibold" style="color: var(--accent);">{{ $onu->sn }}</span>
                            <div class="text-xs mt-0.5" style="color: var(--text-muted);">{{ $onu->type }}</div>
                        </td>

                        {{-- Name --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="text-sm font-medium" style="color: var(--text-primary);">{{ $onu->name }}</span>
                        </td>

                        {{-- Status --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-bold status-badge-{{ $onu->id }}" style="background: rgba(16, 185, 129, 0.12); color: #10b981;">
                                <span class="w-1.5 h-1.5 rounded-full mr-1.5" style="background: #10b981;"></span>
                                online
                            </span>
                        </td>

                        {{-- Signal --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="inline-flex items-center text-xs font-semibold signal-badge-{{ $onu->id }}" style="color: var(--text-muted);">
                                <svg class="w-3.5 h-3.5 mr-1" fill="currentColor" viewBox="0 0 20 20"><path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"></path></svg>
                                <span class="signal-text-{{ $onu->id }}">--</span>
                            </span>
                        </td>

                        {{-- RX Power --}}
                        <td class="px-5 py-3 whitespace-nowrap">
                            <span class="text-sm font-mono font-medium power-rx-{{ $onu->id }}" style="color: var(--text-muted);">--</span>
                        </td>

                        {{-- OLT Target --}}
                        <td class="px-5 py-3 whitespace-nowrap text-sm" style="color: var(--text-secondary);">
                            {{ $onu->olt->name ?? $onu->olt->ip }}
                        </td>

                        {{-- Last Seen --}}
                        <td class="px-5 py-3 whitespace-nowrap text-sm" style="color: var(--text-muted);">
                            {{ $onu->updated_at->format('n/j/Y, H:i:s') }}
                        </td>

                        {{-- Actions --}}
                        <td class="px-5 py-3 whitespace-nowrap text-right">
                            <div class="flex justify-end items-center space-x-2">
                                <a href="{{ route('onus.edit', $onu) }}" class="p-1.5 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)';this.style.background='rgba(6,182,212,0.08)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''" title="Settings">
                                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                                </a>
                                <form action="{{ route('onus.destroy', $onu) }}" method="POST" class="inline" onsubmit="return confirm('Hapus ONU ini dari OLT?');">
                                    @csrf @method('DELETE')
                                    <button type="submit" class="p-1.5 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='#f87171';this.style.background='rgba(239,68,68,0.08)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''" title="Delete">
                                        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                    </button>
                                </form>
                            </div>
                        </td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="9" class="px-6 py-16 text-center">
                            <div class="flex flex-col items-center">
                                <div class="h-16 w-16 rounded-2xl flex items-center justify-center mb-4" style="background: rgba(255,255,255,0.03);">
                                    <svg class="w-8 h-8" style="color: var(--text-muted); opacity: 0.4;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path></svg>
                                </div>
                                <p style="color: var(--text-muted);">Tidak ada ONU yang ditemukan.</p>
                            </div>
                        </td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>

    {{-- Pagination --}}
    @if ($onus->hasPages())
        <div class="px-6 py-4 flex items-center justify-between" style="border-top: 1px solid var(--border-color);">
            <p class="text-sm" style="color: var(--text-muted);">Showing {{ $onus->firstItem() }} to {{ $onus->lastItem() }} of {{ $onus->total() }} results</p>
            <div class="flex items-center space-x-1">
                {{-- Previous --}}
                @if ($onus->onFirstPage())
                    <span class="px-2.5 py-1.5 rounded-lg text-sm" style="color: var(--text-muted); opacity: 0.4;">‹</span>
                @else
                    <a href="{{ $onus->previousPageUrl() }}" class="px-2.5 py-1.5 rounded-lg text-sm transition-all duration-200" style="color: var(--text-secondary);" onmouseover="this.style.background='rgba(255,255,255,0.06)'" onmouseout="this.style.background=''">‹</a>
                @endif

                {{-- Page Numbers --}}
                @foreach ($onus->getUrlRange(max(1, $onus->currentPage() - 3), min($onus->lastPage(), $onus->currentPage() + 3)) as $page => $url)
                    @if ($page == $onus->currentPage())
                        <span class="px-3 py-1.5 rounded-lg text-sm font-bold" style="background: var(--accent); color: white;">{{ $page }}</span>
                    @else
                        <a href="{{ $url }}" class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200" style="color: var(--text-secondary);" onmouseover="this.style.background='rgba(255,255,255,0.06)'" onmouseout="this.style.background=''">{{ $page }}</a>
                    @endif
                @endforeach

                {{-- Ellipsis + Last Pages --}}
                @if ($onus->currentPage() + 3 < $onus->lastPage())
                    <span class="px-2 py-1.5 text-sm" style="color: var(--text-muted);">...</span>
                    <a href="{{ $onus->url($onus->lastPage() - 1) }}" class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200" style="color: var(--text-secondary);" onmouseover="this.style.background='rgba(255,255,255,0.06)'" onmouseout="this.style.background=''">{{ $onus->lastPage() - 1 }}</a>
                    <a href="{{ $onus->url($onus->lastPage()) }}" class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200" style="color: var(--text-secondary);" onmouseover="this.style.background='rgba(255,255,255,0.06)'" onmouseout="this.style.background=''">{{ $onus->lastPage() }}</a>
                @endif

                {{-- Next --}}
                @if ($onus->hasMorePages())
                    <a href="{{ $onus->nextPageUrl() }}" class="px-2.5 py-1.5 rounded-lg text-sm transition-all duration-200" style="color: var(--text-secondary);" onmouseover="this.style.background='rgba(255,255,255,0.06)'" onmouseout="this.style.background=''">›</a>
                @else
                    <span class="px-2.5 py-1.5 rounded-lg text-sm" style="color: var(--text-muted); opacity: 0.4;">›</span>
                @endif
            </div>
        </div>
    @endif
</div>

<script>
    const onuIds = @json($onus->pluck('id'));
    const totalOnusCount = {{ $totalDbOnusCount }};

    function getSignalInfo(rxVal) {
        if (isNaN(rxVal)) return { label: 'N/A', color: '#64748b', icon: false };
        if (rxVal >= -15) return { label: 'Excellent', color: '#10b981', icon: true };
        if (rxVal >= -20) return { label: 'Excellent', color: '#10b981', icon: true };
        if (rxVal >= -25) return { label: 'Good', color: '#06b6d4', icon: true };
        if (rxVal >= -28) return { label: 'Fair', color: '#f59e0b', icon: true };
        return { label: 'Weak', color: '#ef4444', icon: true };
    }

    async function checkAllPower() {
        // Set loading state
        onuIds.forEach(id => {
            const rx = document.querySelector(`.power-rx-${id}`);
            const sig = document.querySelector(`.signal-text-${id}`);
            if (rx) rx.innerHTML = '<svg class="animate-spin h-3.5 w-3.5 inline" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>';
            if (sig) sig.textContent = '...';
        });

        // Batch of 5 for speed
        for (let i = 0; i < onuIds.length; i += 5) {
            const batch = onuIds.slice(i, i + 5);
            await Promise.all(batch.map(id => checkPower(id)));
        }
    }

    function checkPower(id) {
        const rxEl = document.querySelector(`.power-rx-${id}`);
        const sigBadge = document.querySelector(`.signal-badge-${id}`);
        const sigText = document.querySelector(`.signal-text-${id}`);

        return fetch(`/onus/${id}/power`)
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    const rxVal = parseFloat(data.rx_onu);
                    const sig = getSignalInfo(rxVal);

                    // RX Power
                    if (rxEl) {
                        rxEl.textContent = data.rx_onu;
                        rxEl.style.color = sig.color;
                    }

                    // Signal
                    if (sigBadge) sigBadge.style.color = sig.color;
                    if (sigText) sigText.textContent = sig.label;
                } else {
                    throw new Error();
                }
            })
            .catch(() => {
                if (rxEl) { rxEl.textContent = 'N/A'; rxEl.style.color = '#64748b'; }
                if (sigText) sigText.textContent = 'N/A';
                if (sigBadge) sigBadge.style.color = '#64748b';
            });
    }

    let refreshInterval = null;
    let countdown = 30;

    window.onload = () => {
        // 1. Background sync first, then check power
        fetch('/onus/sync-background', {
            method: 'POST',
            headers: { 'X-CSRF-TOKEN': '{{ csrf_token() }}', 'Accept': 'application/json' }
        }).then(r => r.json())
          .then(data => {
              if (data.success && data.count > 0) {
                  window.location.reload();
                  return;
              }
              if (onuIds.length > 0) checkAllPower();
              startAutoRefresh();
          })
          .catch(() => {
              if (onuIds.length > 0) checkAllPower();
              startAutoRefresh();
          });
    };

    function startAutoRefresh() {
        // Countdown indicator
        const header = document.querySelector('h1');
        if (header) {
            const badge = document.createElement('span');
            badge.id = 'refresh-badge';
            badge.style.cssText = 'font-size:11px;font-weight:600;padding:3px 10px;border-radius:20px;margin-left:12px;background:rgba(6,182,212,0.12);color:#06b6d4;vertical-align:middle;';
            badge.textContent = '⟳ 30s';
            header.appendChild(badge);
        }

        countdown = 30;
        refreshInterval = setInterval(() => {
            countdown--;
            const badge = document.getElementById('refresh-badge');
            if (badge) {
                if (countdown <= 5) {
                    badge.style.color = '#f59e0b';
                    badge.style.background = 'rgba(245,158,11,0.12)';
                } else {
                    badge.style.color = '#06b6d4';
                    badge.style.background = 'rgba(6,182,212,0.12)';
                }
                badge.textContent = `⟳ ${countdown}s`;
            }

            if (countdown <= 0) {
                countdown = 30;
                if (onuIds.length > 0) checkAllPower();
            }
        }, 1000);
    }
</script>
@endsection
