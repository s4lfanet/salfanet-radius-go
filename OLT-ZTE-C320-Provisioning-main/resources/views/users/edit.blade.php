@extends('layouts.app')

@section('title', 'Edit User - ' . $user->name)

@section('content')
<div class="animate-fade-in-up max-w-2xl mx-auto">
    <div class="mb-6">
        <a href="{{ route('users.index') }}" class="inline-flex items-center text-sm font-medium transition-colors" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)'" onmouseout="this.style.color='var(--text-muted)'">
            <svg class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" /></svg>
            Back to Users
        </a>
        <h1 class="text-2xl font-bold mt-2" style="color: var(--text-primary);">Edit User</h1>
    </div>

    <!-- User Info Card -->
    <div class="glass-card p-4 mb-5 flex items-center space-x-4">
        <div class="h-12 w-12 rounded-xl flex items-center justify-center text-white font-bold text-sm uppercase" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
            {{ substr($user->name, 0, 2) }}
        </div>
        <div>
            <p class="text-base font-bold" style="color: var(--text-primary);">{{ $user->name }}</p>
            <p class="text-xs font-mono" style="color: var(--text-muted);">{{ $user->email }} · Created {{ $user->created_at->diffForHumans() }}</p>
        </div>
    </div>

    <div class="glass-card p-6">
        <form method="POST" action="{{ route('users.update', $user) }}" class="space-y-5">
            @csrf @method('PUT')

            <div>
                <label class="block text-sm font-semibold mb-2" style="color: var(--text-secondary);">Full Name</label>
                <input type="text" name="name" value="{{ old('name', $user->name) }}" required
                    class="w-full px-4 py-3 rounded-xl text-sm font-medium outline-none transition-all duration-200"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    onfocus="this.style.borderColor='var(--accent)';this.style.boxShadow='0 0 0 3px var(--accent-glow)'"
                    onblur="this.style.borderColor='var(--border-color)';this.style.boxShadow=''">
                @error('name')<p class="mt-1 text-xs" style="color: #ef4444;">{{ $message }}</p>@enderror
            </div>

            <div>
                <label class="block text-sm font-semibold mb-2" style="color: var(--text-secondary);">Email Address</label>
                <input type="email" name="email" value="{{ old('email', $user->email) }}" required
                    class="w-full px-4 py-3 rounded-xl text-sm font-medium outline-none transition-all duration-200"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    onfocus="this.style.borderColor='var(--accent)';this.style.boxShadow='0 0 0 3px var(--accent-glow)'"
                    onblur="this.style.borderColor='var(--border-color)';this.style.boxShadow=''">
                @error('email')<p class="mt-1 text-xs" style="color: #ef4444;">{{ $message }}</p>@enderror
            </div>

            <div class="pt-3" style="border-top: 1px solid var(--border-color);">
                <p class="text-xs font-medium mb-3" style="color: var(--text-muted);">Leave blank to keep current password</p>
            </div>

            <div>
                <label class="block text-sm font-semibold mb-2" style="color: var(--text-secondary);">New Password</label>
                <input type="password" name="password"
                    class="w-full px-4 py-3 rounded-xl text-sm font-medium outline-none transition-all duration-200"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    onfocus="this.style.borderColor='var(--accent)';this.style.boxShadow='0 0 0 3px var(--accent-glow)'"
                    onblur="this.style.borderColor='var(--border-color)';this.style.boxShadow=''"
                    placeholder="Enter new password (optional)">
                @error('password')<p class="mt-1 text-xs" style="color: #ef4444;">{{ $message }}</p>@enderror
            </div>

            <div>
                <label class="block text-sm font-semibold mb-2" style="color: var(--text-secondary);">Confirm New Password</label>
                <input type="password" name="password_confirmation"
                    class="w-full px-4 py-3 rounded-xl text-sm font-medium outline-none transition-all duration-200"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    onfocus="this.style.borderColor='var(--accent)';this.style.boxShadow='0 0 0 3px var(--accent-glow)'"
                    onblur="this.style.borderColor='var(--border-color)';this.style.boxShadow=''"
                    placeholder="Repeat new password">
            </div>

            <div class="flex justify-end pt-2">
                <button type="submit" class="inline-flex items-center px-6 py-3 rounded-xl text-sm font-semibold text-white transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                    <svg class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                    Save Changes
                </button>
            </div>
        </form>
    </div>
</div>
@endsection
