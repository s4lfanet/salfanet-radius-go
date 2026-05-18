<x-guest-layout>
    <!-- Session Status -->
    <x-auth-session-status class="mb-4" :status="session('status')" />

    <h2 class="text-xl font-bold text-center mb-1" style="color: #f1f5f9;">Welcome Back</h2>
    <p class="text-center text-sm mb-6" style="color: #64748b;">Sign in to your account</p>

    <form method="POST" action="{{ route('login') }}">
        @csrf

        <!-- Email Address -->
        <div class="mb-4">
            <label for="email" class="block text-sm font-medium mb-1.5" style="color: #94a3b8;">Email</label>
            <input id="email" type="email" name="email" value="{{ old('email') }}" required autofocus autocomplete="username"
                class="block w-full rounded-xl px-4 py-2.5 text-sm transition-all duration-200 focus:outline-none focus:ring-2"
                style="background: #0a0e1a; border: 1px solid rgba(255,255,255,0.06); color: #f1f5f9; --tw-ring-color: #06b6d4;"
                placeholder="you@example.com">
            <x-input-error :messages="$errors->get('email')" class="mt-2" />
        </div>

        <!-- Password -->
        <div class="mb-4">
            <label for="password" class="block text-sm font-medium mb-1.5" style="color: #94a3b8;">Password</label>
            <input id="password" type="password" name="password" required autocomplete="current-password"
                class="block w-full rounded-xl px-4 py-2.5 text-sm transition-all duration-200 focus:outline-none focus:ring-2"
                style="background: #0a0e1a; border: 1px solid rgba(255,255,255,0.06); color: #f1f5f9; --tw-ring-color: #06b6d4;"
                placeholder="••••••••">
            <x-input-error :messages="$errors->get('password')" class="mt-2" />
        </div>

        <!-- Remember Me -->
        <div class="flex items-center justify-between mb-6">
            <label for="remember_me" class="inline-flex items-center cursor-pointer">
                <input id="remember_me" type="checkbox" name="remember"
                    class="rounded border-gray-600 text-cyan-500 shadow-sm focus:ring-cyan-500"
                    style="background: #0a0e1a;">
                <span class="ms-2 text-sm" style="color: #64748b;">{{ __('Remember me') }}</span>
            </label>
            @if (Route::has('password.request'))
                <a class="text-sm font-medium transition-colors" style="color: #06b6d4;" href="{{ route('password.request') }}">
                    Forgot password?
                </a>
            @endif
        </div>

        <button type="submit" class="w-full py-2.5 px-4 rounded-xl text-sm font-semibold text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-[1.02]" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
            Sign In
        </button>
    </form>
</x-guest-layout>
