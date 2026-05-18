@extends('layouts.app')

@section('content')
<div class="mb-6 flex justify-between items-center">
    <div>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Script Templates</h1>
        <p class="text-sm mt-1" style="color: var(--text-muted);">Template konfigurasi per-merk ONU.</p>
    </div>
    <a href="{{ route('script-templates.create') }}" class="inline-flex items-center px-4 py-2.5 text-sm font-semibold rounded-xl text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
        <svg class="-ml-0.5 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
        Add Template
    </a>
</div>

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
    @forelse ($templates as $template)
        <div class="glass-card p-5 group hover:glow-accent transition-all duration-300 relative">
            {{-- Header --}}
            <div class="flex items-center justify-between mb-4">
                <div class="flex items-center">
                    <div class="h-10 w-10 rounded-xl flex items-center justify-center mr-3" style="background: linear-gradient(135deg, rgba(6, 182, 212, 0.15), rgba(6, 182, 212, 0.05));">
                        <svg class="h-5 w-5" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                    </div>
                    <div>
                        <h3 class="text-base font-bold" style="color: var(--text-primary);">{{ $template->merk }}</h3>
                        @if($template->is_default)
                            <span class="inline-flex items-center text-[10px] font-bold uppercase tracking-wider" style="color: #10b981;">
                                <span class="w-1.5 h-1.5 rounded-full mr-1" style="background: #10b981;"></span>
                                Default
                            </span>
                        @endif
                    </div>
                </div>
            </div>

            {{-- Preview --}}
            <div class="rounded-lg p-3 mb-4 text-xs font-mono leading-relaxed overflow-hidden" style="background: var(--bg-primary); color: var(--text-muted); max-height: 80px;">
                {{ Str::limit($template->gpon_onu_script ?? 'No script preview', 120) }}
            </div>

            {{-- Actions --}}
            <div class="flex justify-end space-x-2">
                <a href="{{ route('script-templates.edit', $template) }}" class="inline-flex items-center px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">
                    <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                    Edit
                </a>
                <form action="{{ route('script-templates.destroy', $template) }}" method="POST" class="inline" onsubmit="return confirm('Hapus template ini?');">
                    @csrf @method('DELETE')
                    <button type="submit" class="inline-flex items-center px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200" style="background: rgba(239, 68, 68, 0.08); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.15);" onmouseover="this.style.background='rgba(239,68,68,0.15)'" onmouseout="this.style.background='rgba(239,68,68,0.08)'">
                        <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                        Delete
                    </button>
                </form>
            </div>
        </div>
    @empty
        <div class="col-span-full glass-card p-16 text-center">
            <div class="h-16 w-16 rounded-2xl flex items-center justify-center mx-auto mb-4" style="background: rgba(255,255,255,0.03);">
                <svg class="w-8 h-8" style="color: var(--text-muted); opacity: 0.4;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
            </div>
            <p style="color: var(--text-muted);">Belum ada Script Template.</p>
            <a href="{{ route('script-templates.create') }}" class="mt-4 inline-block font-semibold transition-colors" style="color: var(--accent);">+ Tambah Template Sekarang</a>
        </div>
    @endforelse
</div>
@endsection
