@extends('layouts.app')

@section('content')
<div class="mb-6 flex justify-between items-center">
    <div>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">ACS Profiles</h1>
        <p class="text-sm mt-1" style="color: var(--text-muted);">Manage Auto Configuration Server profiles.</p>
    </div>
    <a href="{{ route('acs-profiles.create') }}" class="inline-flex items-center px-4 py-2.5 text-sm font-semibold rounded-xl text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
        <svg class="-ml-0.5 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
        Add Profile
    </a>
</div>

<div class="glass-card overflow-hidden">
    <table class="min-w-full">
        <thead>
            <tr style="border-bottom: 1px solid var(--border-color);">
                <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Profile Name</th>
                <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">ACS URL</th>
                <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Username</th>
                <th class="px-6 py-3 text-left text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Status</th>
                <th class="px-6 py-3 text-right text-[11px] font-bold uppercase tracking-wider" style="color: var(--text-muted);">Actions</th>
            </tr>
        </thead>
        <tbody>
            @forelse ($profiles as $profile)
                <tr class="transition-all duration-150" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                    <td class="px-6 py-4 whitespace-nowrap">
                        <div class="flex items-center">
                            <div class="h-9 w-9 rounded-xl flex items-center justify-center mr-3" style="background: rgba(139, 92, 246, 0.1);">
                                <svg class="h-4 w-4" style="color: #8b5cf6;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                            </div>
                            <span class="text-sm font-semibold" style="color: var(--text-primary);">{{ $profile->name }}</span>
                        </div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        <span class="text-sm font-mono" style="color: var(--accent);">{{ $profile->url }}</span>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm" style="color: var(--text-secondary);">
                        {{ $profile->username ?? '—' }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        @if($profile->is_default)
                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-bold" style="background: rgba(16, 185, 129, 0.12); color: #10b981;">
                                <span class="w-1.5 h-1.5 rounded-full mr-1.5" style="background: #10b981;"></span>
                                Default
                            </span>
                        @else
                            <span class="text-xs" style="color: var(--text-muted);">—</span>
                        @endif
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-right">
                        <div class="flex justify-end items-center space-x-2">
                            <a href="{{ route('acs-profiles.edit', $profile) }}" class="p-1.5 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)';this.style.background='rgba(6,182,212,0.08)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''" title="Edit">
                                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                            </a>
                            <form action="{{ route('acs-profiles.destroy', $profile) }}" method="POST" class="inline" onsubmit="return confirm('Hapus ACS Profile ini?');">
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
                    <td colspan="5" class="px-6 py-16 text-center">
                        <div class="flex flex-col items-center">
                            <div class="h-16 w-16 rounded-2xl flex items-center justify-center mb-4" style="background: rgba(255,255,255,0.03);">
                                <svg class="w-8 h-8" style="color: var(--text-muted); opacity: 0.4;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                            </div>
                            <p style="color: var(--text-muted);">Belum ada ACS Profile.</p>
                        </div>
                    </td>
                </tr>
            @endforelse
        </tbody>
    </table>
</div>
@endsection
