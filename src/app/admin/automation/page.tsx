'use client';

import { useState, useEffect } from 'react';
import { Loader2, Bell, Zap, Shield, Calendar, RefreshCw, Save, Plus, Trash2 } from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';

interface NotificationTemplate {
  id: string;
  eventType: string;
  channel: string;
  template: string;
  isEnabled: boolean;
}

interface AlertRule {
  id: string;
  name: string;
  triggerEvent: string;
  conditions: string;
  actions: string;
  isEnabled: boolean;
  priority: number;
}

interface PaymentPromise {
  id: string;
  userId: string;
  username: string;
  promiseDate: string;
  status: string;
  createdByName: string | null;
  notes: string | null;
}

type Tab = 'templates' | 'rules' | 'promises';

export default function AutomationPage() {
  const { loading: permLoading } = usePermissions();
  const [tab, setTab] = useState<Tab>('templates');
  const [loading, setLoading] = useState(true);
  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [promises, setPromises] = useState<PaymentPromise[]>([]);
  const [editingTemplate, setEditingTemplate] = useState<NotificationTemplate | null>(null);

  useEffect(() => {
    loadAll();
  }, []);

  const loadAll = async () => {
    setLoading(true);
    try {
      await Promise.all([loadTemplates(), loadRules(), loadPromises()]);
    } finally {
      setLoading(false);
    }
  };

  const loadTemplates = async () => {
    const res = await fetch('/api/automation/notification-templates');
    const data = await res.json();
    setTemplates(data.data || []);
  };

  const loadRules = async () => {
    const res = await fetch('/api/automation/alert-rules');
    const data = await res.json();
    setRules(data.data || []);
  };

  const loadPromises = async () => {
    const res = await fetch('/api/automation/payment-promises');
    const data = await res.json();
    setPromises(data.data || []);
  };

  const handleSeedTemplates = async () => {
    const res = await fetch('/api/automation/notification-templates/seed', { method: 'POST' });
    if (res.ok) {
      await loadTemplates();
    }
  };

  const handleSeedRules = async () => {
    const res = await fetch('/api/automation/alert-rules/seed', { method: 'POST' });
    if (res.ok) {
      await loadRules();
    }
  };

  const handleToggleTemplate = async (t: NotificationTemplate) => {
    await fetch(`/api/automation/notification-templates/${t.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isEnabled: !t.isEnabled }),
    });
    await loadTemplates();
  };

  const handleSaveTemplate = async () => {
    if (!editingTemplate) return;
    await fetch(`/api/automation/notification-templates/${editingTemplate.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ template: editingTemplate.template }),
    });
    setEditingTemplate(null);
    await loadTemplates();
  };

  const handleToggleRule = async (r: AlertRule) => {
    await fetch(`/api/automation/alert-rules/${r.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isEnabled: !r.isEnabled }),
    });
    await loadRules();
  };

  const handleDeleteRule = async (id: string) => {
    if (!confirm('Hapus aturan alert ini?')) return;
    await fetch(`/api/automation/alert-rules/${id}`, { method: 'DELETE' });
    await loadRules();
  };

  const channelColor: Record<string, string> = {
    wa: 'text-green-400',
    email: 'text-blue-400',
    push: 'text-purple-400',
    telegram: 'text-cyan-400',
    portal: 'text-yellow-400',
  };

  if (permLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Automation & Smart Monitoring</h1>
        <p className="text-sm text-gray-400">Event-driven notifications, alert rules, payment promises</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-gray-800">
        <button
          onClick={() => setTab('templates')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            tab === 'templates' ? 'border-cyan-500 text-cyan-400' : 'border-transparent text-gray-400 hover:text-white'
          }`}
        >
          <Bell className="inline h-4 w-4 mr-1" /> Template Notifikasi
        </button>
        <button
          onClick={() => setTab('rules')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            tab === 'rules' ? 'border-cyan-500 text-cyan-400' : 'border-transparent text-gray-400 hover:text-white'
          }`}
        >
          <Zap className="inline h-4 w-4 mr-1" /> Aturan Alert
        </button>
        <button
          onClick={() => setTab('promises')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            tab === 'promises' ? 'border-cyan-500 text-cyan-400' : 'border-transparent text-gray-400 hover:text-white'
          }`}
        >
          <Calendar className="inline h-4 w-4 mr-1" /> Janji Bayar
        </button>
      </div>

      {/* Templates Tab */}
      {tab === 'templates' && (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <p className="text-sm text-gray-400">{templates.length} template</p>
            <button
              onClick={handleSeedTemplates}
              className="flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-1.5 text-sm text-gray-300 hover:text-white hover:border-cyan-500"
            >
              <RefreshCw className="h-4 w-4" /> Seed Default
            </button>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b border-gray-800 bg-gray-900">
                <tr>
                  <th className="px-4 py-2 text-left text-gray-400">Event</th>
                  <th className="px-4 py-2 text-left text-gray-400">Channel</th>
                  <th className="px-4 py-2 text-left text-gray-400">Template</th>
                  <th className="px-4 py-2 text-center text-gray-400">Enabled</th>
                  <th className="px-4 py-2 text-right text-gray-400">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {templates.map((t) => (
                  <tr key={t.id} className="border-b border-gray-800/50">
                    <td className="px-4 py-2 text-white font-mono text-xs">{t.eventType}</td>
                    <td className={`px-4 py-2 font-medium ${channelColor[t.channel] || 'text-gray-400'}`}>{t.channel}</td>
                    <td className="px-4 py-2 text-gray-300 text-xs max-w-md truncate">
                      {editingTemplate?.id === t.id ? (
                        <textarea
                          value={editingTemplate.template}
                          onChange={(e) => setEditingTemplate({ ...editingTemplate, template: e.target.value })}
                          className="w-full rounded border border-gray-700 bg-gray-800 p-2 text-white text-xs"
                          rows={3}
                        />
                      ) : (
                        t.template
                      )}
                    </td>
                    <td className="px-4 py-2 text-center">
                      <button
                        onClick={() => handleToggleTemplate(t)}
                        className={`rounded px-2 py-0.5 text-xs ${t.isEnabled ? 'bg-green-900/50 text-green-400' : 'bg-gray-800 text-gray-500'}`}
                      >
                        {t.isEnabled ? 'ON' : 'OFF'}
                      </button>
                    </td>
                    <td className="px-4 py-2 text-right">
                      {editingTemplate?.id === t.id ? (
                        <button onClick={handleSaveTemplate} className="text-cyan-400 hover:text-cyan-300">
                          <Save className="h-4 w-4" />
                        </button>
                      ) : (
                        <button
                          onClick={() => setEditingTemplate(t)}
                          className="text-gray-400 hover:text-white"
                        >
                          Edit
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
                {templates.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                      Belum ada template. Klik "Seed Default" untuk membuat.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Rules Tab */}
      {tab === 'rules' && (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <p className="text-sm text-gray-400">{rules.length} aturan</p>
            <button
              onClick={handleSeedRules}
              className="flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-1.5 text-sm text-gray-300 hover:text-white hover:border-cyan-500"
            >
              <RefreshCw className="h-4 w-4" /> Seed Default
            </button>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {rules.map((r) => (
              <div key={r.id} className={`rounded-lg border p-4 ${r.isEnabled ? 'border-gray-800 bg-gray-900/50' : 'border-gray-800/50 bg-gray-900/20 opacity-60'}`}>
                <div className="flex items-start justify-between">
                  <div>
                    <h4 className="text-sm font-medium text-white">{r.name}</h4>
                    <p className="text-xs text-gray-500 mt-0.5">Trigger: <span className="font-mono text-cyan-400">{r.triggerEvent}</span></p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleToggleRule(r)}
                      className={`rounded px-2 py-0.5 text-xs ${r.isEnabled ? 'bg-green-900/50 text-green-400' : 'bg-gray-800 text-gray-500'}`}
                    >
                      {r.isEnabled ? 'ON' : 'OFF'}
                    </button>
                    <button onClick={() => handleDeleteRule(r.id)} className="text-red-400 hover:text-red-300">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                <div className="mt-3 space-y-1 text-xs">
                  <p className="text-gray-400">Conditions: <span className="font-mono text-yellow-400">{r.conditions}</span></p>
                  <p className="text-gray-400">Actions: <span className="font-mono text-green-400">{r.actions}</span></p>
                  <p className="text-gray-500">Priority: {r.priority}</p>
                </div>
              </div>
            ))}
            {rules.length === 0 && (
              <div className="col-span-2 text-center text-gray-500 py-8">
                Belum ada aturan. Klik "Seed Default" untuk membuat.
              </div>
            )}
          </div>
        </div>
      )}

      {/* Promises Tab */}
      {tab === 'promises' && (
        <div className="space-y-4">
          <p className="text-sm text-gray-400">{promises.length} janji bayar aktif</p>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b border-gray-800 bg-gray-900">
                <tr>
                  <th className="px-4 py-2 text-left text-gray-400">Username</th>
                  <th className="px-4 py-2 text-left text-gray-400">Tanggal Janji</th>
                  <th className="px-4 py-2 text-left text-gray-400">Dibuat Oleh</th>
                  <th className="px-4 py-2 text-left text-gray-400">Catatan</th>
                  <th className="px-4 py-2 text-center text-gray-400">Status</th>
                </tr>
              </thead>
              <tbody>
                {promises.map((p) => (
                  <tr key={p.id} className="border-b border-gray-800/50">
                    <td className="px-4 py-2 text-white">{p.username}</td>
                    <td className="px-4 py-2 text-gray-300">{new Date(p.promiseDate).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })}</td>
                    <td className="px-4 py-2 text-gray-400">{p.createdByName || '-'}</td>
                    <td className="px-4 py-2 text-gray-400 text-xs">{p.notes || '-'}</td>
                    <td className="px-4 py-2 text-center">
                      <span className={`rounded px-2 py-0.5 text-xs ${
                        p.status === 'active' ? 'bg-yellow-900/50 text-yellow-400' :
                        p.status === 'fulfilled' ? 'bg-green-900/50 text-green-400' :
                        'bg-red-900/50 text-red-400'
                      }`}>
                        {p.status}
                      </span>
                    </td>
                  </tr>
                ))}
                {promises.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                      Tidak ada janji bayar aktif
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
