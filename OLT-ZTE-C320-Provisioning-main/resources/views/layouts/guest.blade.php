<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ config('app.name', 'OLT Provisioning') }} - Login</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
    </style>
</head>
<body class="antialiased" style="background: #0a0e1a;">
    <div class="min-h-screen flex flex-col sm:justify-center items-center pt-6 sm:pt-0">
        <!-- Logo -->
        <div class="mb-6 flex flex-col items-center">
            <div class="h-14 w-14 rounded-2xl flex items-center justify-center mb-3" style="background: linear-gradient(135deg, #06b6d4, #0891b2); box-shadow: 0 0 30px rgba(6, 182, 212, 0.3);">
                <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                </svg>
            </div>
            <h1 class="text-xl font-bold" style="color: #f1f5f9;">ZTE Provisioner</h1>
            <p class="text-xs font-medium tracking-widest uppercase mt-0.5" style="color: #06b6d4;">PRO EDITION</p>
        </div>

        <!-- Card -->
        <div class="w-full sm:max-w-md px-8 py-6 overflow-hidden rounded-2xl" style="background: linear-gradient(135deg, rgba(26, 31, 46, 0.8), rgba(17, 24, 39, 0.9)); backdrop-filter: blur(20px); border: 1px solid rgba(255,255,255,0.06);">
            {{ $slot }}
        </div>
    </div>
</body>
</html>
