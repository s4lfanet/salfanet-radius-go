@extends('layouts.app')

@section('content')
<div class="mb-6 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
    <div>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Unconfigured ONUs</h1>
        <p class="text-sm mt-1" style="color: var(--text-muted);">Deteksi ONU baru yang belum terdaftar di OLT.</p>
    </div>

    <form action="{{ route('onus.unconfigured') }}" method="GET" class="flex w-full sm:w-auto items-center gap-2">
        <select name="olt_id" class="block w-full sm:w-64 px-4 py-2.5 rounded-xl text-sm transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-card); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" onchange="this.form.submit()">
            <option value="" style="background: var(--bg-secondary);">Select OLT</option>
            @foreach($olts as $olt)
                <option value="{{ $olt->id }}" {{ ($selectedOlt && $selectedOlt->id == $olt->id) ? 'selected' : '' }} style="background: var(--bg-secondary);">
                    {{ $olt->name ?: $olt->ip }}
                </option>
            @endforeach
        </select>
        <button type="submit" class="inline-flex items-center px-4 py-2.5 text-sm font-semibold rounded-xl text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105 flex-shrink-0" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
            <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
            Scan
        </button>
    </form>
</div>

@if($error)
    <div class="mb-6 p-4 rounded-xl border animate-fade-in-up" style="background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.2);">
        <div class="flex items-center">
            <svg class="h-5 w-5 mr-3" style="color: #ef4444;" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>
            <p class="text-sm font-medium" style="color: #ef4444;">{{ $error }}</p>
        </div>
    </div>
@endif

@if(!$selectedOlt)
    <div class="glass-card p-12 text-center">
        <div class="h-20 w-20 rounded-2xl flex items-center justify-center mx-auto mb-4" style="background: rgba(6, 182, 212, 0.08);">
            <svg class="w-10 h-10" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
        </div>
        <h3 class="text-lg font-semibold mb-1" style="color: var(--text-primary);">No OLT Selected</h3>
        <p class="text-sm mb-6" style="color: var(--text-muted);">Pilih OLT dari dropdown di atas untuk men-scan ONU yang belum terdaftar.</p>
        @if($olts->isEmpty())
            <a href="{{ route('olts.create') }}" class="inline-flex items-center px-5 py-2.5 text-sm font-semibold rounded-xl text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                + Add Your First OLT
            </a>
        @endif
    </div>
@else
    <div class="glass-card overflow-hidden">
        <div class="px-6 py-4 border-b flex items-center justify-between" style="border-color: var(--border-color);">
            <div class="flex items-center">
                <div class="h-8 w-8 rounded-lg flex items-center justify-center mr-3" style="background: rgba(6, 182, 212, 0.12);">
                    <svg class="w-4 h-4" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"></path></svg>
                </div>
                <div>
                    <span class="text-sm font-semibold" style="color: var(--text-primary);">{{ $selectedOlt->name ?: $selectedOlt->ip }}</span>
                    <span class="text-xs ml-2 px-2 py-0.5 rounded-md font-medium" style="background: rgba(16, 185, 129, 0.12); color: #10b981;">{{ count($unconfigured) }} found</span>
                </div>
            </div>
        </div>
        <table class="min-w-full">
            <thead>
                <tr style="border-bottom: 1px solid var(--border-color);">
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Port</th>
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">ONU Index</th>
                    <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Serial Number</th>
                    <th class="px-6 py-3 text-right text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Actions</th>
                </tr>
            </thead>
            <tbody>
                @forelse ($unconfigured as $onu)
                    <tr class="transition-all duration-150" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                        <td class="px-6 py-4 whitespace-nowrap">
                            <span class="text-sm font-mono font-medium" style="color: var(--text-primary);">{{ $onu['board'] }}/{{ $onu['slot'] }}/{{ $onu['port'] }}</span>
                        </td>
                        <td class="px-6 py-4 whitespace-nowrap">
                            <span class="text-sm font-medium" style="color: var(--text-secondary);">{{ $onu['onu_index'] }}</span>
                        </td>
                        <td class="px-6 py-4 whitespace-nowrap">
                            <span class="text-sm font-mono font-semibold" style="color: var(--accent);">{{ $onu['sn'] }}</span>
                        </td>
                        <td class="px-6 py-4 whitespace-nowrap text-right">
                            <a href="{{ route('onus.create', ['olt_id' => $selectedOlt->id, 'board' => $onu['board'], 'slot' => $onu['slot'], 'port' => $onu['port'], 'onu_index' => $onu['onu_index'], 'sn' => $onu['sn']]) }}" class="inline-flex items-center px-3.5 py-1.5 text-xs font-bold rounded-lg text-white shadow transition-all duration-200 hover:scale-105" style="background: linear-gradient(135deg, #10b981, #059669);">
                                <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
                                Register
                            </a>
                        </td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="4" class="px-6 py-16 text-center">
                            <div class="flex flex-col items-center">
                                <div class="h-16 w-16 rounded-2xl flex items-center justify-center mb-4" style="background: rgba(16, 185, 129, 0.06);">
                                    <svg class="w-8 h-8" style="color: #10b981; opacity: 0.6;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                                </div>
                                <p class="font-medium" style="color: var(--text-secondary);">All Clear!</p>
                                <p class="text-sm mt-1" style="color: var(--text-muted);">Tidak ada ONU yang belum terkonfigurasi.</p>
                            </div>
                        </td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>
@endif
@endsection
