/**
 * PM2 Ecosystem Configuration — Salfanet Radius ISP
 *
 * Managed processes (post Go-backend migration):
 *   1. salfanet-radius  — Next.js frontend (standalone server.js, port 3000, UI only)
 *   2. salfanet-cron    — Background billing/expiry cron (calls Go API on port 8080)
 *   3. salfanet-wa      — Baileys WhatsApp native service (port 4000, internal only)
 *
 * Go API backend is managed by systemd (salfanet-api.service, port 8080).
 * Nginx routes /api/ → Go:8080, / → Next.js:3000
 *
 * This file is copied by install-pm2.sh and updater.sh to APP_DIR/ecosystem.config.js
 */

const APP_DIR = process.env.APP_DIR || '/var/www/salfanet-radius';

module.exports = {
  apps: [
    // ─────────────────────────────────────────────────────────────────────
    // 1. Main Next.js Application
    // ─────────────────────────────────────────────────────────────────────
    {
      name: 'salfanet-radius',
      script: '.next/standalone/server.js',
      cwd: APP_DIR,
      instances: 1,
      exec_mode: 'fork',          // fork: 1 process, no cluster master overhead
      watch: false,
      max_memory_restart: '600M', // increased from 450M (VPS has 3.8GB)
      node_args: [
        '--max-old-space-size=512', // increased from 400M, reduces GC pressure
        '--max-semi-space-size=16',
      ],
      env: {
        NODE_ENV: 'production',
        NODE_OPTIONS: '--max-old-space-size=512',
        PORT: 3000,
        HOSTNAME: '127.0.0.1',
        TZ: 'Asia/Jakarta',
      },
      error_file: './logs/error.log',
      out_file: './logs/out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      autorestart: true,
      max_restarts: 10,
      min_uptime: '30s',
      // cron_restart removed — was causing 4x/day forced downtime
    },

    // ─────────────────────────────────────────────────────────────────────
    // 2. Background Cron Service (billing, expiry, notifications)
    // ─────────────────────────────────────────────────────────────────────
    {
      name: 'salfanet-cron',
      script: './cron-service.js',
      cwd: APP_DIR,
      instances: 1,
      exec_mode: 'fork',
      watch: false,
      max_memory_restart: '150M',
      node_args: [
        '--max-old-space-size=120',
        '--max-semi-space-size=4',
        '--optimize-for-size',
      ],
      env: {
        NODE_ENV: 'production',
        NODE_OPTIONS: '--max-old-space-size=120',
        API_URL: 'http://localhost:8080',
        TZ: 'Asia/Jakarta',
      },
      error_file: './logs/cron-error.log',
      out_file: './logs/cron-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      autorestart: true,
      max_restarts: 5,
      min_uptime: '10s',
      restart_delay: 5000,
    },

    // ─────────────────────────────────────────────────────────────────────
    // 3. Baileys WhatsApp Native Service
    //    Listens on 127.0.0.1:4000 (internal only, proxied via /api/whatsapp)
    //    Auth files: /var/data/salfanet/baileys_auth/
    // ─────────────────────────────────────────────────────────────────────
    {
      name: 'salfanet-wa',
      script: './wa-service.js',
      cwd: APP_DIR,
      instances: 1,
      exec_mode: 'fork',
      watch: false,
      max_memory_restart: '200M',
      node_args: [
        '--max-old-space-size=180',
        '--max-semi-space-size=4',
      ],
      env: {
        NODE_ENV: 'production',
        NODE_OPTIONS: '--max-old-space-size=180',
        WA_SERVICE_PORT: 4000,
        WA_AUTH_DIR: '/var/data/salfanet/baileys_auth',
        TZ: 'Asia/Jakarta',
      },
      error_file: './logs/wa-error.log',
      out_file: './logs/wa-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      autorestart: true,
      max_restarts: 10,
      min_uptime: '5s',
      restart_delay: 3000,
    },
  ],
};
