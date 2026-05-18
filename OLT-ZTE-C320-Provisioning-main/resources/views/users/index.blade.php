@extends('layouts.app')

@section('title', 'User Management')

@section('content')
<div class="animate-fade-in-up">
    <div class="flex items-center justify-between mb-6">
        <div>
            <h1 class="text-2xl font-bold" style="color: var(--text-primary);">User Management</h1>
            <p class="text-sm mt-1" style="color: var(--text-muted);">{{ $users->total() }} users registered</p>
        </div>
        <a href="{{ route('users.create') }}" class="inline-flex items-center px-4 py-2.5 rounded-xl text-sm font-semibold text-white transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
            <svg class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" /></svg>
            Add User
        </a>
    </div>

    @if(session('success'))
    <div class="mb-4 p-4 rounded-xl text-sm font-medium" style="background: rgba(16, 185, 129, 0.1); border: 1px solid rgba(16, 185, 129, 0.2); color: #10b981;">
        ✓ {{ session('success') }}
    </div>
    @endif

    @if(session('error'))
    <div class="mb-4 p-4 rounded-xl text-sm font-medium" style="background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.2); color: #ef4444;">
        ✕ {{ session('error') }}
    </div>
    @endif

    <div class="glass-card overflow-hidden">
        <table class="w-full text-left">
            <thead>
                <tr style="border-bottom: 1px solid var(--border-color);">
                    <th class="px-6 py-4 text-xs font-bold uppercase tracking-wider" style="color: var(--accent);">User</th>
                    <th class="px-6 py-4 text-xs font-bold uppercase tracking-wider" style="color: var(--accent);">Email</th>
                    <th class="px-6 py-4 text-xs font-bold uppercase tracking-wider" style="color: var(--accent);">Created</th>
                    <th class="px-6 py-4 text-xs font-bold uppercase tracking-wider text-right" style="color: var(--accent);">Actions</th>
                </tr>
            </thead>
            <tbody>
                @forelse($users as $user)
                <tr class="transition-colors duration-150" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                    <td class="px-6 py-4">
                        <div class="flex items-center">
                            <div class="h-9 w-9 rounded-xl flex items-center justify-center text-white font-bold text-xs uppercase" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                                {{ substr($user->name, 0, 2) }}
                            </div>
                            <div class="ml-3">
                                <p class="text-sm font-semibold" style="color: var(--text-primary);">{{ $user->name }}</p>
                                @if($user->id === auth()->id())
                                <span class="text-[10px] px-2 py-0.5 rounded-full font-bold" style="background: rgba(6, 182, 212, 0.15); color: var(--accent);">YOU</span>
                                @endif
                            </div>
                        </div>
                    </td>
                    <td class="px-6 py-4">
                        <span class="text-sm font-mono" style="color: var(--text-secondary);">{{ $user->email }}</span>
                    </td>
                    <td class="px-6 py-4">
                        <span class="text-sm" style="color: var(--text-muted);">{{ $user->created_at->format('d/m/Y H:i') }}</span>
                    </td>
                    <td class="px-6 py-4 text-right">
                        <div class="flex items-center justify-end space-x-2">
                            <a href="{{ route('users.edit', $user) }}" class="p-2 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)';this.style.background='var(--accent-glow)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''">
                                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
                            </a>
                            @if($user->id !== auth()->id())
                            <form action="{{ route('users.destroy', $user) }}" method="POST" onsubmit="return confirm('Delete user {{ $user->name }}?')">
                                @csrf @method('DELETE')
                                <button type="submit" class="p-2 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='#ef4444';this.style.background='rgba(239,68,68,0.1)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''">
                                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                                </button>
                            </form>
                            @endif
                        </div>
                    </td>
                </tr>
                @empty
                <tr>
                    <td colspan="4" class="px-6 py-12 text-center" style="color: var(--text-muted);">No users found.</td>
                </tr>
                @endforelse
            </tbody>
        </table>

        @if($users->hasPages())
        <div class="px-6 py-4" style="border-top: 1px solid var(--border-color);">
            {{ $users->links() }}
        </div>
        @endif
    </div>
</div>
@endsection
