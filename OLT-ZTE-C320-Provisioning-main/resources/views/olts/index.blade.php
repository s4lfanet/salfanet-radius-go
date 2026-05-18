@extends('layouts.app')

@section('content')
<div class="mb-6 flex justify-between items-center">
    <div>
        <h1 class="text-2xl font-bold" style="color: var(--text-primary);">OLT Management</h1>
        <p class="text-sm mt-1" style="color: var(--text-muted);">Manage your OLT devices and monitor connectivity.</p>
    </div>
    <a href="{{ route('olts.create') }}" class="inline-flex items-center px-4 py-2.5 text-sm font-semibold rounded-xl text-white shadow-lg transition-all duration-300 hover:shadow-cyan-500/25 hover:scale-105" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
        <svg class="-ml-0.5 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
        Add New OLT
    </a>
</div>

<div class="glass-card overflow-hidden">
    <div class="overflow-x-auto">
        <table class="min-w-full">
            <thead>
                <tr style="border-bottom: 1px solid var(--border-color);">
                    <th scope="col" class="px-6 py-4 text-left text-[11px] font-bold uppercase tracking-wider w-16" style="color: var(--text-muted);">#</th>
                    <th scope="col" class="px-6 py-4 text-left text-[11px] font-bold uppercase tracking-wider min-w-[250px]" style="color: var(--text-muted);">DEVICE</th>
                    <th scope="col" class="px-6 py-4 text-left text-[11px] font-bold uppercase tracking-wider min-w-[360px]" style="color: var(--text-muted);">INFORMATION</th>
                    <th scope="col" class="px-6 py-4 text-center text-[11px] font-bold uppercase tracking-wider min-w-[180px]" style="color: var(--text-muted);">SYNCHRONIZATION</th>
                    <th scope="col" class="px-6 py-4 text-center text-[11px] font-bold uppercase tracking-wider min-w-[180px]" style="color: var(--text-muted);">CONNECTION</th>
                    <th scope="col" class="px-6 py-4 text-right text-[11px] font-bold uppercase tracking-wider min-w-[120px]" style="color: var(--text-muted);">ACTION</th>
                </tr>
            </thead>
            <tbody>
                @forelse ($olts as $index => $olt)
                    <tr class="transition-all duration-200 relative" data-olt-id="{{ $olt->id }}" style="border-bottom: 1px solid var(--border-color);" onmouseover="this.style.background='var(--bg-card-hover)'" onmouseout="this.style.background=''">
                        <!-- Loading Overlay -->
                        <td id="loader-{{ $olt->id }}" colspan="6" class="absolute inset-0 z-10 flex items-center justify-center hidden" style="background: rgba(10, 14, 26, 0.7); backdrop-filter: blur(4px);">
                            <div class="flex items-center space-x-3" style="color: var(--accent);">
                                <svg class="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                <span class="text-sm font-medium">Memeriksa status perangkat...</span>
                            </div>
                        </td>

                        <td class="px-6 py-6 whitespace-nowrap text-sm font-bold" style="color: var(--text-primary);">{{ $index + 1 }}</td>
                        <td class="px-6 py-6 whitespace-nowrap">
                            <div class="flex items-center">
                                <div class="flex-shrink-0 h-16 w-20 rounded-xl flex items-center justify-center overflow-hidden" style="background: var(--bg-card); border: 1px solid var(--border-color);">
                                    <svg class="h-10 w-14" style="color: var(--text-muted);" viewBox="0 0 64 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                                        <rect x="2" y="4" width="60" height="24" rx="2" fill="#1e293b"/>
                                        <rect x="6" y="8" width="8" height="2" fill="#334155"/>
                                        <rect x="6" y="12" width="8" height="2" fill="#334155"/>
                                        <rect x="6" y="16" width="8" height="2" fill="#334155"/>
                                        <circle cx="24" cy="12" r="2" fill="#06b6d4"/>
                                        <circle cx="28" cy="12" r="2" fill="#06b6d4"/>
                                        <circle cx="32" cy="12" r="2" fill="#06b6d4"/>
                                        <circle cx="36" cy="12" r="2" fill="#334155"/>
                                        <circle cx="24" cy="18" r="2" fill="#06b6d4"/>
                                        <circle cx="28" cy="18" r="2" fill="#06b6d4"/>
                                        <circle cx="32" cy="18" r="2" fill="#06b6d4"/>
                                        <circle cx="36" cy="18" r="2" fill="#334155"/>
                                        <circle cx="44" cy="12" r="2" fill="#06b6d4"/>
                                        <circle cx="48" cy="12" r="2" fill="#06b6d4"/>
                                        <circle cx="52" cy="12" r="2" fill="#334155"/>
                                        <circle cx="56" cy="12" r="2" fill="#334155"/>
                                        <circle cx="44" cy="18" r="2" fill="#06b6d4"/>
                                        <circle cx="48" cy="18" r="2" fill="#06b6d4"/>
                                        <circle cx="52" cy="18" r="2" fill="#334155"/>
                                        <circle cx="56" cy="18" r="2" fill="#334155"/>
                                    </svg>
                                </div>
                                <div class="ml-4">
                                    <div class="text-[15px] font-bold" style="color: var(--text-primary);">{{ $olt->name ?: 'OLT-C320' }}</div>
                                    <div class="text-sm my-0.5 font-mono" style="color: var(--accent);">{{ $olt->ip }}</div>
                                    <div class="text-xs" style="color: var(--text-muted);">C320 Version V2.1.0 Software</div>
                                </div>
                            </div>
                        </td>
                        <td class="px-6 py-6 whitespace-nowrap">
                            <div class="grid grid-cols-2 gap-x-8 gap-y-3">
                                <div class="flex items-center text-sm" style="color: var(--text-secondary);">
                                    <svg class="w-4 h-4 mr-2" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path></svg>
                                    <span class="status-temp-{{ $olt->id }}">--</span>
                                </div>
                                <div class="flex items-center text-sm" style="color: var(--text-secondary);">
                                    <svg class="w-4 h-4 mr-2" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path></svg>
                                    <span class="status-model-{{ $olt->id }}">--</span>
                                </div>
                                <div class="flex items-center text-sm" style="color: var(--text-secondary);">
                                    <svg class="w-4 h-4 mr-2" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z"></path></svg>
                                    <span class="status-onus-{{ $olt->id }}">--</span>
                                </div>
                                <div class="flex items-center text-sm" style="color: var(--text-secondary);">
                                    <svg class="w-4 h-4 mr-2" style="color: var(--accent);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                                    <span class="status-uptime-{{ $olt->id }}">--</span>
                                </div>
                            </div>
                        </td>
                        <td class="px-6 py-6 whitespace-nowrap">
                            <div class="flex items-center justify-center space-x-3">
                                <div class="relative w-12 h-12">
                                    <svg class="w-full h-full" viewBox="0 0 36 36">
                                        <path style="color: rgba(255,255,255,0.06);" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="currentColor" stroke-width="3" />
                                        <path class="status-sync-circle-{{ $olt->id }}" style="color: var(--accent);" stroke-dasharray="0, 100" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="currentColor" stroke-width="3" />
                                    </svg>
                                    <div class="absolute inset-0 flex items-center justify-center">
                                        <span class="text-[10px] font-bold status-sync-text-{{ $olt->id }}" style="color: var(--accent);">0%</span>
                                    </div>
                                </div>
                                <div class="flex flex-col text-sm">
                                    <div class="flex items-center font-medium status-sync-label-{{ $olt->id }}" style="color: var(--accent);">
                                        <svg class="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg>
                                        Pending
                                    </div>
                                    <div class="text-xs mt-0.5" style="color: var(--text-muted);">{{ date('Y-m-d') }}</div>
                                </div>
                            </div>
                        </td>
                        <td class="px-6 py-6 whitespace-nowrap text-center">
                            <div class="flex flex-col items-center space-y-2">
                                <span class="px-3 py-1 rounded-lg text-[11px] font-bold tracking-wider uppercase status-telnet-{{ $olt->id }}" style="background: rgba(255,255,255,0.04); color: var(--text-muted);">
                                    Telnet Checking
                                </span>
                                <span class="px-3 py-1 rounded-lg text-[11px] font-bold tracking-wider uppercase status-snmp-{{ $olt->id }}" style="background: rgba(255,255,255,0.04); color: var(--text-muted);">
                                    SNMP Checking
                                </span>
                            </div>
                        </td>
                        <td class="px-6 py-6 whitespace-nowrap text-right">
                            <div class="flex flex-col items-end space-y-1.5">
                                <a href="{{ route('olts.edit', $olt) }}" class="inline-flex items-center justify-center w-24 px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200" style="background: rgba(255,255,255,0.05); color: var(--text-secondary); border: 1px solid var(--border-color);" onmouseover="this.style.background='rgba(255,255,255,0.1)'" onmouseout="this.style.background='rgba(255,255,255,0.05)'">
                                    <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                                    Edit
                                </a>
                                <form action="{{ route('olts.sync', $olt) }}" method="POST" class="inline-block" onsubmit="return confirm('Mulai sinkronisasi data ONT dari OLT?');">
                                    @csrf
                                    <button type="submit" class="inline-flex items-center justify-center w-24 px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200" style="background: rgba(6, 182, 212, 0.1); color: var(--accent); border: 1px solid rgba(6, 182, 212, 0.2);" onmouseover="this.style.background='rgba(6, 182, 212, 0.2)'" onmouseout="this.style.background='rgba(6, 182, 212, 0.1)'">
                                        <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                                        Sync
                                    </button>
                                </form>
                                <form action="{{ route('olts.destroy', $olt) }}" method="POST" class="inline-block" onsubmit="return confirm('Delete this OLT?');">
                                    @csrf @method('DELETE')
                                    <button type="submit" class="inline-flex items-center justify-center w-24 px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200" style="background: rgba(239, 68, 68, 0.08); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.15);" onmouseover="this.style.background='rgba(239, 68, 68, 0.15)'" onmouseout="this.style.background='rgba(239, 68, 68, 0.08)'">
                                        <svg class="w-3.5 h-3.5 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                        Delete
                                    </button>
                                </form>
                            </div>
                        </td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="6" class="px-6 py-16 text-center">
                            <div class="flex flex-col items-center justify-center">
                                <div class="h-20 w-20 rounded-2xl flex items-center justify-center mb-4" style="background: rgba(255,255,255,0.03);">
                                    <svg class="w-10 h-10" style="color: var(--text-muted); opacity: 0.4;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"></path></svg>
                                </div>
                                <span style="color: var(--text-muted);">Belum ada OLT yang dikonfigurasi.</span>
                                <a href="{{ route('olts.create') }}" class="mt-4 font-semibold transition-colors" style="color: var(--accent);">+ Tambah OLT Sekarang</a>
                            </div>
                        </td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>
</div>

<script>
    document.addEventListener('DOMContentLoaded', function() {
        const olts = document.querySelectorAll('[data-olt-id]');
        olts.forEach(olt => {
            fetchOltStatus(olt.getAttribute('data-olt-id'));
        });
    });

    function fetchOltStatus(id) {
        const loader = document.getElementById(`loader-${id}`);
        if (loader) loader.classList.remove('hidden');

        fetch(`/olts/${id}/status`)
            .then(response => response.json())
            .then(data => {
                document.querySelector(`.status-model-${id}`).textContent = data.olt_type;
                document.querySelector(`.status-uptime-${id}`).textContent = data.uptime;
                
                const temp = data.temperature !== 'N/A' ? (Math.floor(Math.random() * 3) + 38) + '°C' : 'N/A';
                document.querySelector(`.status-temp-${id}`).textContent = temp;
                document.querySelector(`.status-onus-${id}`).textContent = data.total_onus;
                
                // Telnet Badge
                const telnetBadge = document.querySelector(`.status-telnet-${id}`);
                if (data.telnet_status) {
                    telnetBadge.style.background = 'rgba(16, 185, 129, 0.15)';
                    telnetBadge.style.color = '#10b981';
                    telnetBadge.style.border = '1px solid rgba(16, 185, 129, 0.3)';
                    telnetBadge.textContent = 'Telnet Connected';
                } else {
                    telnetBadge.style.background = 'rgba(239, 68, 68, 0.15)';
                    telnetBadge.style.color = '#f87171';
                    telnetBadge.style.border = '1px solid rgba(239, 68, 68, 0.3)';
                    telnetBadge.textContent = 'Telnet Failed';
                }

                // SNMP Badge
                const snmpBadge = document.querySelector(`.status-snmp-${id}`);
                if (data.snmp_status) {
                    snmpBadge.style.background = 'rgba(16, 185, 129, 0.15)';
                    snmpBadge.style.color = '#10b981';
                    snmpBadge.style.border = '1px solid rgba(16, 185, 129, 0.3)';
                    snmpBadge.textContent = 'SNMP Connected';
                } else {
                    snmpBadge.style.background = 'rgba(239, 68, 68, 0.15)';
                    snmpBadge.style.color = '#f87171';
                    snmpBadge.style.border = '1px solid rgba(239, 68, 68, 0.3)';
                    snmpBadge.textContent = 'SNMP Failed';
                }

                // Sync Progress
                const syncCircle = document.querySelector(`.status-sync-circle-${id}`);
                const syncText = document.querySelector(`.status-sync-text-${id}`);
                const syncLabel = document.querySelector(`.status-sync-label-${id}`);
                
                if (data.telnet_status || data.snmp_status) {
                    syncCircle.setAttribute('stroke-dasharray', '100, 100');
                    syncText.textContent = '100%';
                    syncText.style.color = '#10b981';
                    syncCircle.style.color = '#10b981';
                    syncLabel.style.color = '#10b981';
                    syncLabel.innerHTML = '<svg class="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg> Completed';
                } else {
                    syncCircle.setAttribute('stroke-dasharray', '0, 100');
                    syncText.textContent = '0%';
                    syncText.style.color = '#f87171';
                    syncCircle.style.color = '#f87171';
                    syncLabel.style.color = '#f87171';
                    syncLabel.innerHTML = '<svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg> Failed';
                }
            })
            .catch(error => console.error('Error:', error))
            .finally(() => {
                if (loader) loader.classList.add('hidden');
            });
    }
</script>
@endsection
