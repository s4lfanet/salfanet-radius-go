'use client';

import { useState, useCallback, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  RefreshCw,
  Power,
  RotateCcw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Trash2,
  Server,
  Activity,
  ListTodo,
  ChevronDown,
  ChevronUp,
  LayoutList,
  Wifi,
  Globe,
  Save,
  X,
  Edit3,
} from 'lucide-react';
import { DeviceStatusBadge } from '@/components/genieacs/DeviceStatusBadge';
import { TaskStatusBadge } from '@/components/genieacs/TaskStatusBadge';
import { formatWIB, formatFromUTC } from '@/lib/timezone';
import { useToast } from '@/components/cyberpunk/CyberToast';

interface GenieDevice {
  _id: string;
  _lastInform?: string;
  [key: string]: unknown;
}

interface GenieTask {
  _id: string;
  name: string;
  status?: string;
  timestamp?: string;
  fault?: { code: string; message: string };
}

interface WifiConfig {
  index: number;
  ssid: string;
  enable: boolean;
  beaconType: string;
  authMode: string;
  encryptionMode: string;
  keyPassphrase: string;
  channel: string;
  maxBitRate: string;
  standard: string;
  bssid: string;
  totalAssociations: string;
  radioEnabled: string;
}

interface WanInfo {
  username: string;
  externalIPAddress: string;
  connectionStatus: string;
  enable: string;
  vlan: string;
  serviceList: string;
  dnsServers: string;
  defaultGateway: string;
  remoteIPAddress: string;
  pppoeACName: string;
  uptime: string;
  macAddress: string;
  natEnabled: string;
  pppAuthProtocol: string;
}

function getParam(device: GenieDevice, ...paths: string[]): string {
  for (const path of paths) {
    const parts = path.split('.');
    let v: unknown = device;
    for (const p of parts) {
      if (v && typeof v === 'object') v = (v as Record<string, unknown>)[p];
      else { v = undefined; break; }
    }
    if (v && typeof v === 'object' && '_value' in (v as object)) {
      const val = (v as { _value: unknown })._value;
      if (val !== null && val !== undefined && val !== '') return String(val);
    }
    if (v !== null && v !== undefined && typeof v !== 'object') return String(v);
  }
  return '-';
}

function parseWifiConfigs(device: GenieDevice): WifiConfig[] {
  const configs: WifiConfig[] = [];
  try {
    const igd = (device as Record<string, unknown>).InternetGatewayDevice as Record<string, unknown> | undefined;
    if (!igd) return configs;
    const lan = igd.LANDevice as Record<string, unknown> | undefined;
    if (!lan) return configs;
    const lan1 = lan['1'] as Record<string, unknown> | undefined;
    if (!lan1) return configs;
    const wlan = lan1.WLANConfiguration as Record<string, unknown> | undefined;
    if (!wlan) return configs;
    for (let i = 1; i <= 8; i++) {
      const wl = wlan[String(i)] as Record<string, unknown> | undefined;
      if (!wl) continue;
      const getVal = (field: string): string => {
        const p = wl[field] as Record<string, unknown> | undefined;
        if (p && '_value' in p) {
          const v = p._value;
          if (v !== null && v !== undefined && v !== '') return String(v);
        }
        return '';
      };
      const ssid = getVal('SSID');
      if (!ssid) continue;
      configs.push({
        index: i,
        ssid,
        enable: getVal('Enable') === 'true' || getVal('Enable') === 'True',
        beaconType: getVal('BeaconType'),
        authMode: getVal('IEEE11iAuthenticationMode'),
        encryptionMode: getVal('IEEE11iEncryptionModes'),
        keyPassphrase: getVal('KeyPassphrase'),
        channel: getVal('Channel'),
        maxBitRate: getVal('MaxBitRate'),
        standard: getVal('Standard'),
        bssid: getVal('BSSID'),
        totalAssociations: getVal('TotalAssociations'),
        radioEnabled: getVal('RadioEnabled'),
      });
    }
  } catch { /* ignore */ }
  return configs;
}

function parseWanInfo(device: GenieDevice): WanInfo[] {
  const results: WanInfo[] = [];
  try {
    const igd = (device as Record<string, unknown>).InternetGatewayDevice as Record<string, unknown> | undefined;
    if (!igd) return results;
    const wan = igd.WANDevice as Record<string, unknown> | undefined;
    if (!wan) return results;
    const wan1 = wan['1'] as Record<string, unknown> | undefined;
    if (!wan1) return results;
    const wanc = wan1.WANConnectionDevice as Record<string, unknown> | undefined;
    if (!wanc) return results;
    for (let wi = 1; wi <= 4; wi++) {
      const wd = wanc[String(wi)] as Record<string, unknown> | undefined;
      if (!wd) continue;
      const ppp = wd.WANPPPConnection as Record<string, unknown> | undefined;
      if (!ppp) continue;
      for (let ci = 1; ci <= 4; ci++) {
        const conn = ppp[String(ci)] as Record<string, unknown> | undefined;
        if (!conn) continue;
        const g = (f: string): string => {
          const p = conn[f] as Record<string, unknown> | undefined;
          if (p && '_value' in p) {
            const v = p._value;
            if (v !== null && v !== undefined && v !== '') return String(v);
          }
          return '';
        };
        const username = g('Username');
        const status = g('ConnectionStatus');
        if (!username && status !== 'Connected') continue;
        results.push({
          username,
          externalIPAddress: g('ExternalIPAddress'),
          connectionStatus: status,
          enable: g('Enable'),
          vlan: g('X_HW_VLAN'),
          serviceList: g('X_HW_SERVICELIST'),
          dnsServers: g('DNSServers'),
          defaultGateway: g('DefaultGateway'),
          remoteIPAddress: g('RemoteIPAddress'),
          pppoeACName: g('PPPoEACName'),
          uptime: g('Uptime'),
          macAddress: g('MACAddress'),
          natEnabled: g('NATEnabled'),
          pppAuthProtocol: g('PPPAuthenticationProtocol'),
        });
      }
    }
  } catch { /* ignore */ }
  return results;
}

function parseLanInfo(device: GenieDevice) {
  try {
    const igd = (device as Record<string, unknown>).InternetGatewayDevice as Record<string, unknown> | undefined;
    if (!igd) return null;
    const lan = igd.LANDevice as Record<string, unknown> | undefined;
    if (!lan) return null;
    const lan1 = lan['1'] as Record<string, unknown> | undefined;
    if (!lan1) return null;
    const host = lan1.LANHostConfigConfig as Record<string, unknown> | undefined;
    if (!host) return null;
    const g = (f: string): string => {
      const p = host[f] as Record<string, unknown> | undefined;
      if (p && '_value' in p) {
        const v = p._value;
        if (v !== null && v !== undefined && v !== '') return String(v);
      }
      return '';
    };
    return {
      ip: g('IPInterfaceIPAddress'),
      subnet: g('IPInterfaceSubnetMask'),
      dhcpServerEnable: g('DHCPServerEnable'),
      minAddress: g('MinAddress'),
      maxAddress: g('MaxAddress'),
      leaseTime: g('DHCPServerLeaseTime'),
      domain: g('DomainName'),
    };
  } catch { return null; }
}

interface ActionButtonProps {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  loading?: boolean;
  variant?: 'default' | 'danger';
}

function ActionButton({ label, icon, onClick, loading, variant = 'default' }: ActionButtonProps) {
  const cls = variant === 'danger'
    ? 'border-red-300 text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-900/20'
    : 'border-gray-300 text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800';
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className={`flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm disabled:opacity-50 transition-colors ${cls}`}
    >
      {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : icon}
      {label}
    </button>
  );
}

function InfoCard({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
      <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
      <p className={`mt-1 text-sm font-medium text-gray-900 dark:text-white break-all ${mono ? 'font-mono' : ''}`}>
        {value}
      </p>
    </div>
  );
}

export default function DeviceDetailPage() {
  const params = useParams();
  const deviceId = params.deviceId as string;
  const router = useRouter();
  const { addToast, confirm } = useToast();

  const [device, setDevice] = useState<GenieDevice | null>(null);
  const [tasks, setTasks] = useState<GenieTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showRaw, setShowRaw] = useState(false);
  const [showTasks, setShowTasks] = useState(true);
  const [editingWifi, setEditingWifi] = useState<number | null>(null);
  const [editingWan, setEditingWan] = useState<number | null>(null);
  const [wifiSaving, setWifiSaving] = useState(false);
  const [wanSaving, setWanSaving] = useState(false);

  const encodedId = encodeURIComponent(deviceId);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [devRes, taskRes] = await Promise.all([
        fetch(`/api/genieacs/devices/${encodedId}`),
        fetch(`/api/genieacs/devices/${encodedId}/tasks`),
      ]);
      if (!devRes.ok) {
        const j = await devRes.json().catch(() => ({}));
        throw new Error(j.error ?? `HTTP ${devRes.status}`);
      }
      const devJson = await devRes.json();
      setDevice(devJson.data);

      if (taskRes.ok) {
        const taskJson = await taskRes.json();
        setTasks(Array.isArray(taskJson.data) ? taskJson.data : []);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [encodedId]);

  useEffect(() => { load(); }, [load]);

  async function runAction(action: string, label: string, body?: object) {
    setActionLoading(action);
    setError(null);
    setSuccess(null);
    try {
      const res = await fetch(`/api/genieacs/devices/${encodedId}/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
      });
      const j = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(j.error ?? `HTTP ${res.status}`);
      setSuccess(`${label} berhasil dikirim ke device.`);
      load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setActionLoading(null);
    }
  }

  async function handleDelete() {
    if (!await confirm({
      title: 'Delete Device',
      message: 'This will permanently remove the device from GenieACS. Are you sure?',
      confirmText: 'Yes, delete',
      cancelText: 'Cancel',
      variant: 'danger',
    })) return;
    setActionLoading('delete');
    try {
      const res = await fetch(`/api/genieacs/devices/${encodedId}`, { method: 'DELETE' });
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        throw new Error(j.error ?? 'Delete failed');
      }
      addToast({ type: 'success', title: 'Deleted', description: 'Device removed from GenieACS', duration: 3000 });
      router.push('/admin/genieacs/devices');
    } catch (e) {
      setError((e as Error).message);
      setActionLoading(null);
    }
  }

  async function handleWifiSave(wlanIndex: number, ssid: string, password: string, securityMode: string, enabled: boolean) {
    setWifiSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await fetch(`/api/genieacs/devices/${encodedId}/wifi`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wlanIndex, ssid, password, securityMode, enabled }),
      });
      const j = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(j.error ?? `HTTP ${res.status}`);
      setSuccess(`WiFi "${ssid}" berhasil diupdate. Task dikirim ke device.`);
      setEditingWifi(null);
      load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setWifiSaving(false);
    }
  }

  async function handleWanSave(wanIndex: number, connIndex: number, username: string, password: string, vlan: string, serviceList: string, enable: boolean) {
    setWanSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await fetch(`/api/genieacs/devices/${encodedId}/wan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wanIndex, connIndex, username, password, vlan, serviceList, enable }),
      });
      const j = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(j.error ?? `HTTP ${res.status}`);
      setSuccess(`WAN connection berhasil diupdate. Task dikirim ke device.`);
      setEditingWan(null);
      load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setWanSaving(false);
    }
  }

  const serialNumber = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.SerialNumber._value',
    'InternetGatewayDevice.DeviceInfo.SerialNumber',
  ) : '-';
  const manufacturer = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.Manufacturer._value',
    'InternetGatewayDevice.DeviceInfo.Manufacturer',
  ) : '-';
  const model = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.ModelName._value',
    'InternetGatewayDevice.DeviceInfo.ModelName',
  ) : '-';
  const swVersion = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.SoftwareVersion._value',
    'InternetGatewayDevice.DeviceInfo.SoftwareVersion',
  ) : '-';
  const hwVersion = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.HardwareVersion._value',
    'InternetGatewayDevice.DeviceInfo.HardwareVersion',
  ) : '-';
  const uptime = device ? getParam(device,
    'InternetGatewayDevice.DeviceInfo.UpTime._value',
    'InternetGatewayDevice.DeviceInfo.UpTime',
  ) : '-';
  const lastInform = device ? String((device as Record<string, unknown>)._lastInform ?? '') : '';
  const pppoeIP = device ? getParam(device,
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress._value',
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress',
  ) : '-';
  const rxPower = device ? getParam(device,
    'InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.RXPower._value',
  ) : '-';

  const wifiConfigs = device ? parseWifiConfigs(device) : [];
  const wanInfos = device ? parseWanInfo(device) : [];
  const lanInfo = device ? parseLanInfo(device) : null;

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Link
            href="/admin/genieacs/devices"
            className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <ArrowLeft className="h-4 w-4" />
            Devices
          </Link>
          <span className="text-gray-300 dark:text-gray-600">/</span>
          <span className="text-sm font-medium text-gray-900 dark:text-white truncate max-w-xs">
            {serialNumber !== '-' ? serialNumber : deviceId}
          </span>
          {lastInform && <DeviceStatusBadge lastInform={lastInform} />}
        </div>
        <div className="flex flex-wrap gap-2">
          <Link
            href={`/admin/genieacs/devices/${encodeURIComponent(deviceId)}/parameters`}
            className="flex items-center gap-1.5 rounded-md border border-blue-300 bg-blue-50 px-3 py-1.5 text-sm text-blue-700 hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-400 dark:hover:bg-blue-900/40"
          >
            <LayoutList className="h-4 w-4" />
            Browse Parameters
          </Link>
          <ActionButton
            label="Refresh"
            icon={<RefreshCw className="h-4 w-4" />}
            onClick={load}
            loading={loading}
          />
          <ActionButton
            label="Reboot"
            icon={<Power className="h-4 w-4" />}
            onClick={() => runAction('reboot', 'Reboot')}
            loading={actionLoading === 'reboot'}
          />
          <ActionButton
            label="Factory Reset"
            icon={<RotateCcw className="h-4 w-4" />}
            onClick={() => runAction('factory-reset', 'Factory reset')}
            loading={actionLoading === 'factory-reset'}
            variant="danger"
          />
          <ActionButton
            label="Refresh Params"
            icon={<Activity className="h-4 w-4" />}
            onClick={() => runAction('refresh', 'Refresh parameters', { objectName: 'InternetGatewayDevice' })}
            loading={actionLoading === 'refresh'}
          />
          <ActionButton
            label="Delete"
            icon={<Trash2 className="h-4 w-4" />}
            onClick={handleDelete}
            loading={actionLoading === 'delete'}
            variant="danger"
          />
        </div>
      </div>

      {/* Alerts */}
      {error && (
        <div className="flex items-center gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 flex-shrink-0" />
          {error}
        </div>
      )}
      {success && (
        <div className="flex items-center gap-2 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400">
          <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
          {success}
        </div>
      )}

      {/* Device info cards */}
      {device && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <InfoCard label="Serial Number" value={serialNumber} />
          <InfoCard label="Manufacturer" value={manufacturer} />
          <InfoCard label="Model" value={model} />
          <InfoCard label="Software Version" value={swVersion} />
          <InfoCard label="Hardware Version" value={hwVersion} />
          <InfoCard label="PPPoE IP" value={pppoeIP} />
          <InfoCard label="RX Power" value={rxPower} />
          <InfoCard label="Uptime" value={uptime} />
          <InfoCard label="Last Inform" value={lastInform ? formatFromUTC(lastInform) : '-'} />
          <InfoCard label="Device ID" value={deviceId} mono />
        </div>
      )}

      {/* WiFi Configuration */}
      {device && wifiConfigs.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
          <div className="flex items-center gap-2 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <Wifi className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">WiFi Configuration</h3>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {wifiConfigs.map((wifi) => (
              <div key={wifi.index} className="p-4">
                {editingWifi === wifi.index ? (
                  <WifiEditForm
                    wifi={wifi}
                    onSave={(ssid, password, securityMode, enabled) =>
                      handleWifiSave(wifi.index, ssid, password, securityMode, enabled)
                    }
                    onCancel={() => setEditingWifi(null)}
                    saving={wifiSaving}
                  />
                ) : (
                  <div className="flex items-start justify-between gap-4">
                    <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-3 lg:grid-cols-4">
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">WLAN Index</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.index}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">SSID</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.ssid}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Status</span>
                        <p className={`font-medium ${wifi.enable ? 'text-green-600 dark:text-green-400' : 'text-gray-400'}`}>
                          {wifi.enable ? 'Enabled' : 'Disabled'}
                        </p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Security</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.beaconType || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Auth Mode</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.authMode || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Encryption</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.encryptionMode || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Password</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.keyPassphrase ? '••••••••' : '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Channel</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.channel || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Standard</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.standard || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">BSSID</span>
                        <p className="font-mono text-xs text-gray-900 dark:text-white">{wifi.bssid || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Associated</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wifi.totalAssociations || '0'}</p>
                      </div>
                    </div>
                    <button
                      onClick={() => setEditingWifi(wifi.index)}
                      className="flex items-center gap-1 rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800 flex-shrink-0"
                    >
                      <Edit3 className="h-3 w-3" />
                      Edit
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* WAN Configuration */}
      {device && wanInfos.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
          <div className="flex items-center gap-2 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <Globe className="h-5 w-5 text-green-600 dark:text-green-400" />
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">WAN Configuration</h3>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {wanInfos.map((wan, i) => (
              <div key={i} className="p-4">
                {editingWan === i ? (
                  <WanEditForm
                    wan={wan}
                    wanIndex={i + 1}
                    connIndex={1}
                    onSave={(username, password, vlan, serviceList, enable) =>
                      handleWanSave(i + 1, 1, username, password, vlan, serviceList, enable)
                    }
                    onCancel={() => setEditingWan(null)}
                    saving={wanSaving}
                  />
                ) : (
                  <div className="flex items-start justify-between gap-4">
                    <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-3 lg:grid-cols-4">
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Username (PPPoE)</span>
                        <p className="font-medium text-gray-900 dark:text-white break-all">{wan.username || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">IP Address</span>
                        <p className="font-mono text-sm text-gray-900 dark:text-white">{wan.externalIPAddress || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Status</span>
                        <p className={`font-medium ${wan.connectionStatus === 'Connected' ? 'text-green-600 dark:text-green-400' : 'text-gray-400'}`}>
                          {wan.connectionStatus || '-'}
                        </p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Enable</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.enable || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">VLAN</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.vlan || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Service List</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.serviceList || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">DNS</span>
                        <p className="font-mono text-xs text-gray-900 dark:text-white">{wan.dnsServers || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Gateway</span>
                        <p className="font-mono text-xs text-gray-900 dark:text-white">{wan.defaultGateway || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">AC Name</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.pppoeACName || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">MAC Address</span>
                        <p className="font-mono text-xs text-gray-900 dark:text-white">{wan.macAddress || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">NAT</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.natEnabled || '-'}</p>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Auth Protocol</span>
                        <p className="font-medium text-gray-900 dark:text-white">{wan.pppAuthProtocol || '-'}</p>
                      </div>
                    </div>
                    <button
                      onClick={() => setEditingWan(i)}
                      className="flex items-center gap-1 rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800 flex-shrink-0"
                    >
                      <Edit3 className="h-3 w-3" />
                      Edit
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* LAN Configuration */}
      {device && lanInfo && (
        <div className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
          <div className="flex items-center gap-2 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <Server className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">LAN Configuration</h3>
          </div>
          <div className="p-4">
            <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-3 lg:grid-cols-4">
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">LAN IP</span>
                <p className="font-mono text-sm text-gray-900 dark:text-white">{lanInfo.ip || '-'}</p>
              </div>
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">Subnet Mask</span>
                <p className="font-mono text-sm text-gray-900 dark:text-white">{lanInfo.subnet || '-'}</p>
              </div>
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">DHCP Server</span>
                <p className="font-medium text-gray-900 dark:text-white">{lanInfo.dhcpServerEnable || '-'}</p>
              </div>
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">DHCP Range</span>
                <p className="font-mono text-xs text-gray-900 dark:text-white">
                  {lanInfo.minAddress && lanInfo.maxAddress ? `${lanInfo.minAddress} - ${lanInfo.maxAddress}` : '-'}
                </p>
              </div>
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">Lease Time</span>
                <p className="font-medium text-gray-900 dark:text-white">{lanInfo.leaseTime || '-'}</p>
              </div>
              <div>
                <span className="text-xs text-gray-500 dark:text-gray-400">Domain</span>
                <p className="font-medium text-gray-900 dark:text-white">{lanInfo.domain || '-'}</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tasks */}
      <div className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
        <button
          className="flex w-full items-center justify-between px-4 py-3"
          onClick={() => setShowTasks((v) => !v)}
        >
          <span className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
            <ListTodo className="h-4 w-4" />
            Tasks ({tasks.length})
          </span>
          {showTasks ? <ChevronUp className="h-4 w-4 text-gray-400" /> : <ChevronDown className="h-4 w-4 text-gray-400" />}
        </button>
        {showTasks && (
          <div className="border-t border-gray-200 dark:border-gray-700">
            {tasks.length === 0 ? (
              <p className="px-4 py-3 text-sm text-gray-500">No tasks pending.</p>
            ) : (
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700 text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800">
                  <tr>
                    {['Task', 'Status', 'Created'].map((h) => (
                      <th key={h} className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                  {tasks.map((task) => (
                    <tr key={task._id}>
                      <td className="px-4 py-2 font-mono text-xs">{task.name}</td>
                      <td className="px-4 py-2">
                        <TaskStatusBadge status={task.status ?? 'pending'} />
                      </td>
                      <td className="px-4 py-2 text-gray-500 text-xs">
                        {task.timestamp ? formatFromUTC(task.timestamp) : '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Raw JSON toggle */}
      {device && (
        <div className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
          <button
            className="flex w-full items-center justify-between px-4 py-3"
            onClick={() => setShowRaw((v) => !v)}
          >
            <span className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
              <Server className="h-4 w-4" />
              Raw NBI Data
            </span>
            {showRaw ? <ChevronUp className="h-4 w-4 text-gray-400" /> : <ChevronDown className="h-4 w-4 text-gray-400" />}
          </button>
          {showRaw && (
            <div className="border-t border-gray-200 p-4 dark:border-gray-700">
              <pre className="overflow-auto max-h-96 text-xs text-gray-700 dark:text-gray-300">
                {JSON.stringify(device, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function WifiEditForm({
  wifi,
  onSave,
  onCancel,
  saving,
}: {
  wifi: WifiConfig;
  onSave: (ssid: string, password: string, securityMode: string, enabled: boolean) => void;
  onCancel: () => void;
  saving: boolean;
}) {
  const [ssid, setSsid] = useState(wifi.ssid);
  const [password, setPassword] = useState(wifi.keyPassphrase || '');
  const [securityMode, setSecurityMode] = useState(
    wifi.beaconType === 'None' ? 'Open' :
    wifi.beaconType === '11i' ? 'WPA2-PSK' :
    wifi.beaconType === 'WPA' ? 'WPA-PSK' :
    wifi.beaconType === 'WPAand11i' ? 'WPA-WPA2-PSK' : 'WPA2-PSK'
  );
  const [enabled, setEnabled] = useState(wifi.enable);

  return (
    <div className="space-y-3 rounded-lg border border-blue-200 bg-blue-50/50 p-4 dark:border-blue-800 dark:bg-blue-900/10">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">SSID</label>
          <input
            type="text"
            value={ssid}
            onChange={(e) => setSsid(e.target.value)}
            maxLength={32}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">Security Mode</label>
          <select
            value={securityMode}
            onChange={(e) => setSecurityMode(e.target.value)}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          >
            <option value="Open">Open (No Password)</option>
            <option value="WPA2-PSK">WPA2-PSK (Recommended)</option>
            <option value="WPA-PSK">WPA-PSK</option>
            <option value="WPA-WPA2-PSK">WPA/WPA2-PSK (Mixed)</option>
          </select>
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">
            Password {securityMode === 'Open' && '(not required)'}
          </label>
          <input
            type="text"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={securityMode === 'Open'}
            placeholder={securityMode === 'Open' ? 'N/A' : '8-63 characters'}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">Enabled</label>
          <div className="mt-1">
            <button
              type="button"
              onClick={() => setEnabled(!enabled)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                enabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                enabled ? 'translate-x-6' : 'translate-x-1'
              }`} />
            </button>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={() => onSave(ssid, password, securityMode, enabled)}
          disabled={saving || !ssid}
          className="flex items-center gap-1.5 rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          Save
        </button>
        <button
          onClick={onCancel}
          className="flex items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
      </div>
    </div>
  );
}

function WanEditForm({
  wan,
  onSave,
  onCancel,
  saving,
}: {
  wan: WanInfo;
  wanIndex: number;
  connIndex: number;
  onSave: (username: string, password: string, vlan: string, serviceList: string, enable: boolean) => void;
  onCancel: () => void;
  saving: boolean;
}) {
  const [username, setUsername] = useState(wan.username || '');
  const [password, setPassword] = useState('');
  const [vlan, setVlan] = useState(wan.vlan || '');
  const [serviceList, setServiceList] = useState(wan.serviceList || '');
  const [enable, setEnable] = useState(wan.enable === 'true' || wan.enable === 'True');

  return (
    <div className="space-y-3 rounded-lg border border-green-200 bg-green-50/50 p-4 dark:border-green-800 dark:bg-green-900/10">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">PPPoE Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">PPPoE Password</label>
          <input
            type="text"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Leave empty to keep current"
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">VLAN</label>
          <input
            type="text"
            value={vlan}
            onChange={(e) => setVlan(e.target.value)}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">Service List</label>
          <input
            type="text"
            value={serviceList}
            onChange={(e) => setServiceList(e.target.value)}
            placeholder="e.g. TR069_INTERNET"
            className="mt-1 w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-600 dark:text-gray-400">Enable</label>
          <div className="mt-1">
            <button
              type="button"
              onClick={() => setEnable(!enable)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                enable ? 'bg-green-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                enable ? 'translate-x-6' : 'translate-x-1'
              }`} />
            </button>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={() => onSave(username, password, vlan, serviceList, enable)}
          disabled={saving}
          className="flex items-center gap-1.5 rounded-md bg-green-600 px-3 py-1.5 text-sm text-white hover:bg-green-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          Save
        </button>
        <button
          onClick={onCancel}
          className="flex items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
      </div>
    </div>
  );
}
