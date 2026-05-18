@extends('layouts.app')

@section('content')
<div class="mb-6">
    <div class="flex items-center space-x-3">
        <a href="{{ route('olts.index') }}" class="transition-colors" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)'" onmouseout="this.style.color='var(--text-muted)'">
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </a>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Edit OLT</h1>
    </div>
</div>

<div class="glass-card max-w-3xl">
    <form action="{{ route('olts.update', $olt) }}" method="POST" class="p-6">
        @csrf
        @method('PUT')

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="col-span-2">
                <label for="name" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">OLT Name (Optional)</label>
                <input type="text" name="name" id="name" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('name', $olt->name) }}">
                @error('name') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
            </div>

            <div class="col-span-2 md:col-span-1">
                <label for="ip" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">IP Address <span style="color: #f87171;">*</span></label>
                <input type="text" name="ip" id="ip" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('ip', $olt->ip) }}" required>
                @error('ip') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
            </div>
            
            <div class="col-span-2"><hr style="border-color: var(--border-color);" class="my-2"></div>

            <!-- Telnet Settings -->
            <div class="col-span-2 md:col-span-1">
                <h3 class="text-lg font-semibold mb-4 flex items-center" style="color: var(--text-primary);">
                    <span class="h-2 w-2 rounded-full mr-2" style="background: var(--accent);"></span>
                    Telnet Settings
                </h3>
                <div class="space-y-4">
                    <div>
                        <label for="telnet_username" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Username</label>
                        <input type="text" name="telnet_username" id="telnet_username" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('telnet_username', $olt->telnet_username) }}">
                        @error('telnet_username') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                    <div>
                        <label for="telnet_password" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Password</label>
                        <input type="password" name="telnet_password" id="telnet_password" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('telnet_password', $olt->telnet_password) }}">
                        @error('telnet_password') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                    <div>
                        <label for="telnet_port" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Port</label>
                        <input type="number" name="telnet_port" id="telnet_port" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('telnet_port', $olt->telnet_port) }}" required>
                        @error('telnet_port') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                </div>
            </div>

            <!-- SNMP Settings -->
            <div class="col-span-2 md:col-span-1">
                <h3 class="text-lg font-semibold mb-4 flex items-center" style="color: var(--text-primary);">
                    <span class="h-2 w-2 rounded-full mr-2" style="background: #8b5cf6;"></span>
                    SNMP Settings
                </h3>
                <div class="space-y-4">
                    <div>
                        <label for="snmp_version" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">SNMP Version <span style="color: #f87171;">*</span></label>
                        <select name="snmp_version" id="snmp_version" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" onchange="toggleSnmpLabels(this.value)">
                            <option value="v2c" {{ old('snmp_version', $olt->snmp_version ?? 'v2c') == 'v2c' ? 'selected' : '' }}>SNMPv2c (Community String)</option>
                            <option value="v3" {{ old('snmp_version', $olt->snmp_version ?? 'v2c') == 'v3' ? 'selected' : '' }}>SNMPv3 (Username/Password)</option>
                        </select>
                        @error('snmp_version') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                    <div>
                        <label for="snmp_username" id="snmp_username_label" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Community String</label>
                        <input type="text" name="snmp_username" id="snmp_username" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('snmp_username', $olt->snmp_username) }}">
                        @error('snmp_username') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                    <div id="snmp_password_row">
                        <label for="snmp_password" id="snmp_password_label" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Auth Password</label>
                        <input type="password" name="snmp_password" id="snmp_password" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('snmp_password', $olt->snmp_password) }}">
                        @error('snmp_password') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                    <div>
                        <label for="snmp_port" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Port</label>
                        <input type="number" name="snmp_port" id="snmp_port" class="mt-1 block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);" value="{{ old('snmp_port', $olt->snmp_port) }}" required>
                        @error('snmp_port') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                    </div>
                </div>
            </div>
        </div>

        <div class="mt-8 flex justify-end space-x-3">
            <a href="{{ route('olts.index') }}" class="py-2.5 px-5 rounded-xl text-sm font-medium transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">
                Cancel
            </a>
            <button type="submit" class="py-2.5 px-5 rounded-xl text-sm font-semibold text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                Update OLT
            </button>
        </div>
    </form>
</div>

<script>
function toggleSnmpLabels(version) {
    const usernameLabel = document.getElementById('snmp_username_label');
    const passwordRow = document.getElementById('snmp_password_row');
    const usernameInput = document.getElementById('snmp_username');

    if (version === 'v2c') {
        usernameLabel.textContent = 'Community String';
        usernameInput.placeholder = 'e.g. public / rconfigrw';
        passwordRow.style.display = 'none';
    } else {
        usernameLabel.textContent = 'Username';
        usernameInput.placeholder = 'e.g. oltuser';
        passwordRow.style.display = 'block';
    }
}
document.addEventListener('DOMContentLoaded', function() {
    toggleSnmpLabels(document.getElementById('snmp_version').value);
});
</script>
@endsection
