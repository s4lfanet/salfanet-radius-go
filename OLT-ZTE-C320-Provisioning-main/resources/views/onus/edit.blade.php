@extends('layouts.app')

@section('content')
<div class="mb-6">
    <div class="flex items-center space-x-3">
        <a href="{{ route('onus.index') }}" class="transition-colors" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)'" onmouseout="this.style.color='var(--text-muted)'">
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </a>
        <div>
            <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Setting ONU</h1>
            <p class="text-sm mt-0.5" style="color: var(--text-muted);">Konfigurasi perangkat ONT yang sudah ter-provisioning.</p>
        </div>
    </div>
</div>

<div class="glass-card max-w-2xl">
    <form action="{{ route('onus.update', $onu) }}" method="POST" class="p-6">
        @csrf @method('PUT')

        {{-- Device Info (Read Only) --}}
        <div class="mb-6">
            <label class="block text-sm font-medium mb-2" style="color: var(--text-secondary);">Device Information</label>
            <div class="rounded-xl p-4 flex justify-between items-center" style="background: var(--bg-primary); border: 1px solid var(--border-color);">
                <div class="flex items-center space-x-3">
                    <div class="h-10 w-10 rounded-xl flex items-center justify-center" style="background: rgba(6, 182, 212, 0.12);">
                        <svg class="w-5 h-5" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z"></path></svg>
                    </div>
                    <div>
                        <span class="text-sm font-bold font-mono" style="color: var(--text-primary);">{{ $onu->sn }}</span>
                        <span class="mx-2 text-xs" style="color: var(--border-color);">|</span>
                        <span class="text-sm font-mono" style="color: var(--text-secondary);">{{ $onu->type }}</span>
                    </div>
                </div>
                <span class="text-xs font-mono px-2.5 py-1 rounded-lg font-semibold" style="background: rgba(6,182,212,0.1); color: var(--accent);">
                    gpon-onu_{{ $onu->board }}/{{ $onu->slot }}/{{ $onu->port }}:{{ $onu->onu_index }}
                </span>
            </div>
        </div>

        {{-- Name --}}
        <div class="mb-6">
            <label for="name" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Nama / Keterangan Pelanggan</label>
            <input type="text" name="name" id="name" value="{{ old('name', $onu->name) }}" required class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
            <p class="mt-1.5 text-xs" style="color: var(--text-muted);">Nama akan diubah di OLT menggunakan command <code class="px-1 py-0.5 rounded text-[10px] font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">name [keterangan]</code>.</p>
            @error('name') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
        </div>

        {{-- Actions --}}
        <div class="flex justify-end space-x-3 pt-5" style="border-top: 1px solid var(--border-color);">
            <button type="button" onclick="window.history.back();" class="py-2.5 px-5 rounded-xl text-sm font-medium transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">
                Batal
            </button>
            <button type="submit" class="inline-flex items-center py-2.5 px-5 rounded-xl text-sm font-semibold text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path></svg>
                Simpan Konfigurasi
            </button>
        </div>
    </form>
</div>
@endsection
