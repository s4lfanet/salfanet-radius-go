@extends('layouts.app')

@section('content')
<div class="mb-8">
    <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Dashboard</h1>
    <p class="text-sm mt-1" style="color: var(--text-muted);">Overview of your OLT and ONU systems.</p>
</div>

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
    <!-- Stat Card: Total OLTs -->
    <div class="glass-card p-6 flex items-center group hover:glow-accent transition-all duration-300">
        <div class="h-12 w-12 rounded-xl flex items-center justify-center mr-4" style="background: linear-gradient(135deg, rgba(6, 182, 212, 0.15), rgba(6, 182, 212, 0.05));">
            <svg class="h-6 w-6" style="color: var(--accent);" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
        </div>
        <div>
            <p class="text-sm font-medium" style="color: var(--text-muted);">Total OLTs</p>
            <p class="text-3xl font-bold" style="color: var(--text-primary);">{{ $totalOlts }}</p>
        </div>
    </div>

    <!-- Stat Card: Total Provisioned ONUs -->
    <div class="glass-card p-6 flex items-center group hover:glow-accent transition-all duration-300">
        <div class="h-12 w-12 rounded-xl flex items-center justify-center mr-4" style="background: linear-gradient(135deg, rgba(139, 92, 246, 0.15), rgba(139, 92, 246, 0.05));">
            <svg class="h-6 w-6" style="color: #8b5cf6;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
        </div>
        <div>
            <p class="text-sm font-medium" style="color: var(--text-muted);">Provisioned ONUs</p>
            <p class="text-3xl font-bold" style="color: var(--text-primary);">{{ $totalOnus }}</p>
        </div>
    </div>

    <!-- Stat Card: System Status -->
    <div class="glass-card p-6 flex items-center group hover:glow-accent transition-all duration-300">
        <div class="h-12 w-12 rounded-xl flex items-center justify-center mr-4" style="background: linear-gradient(135deg, rgba(16, 185, 129, 0.15), rgba(16, 185, 129, 0.05));">
            <svg class="h-6 w-6" style="color: #10b981;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
        </div>
        <div>
            <p class="text-sm font-medium" style="color: var(--text-muted);">System Status</p>
            <p class="text-lg font-bold" style="color: #10b981;">● Online</p>
        </div>
    </div>
</div>

<div class="glass-card overflow-hidden">
    <div class="px-6 py-4 border-b flex justify-between items-center" style="border-color: var(--border-color);">
        <h3 class="text-base font-semibold" style="color: var(--text-primary);">Recently Provisioned ONUs</h3>
        <a href="{{ route('onus.index') }}" class="text-sm font-medium transition-colors" style="color: var(--accent);">View all →</a>
    </div>
    <div>
        @if($recentOnus->count() > 0)
        <table class="min-w-full">
            <thead>
                <tr style="border-bottom: 1px solid var(--border-color);">
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">ONU Name</th>
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Hardware / Port</th>
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">OLT Target</th>
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Provision Date</th>
                </tr>
            </thead>
            <tbody>
                @foreach($recentOnus as $onu)
                <tr class="transition-colors" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                    <td class="px-6 py-4 whitespace-nowrap">
                        <div class="font-medium" style="color: var(--text-primary);">{{ $onu->name }}</div>
                        <div class="text-sm" style="color: var(--text-muted);">{{ $onu->sn }}</div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        <div class="text-sm font-mono px-2 py-0.5 rounded-md inline-block" style="color: var(--accent); background: rgba(6,182,212,0.08);">gpon-olt_{{ $onu->board }}/{{ $onu->slot }}/{{ $onu->port }}:{{ $onu->onu_index }}</div>
                        <div class="text-xs mt-1" style="color: var(--text-muted);">{{ $onu->type }}</div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm" style="color: var(--text-secondary);">
                        {{ $onu->olt->name ?? $onu->olt->ip }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm" style="color: var(--text-muted);">
                        {{ $onu->created_at->diffForHumans() }}
                    </td>
                </tr>
                @endforeach
            </tbody>
        </table>
        @else
        <div class="p-12 text-center">
            <svg class="w-16 h-16 mx-auto mb-4" style="color: var(--text-muted); opacity: 0.3;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            <p style="color: var(--text-muted);">No ONUs have been provisioned yet.</p>
        </div>
        @endif
    </div>
</div>
@endsection
