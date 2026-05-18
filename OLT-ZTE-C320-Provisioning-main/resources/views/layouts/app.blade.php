<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ config('app.name', 'OLT Provisioning') }}</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        
        /* Dark theme variables */
        :root {
            --bg-primary: #0a0e1a;
            --bg-secondary: #111827;
            --bg-card: #1a1f2e;
            --bg-card-hover: #222838;
            --border-color: rgba(255,255,255,0.06);
            --text-primary: #f1f5f9;
            --text-secondary: #94a3b8;
            --text-muted: #64748b;
            --accent: #06b6d4;
            --accent-glow: rgba(6, 182, 212, 0.15);
        }

        /* Glassmorphism card */
        .glass-card {
            background: linear-gradient(135deg, rgba(26, 31, 46, 0.8), rgba(17, 24, 39, 0.9));
            backdrop-filter: blur(20px);
            border: 1px solid var(--border-color);
            border-radius: 16px;
        }

        /* Sidebar gradient */
        .sidebar-gradient {
            background: linear-gradient(180deg, #0f172a 0%, #0a0e1a 50%, #0d1321 100%);
        }

        /* Glow effect */
        .glow-accent {
            box-shadow: 0 0 20px var(--accent-glow), 0 0 60px rgba(6, 182, 212, 0.05);
        }

        /* Active nav indicator */
        .nav-active {
            background: linear-gradient(135deg, rgba(6, 182, 212, 0.12), rgba(6, 182, 212, 0.05));
            border-left: 3px solid var(--accent);
            color: var(--accent);
        }
        .nav-active svg { color: var(--accent); }

        /* Custom scrollbar */
        ::-webkit-scrollbar { width: 5px; height: 5px; }
        ::-webkit-scrollbar-track { background: var(--bg-primary); }
        ::-webkit-scrollbar-thumb { background: #334155; border-radius: 10px; }
        ::-webkit-scrollbar-thumb:hover { background: #475569; }

        /* Animation */
        @keyframes fadeInUp {
            from { opacity: 0; transform: translateY(15px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .animate-fade-in-up { animation: fadeInUp 0.5s ease-out; }

        @keyframes pulse-glow {
            0%, 100% { box-shadow: 0 0 5px rgba(6, 182, 212, 0.3); }
            50% { box-shadow: 0 0 15px rgba(6, 182, 212, 0.6); }
        }
        .pulse-glow { animation: pulse-glow 2s ease-in-out infinite; }

        @keyframes slideIn {
            from { transform: translateX(-100%); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
        .animate-slide-in { animation: slideIn 0.3s ease-out; }
    </style>
</head>
<body class="antialiased" style="background: var(--bg-primary); color: var(--text-primary);">
    
    <div class="flex h-screen overflow-hidden" x-data="{ sidebarOpen: false }">
        
        <!-- Mobile sidebar backdrop -->
        <div x-show="sidebarOpen" class="fixed inset-0 z-20 bg-black/60 backdrop-blur-sm transition-opacity md:hidden" @click="sidebarOpen = false" x-transition.opacity style="display: none;"></div>

        <!-- Sidebar -->
        <aside :class="sidebarOpen ? 'flex absolute inset-y-0 left-0 z-50 animate-slide-in' : 'hidden md:flex'" class="w-64 sidebar-gradient flex-col h-full border-r" style="border-color: var(--border-color);">
            <!-- Sidebar Header -->
            <div class="h-16 flex items-center px-5 border-b" style="border-color: var(--border-color);">
                <div class="flex items-center space-x-3">
                    <div class="h-9 w-9 rounded-xl flex items-center justify-center" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                        <svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                        </svg>
                    </div>
                    <div>
                        <span class="text-base font-bold" style="color: var(--text-primary);">ZTE Provisioner</span>
                        <p class="text-[10px] font-medium tracking-widest uppercase" style="color: var(--accent);">PRO EDITION</p>
                    </div>
                </div>
            </div>

            <!-- Sidebar Navigation -->
            <nav class="flex-1 overflow-y-auto py-4 px-3 space-y-1">
                <p class="px-3 mb-2 text-[10px] font-bold tracking-widest uppercase" style="color: var(--text-muted);">Main Menu</p>
                
                <!-- Dashboard -->
                <a href="{{ route('dashboard') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('dashboard') ? 'nav-active' : '' }}" style="{{ request()->routeIs('dashboard') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5 transition-colors" style="{{ request()->routeIs('dashboard') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                    </svg>
                    Dashboard
                </a>

                <!-- Unconfigured ONUs -->
                <a href="{{ route('onus.unconfigured') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('onus.unconfigured') ? 'nav-active' : '' }}" style="{{ request()->routeIs('onus.unconfigured') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('onus.unconfigured') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                    </svg>
                    Unconfigured ONUs
                </a>

                <!-- Provisioned ONUs -->
                <a href="{{ route('onus.index') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('onus.index') ? 'nav-active' : '' }}" style="{{ request()->routeIs('onus.index') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('onus.index') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                    Provisioned ONUs
                </a>

                <p class="px-3 mt-5 mb-2 text-[10px] font-bold tracking-widest uppercase" style="color: var(--text-muted);">Settings</p>

                <!-- OLT Management -->
                <a href="{{ route('olts.index') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('olts.*') ? 'nav-active' : '' }}" style="{{ request()->routeIs('olts.*') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('olts.*') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                    </svg>
                    OLT Management
                </a>

                <!-- ACS Profiles -->
                <a href="{{ route('acs-profiles.index') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('acs-profiles.*') ? 'nav-active' : '' }}" style="{{ request()->routeIs('acs-profiles.*') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('acs-profiles.*') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
                    </svg>
                    ACS Settings
                </a>

                <!-- Script Templates -->
                <a href="{{ route('script-templates.index') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('script-templates.*') ? 'nav-active' : '' }}" style="{{ request()->routeIs('script-templates.*') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('script-templates.*') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                    </svg>
                    Script Templates
                </a>

                <!-- User Management -->
                <a href="{{ route('users.index') }}" class="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 {{ request()->routeIs('users.*') ? 'nav-active' : '' }}" style="{{ request()->routeIs('users.*') ? '' : 'color: var(--text-secondary);' }}" onmouseover="if(!this.classList.contains('nav-active'))this.style.background='rgba(255,255,255,0.04)'" onmouseout="if(!this.classList.contains('nav-active'))this.style.background=''">
                    <svg class="mr-3 flex-shrink-0 h-5 w-5" style="{{ request()->routeIs('users.*') ? 'color: var(--accent);' : 'color: var(--text-muted);' }}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"></path>
                    </svg>
                    User Management
                </a>
            </nav>

            <!-- Sidebar Footer -->
            <div class="p-4 border-t flex justify-between items-center" style="border-color: var(--border-color);">
                <div class="flex items-center">
                    <div class="flex-shrink-0">
                        <div class="h-9 w-9 rounded-xl flex items-center justify-center text-white font-bold text-xs shadow-lg uppercase" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                            {{ substr(Auth::user()->name ?? 'AD', 0, 2) }}
                        </div>
                    </div>
                    <div class="ml-3">
                        <p class="text-sm font-semibold truncate max-w-[100px]" style="color: var(--text-primary);">{{ Auth::user()->name ?? 'Administrator' }}</p>
                        <p class="text-[11px] font-medium" style="color: var(--text-muted);">System Admin</p>
                    </div>
                </div>
                <form method="POST" action="{{ route('logout') }}" class="ml-2">
                    @csrf
                    <button type="submit" class="p-1.5 rounded-lg transition-all duration-200" style="color: var(--text-muted);" onmouseover="this.style.color='#ef4444';this.style.background='rgba(239,68,68,0.1)'" onmouseout="this.style.color='var(--text-muted)';this.style.background=''">
                        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                        </svg>
                    </button>
                </form>
            </div>
        </aside>

        <!-- Main Content Wrapper -->
        <div class="flex-1 flex flex-col overflow-hidden">
            
            <!-- Mobile Header -->
            <header class="md:hidden h-16 flex items-center justify-between px-4 z-10 border-b" style="background: var(--bg-secondary); border-color: var(--border-color);">
                <div class="flex items-center space-x-2">
                    <div class="h-8 w-8 rounded-lg flex items-center justify-center" style="background: linear-gradient(135deg, #06b6d4, #0891b2);">
                        <svg class="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                        </svg>
                    </div>
                    <span class="text-base font-bold" style="color: var(--text-primary);">ZTE Provisioner</span>
                </div>
                <button @click="sidebarOpen = true" class="focus:outline-none p-1.5 rounded-lg" style="color: var(--text-secondary);">
                    <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                    </svg>
                </button>
            </header>

            <!-- Main Content -->
            <main class="flex-1 overflow-x-hidden overflow-y-auto" style="background: var(--bg-primary);">
                <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                    
                    <!-- Flash Messages -->
                    @if (session('success'))
                        <div class="mb-6 p-4 rounded-xl border animate-fade-in-up" style="background: rgba(16, 185, 129, 0.08); border-color: rgba(16, 185, 129, 0.2);">
                            <div class="flex items-center">
                                <svg class="h-5 w-5 mr-3" style="color: #10b981;" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                </svg>
                                <p class="text-sm font-medium" style="color: #10b981;">{{ session('success') }}</p>
                            </div>
                        </div>
                    @endif

                    @if (session('error'))
                        <div class="mb-6 p-4 rounded-xl border animate-fade-in-up" style="background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.2);">
                            <div class="flex items-center">
                                <svg class="h-5 w-5 mr-3" style="color: #ef4444;" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                </svg>
                                <p class="text-sm font-medium" style="color: #ef4444;">{{ session('error') }}</p>
                            </div>
                        </div>
                    @endif

                    <!-- Content -->
                    <div class="animate-fade-in-up">
                        @yield('content')
                    </div>
                </div>
            </main>
        </div>
    </div>

</body>
</html>
