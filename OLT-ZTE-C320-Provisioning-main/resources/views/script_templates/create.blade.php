@extends('layouts.app')

@section('content')
<div class="mb-6">
    <div class="flex items-center space-x-3">
        <a href="{{ route('script-templates.index') }}" class="transition-colors" style="color: var(--text-muted);" onmouseover="this.style.color='var(--accent)'" onmouseout="this.style.color='var(--text-muted)'">
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </a>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">Add New Script Template</h1>
    </div>
</div>

<div class="glass-card max-w-4xl">
    <form action="{{ route('script-templates.store') }}" method="POST" class="p-6">
        @csrf
        <div class="space-y-5">
            <div>
                <label for="merk" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">Merk / Brand <span style="color: #f87171;">*</span></label>
                <input type="text" name="merk" id="merk" value="{{ old('merk') }}" placeholder="e.g. ZTE F609" required class="block w-full rounded-xl sm:text-sm px-4 py-2.5 transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); --tw-ring-color: var(--accent);">
                @error('merk') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                <div class="mt-2 flex flex-wrap gap-1.5">
                    <span class="text-xs" style="color: var(--text-muted);">Placeholders:</span>
                    <code class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">{UP_PROFILE}</code>
                    <code class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">{DOWN_PROFILE}</code>
                    <code class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">{ACS_URL}</code>
                    <code class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">{ACS_USER}</code>
                    <code class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background: rgba(6,182,212,0.1); color: var(--accent);">{ACS_PASS}</code>
                </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div>
                    <label for="gpon_onu_script" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">interface gpon-onu Script <span style="color: #f87171;">*</span></label>
                    <textarea name="gpon_onu_script" id="gpon_onu_script" rows="14" class="w-full p-4 text-xs font-mono rounded-xl whitespace-pre transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--accent); --tw-ring-color: var(--accent); line-height: 1.6;" spellcheck="false" required>{{ old('gpon_onu_script', "tcont 1 name TR069 profile {UP_PROFILE}\ntcont 2 name PPPoE profile {UP_PROFILE}\ntcont 3 name HOSTPOT profile {UP_PROFILE}\ngemport 1 tcont 1\ngemport 1 traffic-limit downstream {DOWN_PROFILE}\ngemport 2 tcont 2\ngemport 2 traffic-limit downstream {DOWN_PROFILE}\ngemport 3 tcont 3\ngemport 3 traffic-limit downstream {DOWN_PROFILE}\nservice-port 1 vport 1 user-vlan 100 vlan 100\nservice-port 2 vport 2 user-vlan 301 vlan 301\nservice-port 3 vport 3 user-vlan 302 vlan 302") }}</textarea>
                    @error('gpon_onu_script') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                </div>
                <div>
                    <label for="pon_onu_mng_script" class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary);">pon-onu-mng Script <span style="color: #f87171;">*</span></label>
                    <textarea name="pon_onu_mng_script" id="pon_onu_mng_script" rows="14" class="w-full p-4 text-xs font-mono rounded-xl whitespace-pre transition-all duration-200 focus:outline-none focus:ring-2" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--accent); --tw-ring-color: var(--accent); line-height: 1.6;" spellcheck="false" required>{{ old('pon_onu_mng_script', "service TR069 gemport 1 vlan 100\nservice PPPoE gemport 2 vlan 301\nservice HOSTPOT gemport 3 vlan 302\nvlan port veip_1 mode hybrid def-vlan 100\nvlan port veip_1 vlan 100,301,302\ntr069-mgmt 1 state unlock\ntr069-mgmt 1 acs {ACS_URL} validate basic username {ACS_USER} password {ACS_PASS}") }}</textarea>
                    @error('pon_onu_mng_script') <p class="mt-1 text-sm" style="color: #f87171;">{{ $message }}</p> @enderror
                </div>
            </div>

            <div class="flex items-center pt-2">
                <input id="is_default" name="is_default" type="checkbox" value="1" {{ old('is_default') ? 'checked' : '' }} class="h-4 w-4 rounded border-gray-600 focus:ring-cyan-500" style="background: var(--bg-primary); color: var(--accent);">
                <label for="is_default" class="ml-2 block text-sm" style="color: var(--text-secondary);">Set as Default Script Template</label>
            </div>
        </div>

        <div class="mt-8 flex justify-end space-x-3">
            <a href="{{ route('script-templates.index') }}" class="py-2.5 px-5 rounded-xl text-sm font-medium transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">Cancel</a>
            <button type="submit" class="py-2.5 px-5 rounded-xl text-sm font-semibold text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">Save Template</button>
        </div>
    </form>
</div>
@endsection
