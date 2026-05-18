@extends('layouts.app')

@section('content')
<div class="mb-6">
    <div class="flex items-center space-x-3">
        <a href="{{ route('acs-profiles.index') }}" class="transition-colors" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)'" onmouseout="this.style.color='var(--text-muted)'">
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </a>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Edit ACS Profile</h1>
    </div>
</div>

<div class="glass-card max-w-2xl">
    <form action="{{ route('acs-profiles.update', $acsProfile) }}" method="POST" class="p-6">
        @csrf @method('PUT')
        <div class="space-y-5">
            <div>
                <label for="name" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Profile Name <span style="color: #f87171;">*</span></label>
                <input type="text" name="name" id="name" value="{{ old('name', $acsProfile->name) }}" required class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
                @error('name') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
            </div>
            <div>
                <label for="url" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">ACS URL <span style="color: #f87171;">*</span></label>
                <input type="url" name="url" id="url" value="{{ old('url', $acsProfile->url) }}" required class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
                @error('url') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div>
                    <label for="username" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Username</label>
                    <input type="text" name="username" id="username" value="{{ old('username', $acsProfile->username) }}" class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
                    @error('username') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                </div>
                <div>
                    <label for="password" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Password</label>
                    <input type="text" name="password" id="password" value="{{ old('password', $acsProfile->password) }}" class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
                    @error('password') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                </div>
            </div>
            <div class="flex items-center pt-2">
                <input id="is_default" name="is_default" type="checkbox" value="1" {{ old('is_default', $acsProfile->is_default) ? 'checked' : '' }} class="h-4 w-4 rounded border-gray-600 focus:ring-cyan-500" style="background: var(--bg-primary); color: var(--accent);">
                <label for="is_default" class="ml-2 block text-sm" style="color: var(--text-secondary);">Set as Default ACS Profile</label>
            </div>
        </div>

        <div class="mt-8 flex justify-end space-x-3">
            <a href="{{ route('acs-profiles.index') }}" class="py-2.5 px-5 rounded-xl text-sm font-medium transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">Cancel</a>
            <button type="submit" class="py-2.5 px-5 rounded-xl text-sm font-semibold text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">Update Profile</button>
        </div>
    </form>
</div>
@endsection
