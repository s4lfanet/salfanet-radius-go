'use client';

import { useState, useEffect, useCallback, use, useMemo, useRef } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { formatWIB } from '@/lib/timezone';
import { Switch } from '@/components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  LineChart, Line, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import {
  Server, RefreshCw, AlertCircle, Wifi, WifiOff,
  Thermometer, Clock, Activity, ArrowLeft, Save, TestTube,
  Power, Download, CheckCircle, Signal, Plus, X, Cpu, Zap,
  Eye, UserPlus, Trash2, Pencil, Eraser,
} from 'lucide-react';

interface ONU {
  id: string;
  frame: number;
  slot: number;
  port: number;
  onuId: number;
  onuType?: string | null;
  serialNumber: string | null;
  macAddress: string | null;
  status: string;
  description: string | null;
  rxPower: number | null;
  txPower: number | null;
  temperature: number | null;
  distance: number | null;
  lastSeenAt: string | null;
  customer: { id: string; username: string; name: string; phone: string } | null;
  odpId?: string | null;
  odpName?: string | null;
}

interface OLTDetail {
  id: string;
  name: string;
  ipAddress: string;
  vendor: string | null;
  model: string | null;
  firmwareVersion: string | null;
  isOnline: boolean;
  temperature: number | null;
  uptime: number | null;
  totalOnu: number;
  onlineOnu: number;
  offlineOnu: number;
  lastPollAt: string | null;
  monitoringEnabled: boolean;
  snmpEnabled: boolean;
  snmpCommunity: string;
  snmpPort: number;
  sshEnabled: boolean;
  sshPort: number;
  telnetEnabled: boolean;
  telnetPort: number;
  username: string | null;
  pollingInterval: number;
  onuStatuses: ONU[];
  alerts: any[];
  performanceMetrics: any[];
  monitoringLogs: any[];
  routers: { id: string; routerId: string; router: { id: string; name: string; ipAddress: string } }[];
}

// -----------------------------------------------------------------------------
// ZTE C320 Realistic Chassis Diagram
// -----------------------------------------------------------------------------

/** Card type visual metadata */
const CARD_META: Record<string, { label: string; color: string; portRows: number; portCols: number }> = {
  MCUD1:   { label: 'MCUD1',   color: '#2563eb', portRows: 0, portCols: 0 },
  MCUD:    { label: 'MCUD',    color: '#2563eb', portRows: 0, portCols: 0 },
  GTGQ:    { label: 'GTGQ',    color: '#15803d', portRows: 4, portCols: 4 },  // 16-port GPON
  GTGH:    { label: 'GTGH',    color: '#15803d', portRows: 2, portCols: 4 },  // 8-port GPON
  GTGO:    { label: 'GTGO',    color: '#15803d', portRows: 1, portCols: 4 },  // 4-port GPON
  GTGHG:   { label: 'GTGHG',   color: '#16a34a', portRows: 4, portCols: 4 },  // 16-port GPON (ZTE C320)
  'SMXA-B':{ label: 'SMXA-B',  color: '#1d4ed8', portRows: 2, portCols: 3 },  // 5-port uplink (3 GE + 2 10GE)
  'SMXA-A':{ label: 'SMXA-A',  color: '#1d4ed8', portRows: 1, portCols: 2 },  // 2x 10GE uplink
  SMXA:    { label: 'SMXA',    color: '#1d4ed8', portRows: 1, portCols: 4 },  // generic SMXA uplink
  GICF:    { label: 'GICF',    color: '#1e40af', portRows: 2, portCols: 2 },  // 4-port uplink
  GISF:    { label: 'GISF',    color: '#1e40af', portRows: 1, portCols: 4 },
  empty:   { label: 'EMPTY',   color: '#374151', portRows: 0, portCols: 0 },
};

interface ApiChassisSlot {
  index: number;
  label: string;
  type: string;
  present: boolean;
  cardType: string;
  hardVer?: string;
  softVer?: string;
  cardStatus?: string;
  portCount: number;
  ports: Array<{
    port: number;
    iface?: string;
    onuCount: number;
    onlineCount: number;
    hasOnus: boolean;
    adminStatus?: string;
    linkStatus?: string;
    speed?: string;
    physicalType?: string;
    description?: string;
    isEnabled?: boolean;
    isLinked?: boolean;
  }>;
  uplinkIfaces?: string[];
  description?: string;
}

// -- Uplink Port Detail Modal -------------------------------------------------
function UplinkPortModal({ oltId, port, onClose }: { oltId: string; port: string; onClose: () => void }) {
  const [activeTab, setActiveTab] = useState<'status' | 'vlan' | 'config' | 'optical'>('status');
  const [tabData, setTabData] = useState<{ raw: string; parsed: Record<string, string> } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newVlanId, setNewVlanId] = useState('');
  const [vlanMode, setVlanMode] = useState<'tag' | 'access'>('tag');
  const [actionLoading, setActionLoading] = useState(false);
  const [actionMsg, setActionMsg] = useState<string | null>(null);

  const fetchTab = useCallback(async (tab: string) => {
    setLoading(true); setError(null); setTabData(null);
    try {
      const res = await fetch(`/api/olt/${oltId}/uplink?port=${encodeURIComponent(port)}&tab=${tab}`);
      const json = await res.json();
      if (json.success) setTabData(json.data);
      else setError(json.error ?? 'Failed to load');
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [oltId, port]);

  useEffect(() => { fetchTab(activeTab); }, [activeTab, fetchTab]);

  const doAction = async (action: string, extra: Record<string, any> = {}) => {
    setActionLoading(true); setActionMsg(null);
    try {
      const res = await fetch(`/api/olt/${oltId}/uplink`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port, action, ...extra }),
      });
      const json = await res.json();
      setActionMsg(json.success ? '[OK] Success' : `Error: ${json.error}`);
      if (json.success) fetchTab(activeTab);
    } catch (e: any) { setActionMsg(`Error: ${e.message}`); }
    finally { setActionLoading(false); }
  };

  const parsed = tabData?.parsed ?? {};
  const raw = tabData?.raw ?? '';

  const renderStatus = () => {
    const linkStatus = parsed['Link Status'] ?? 'Unknown';
    const adminStatus = parsed['Admin Status'] ?? 'Unknown';
    const isEnabled = /up|enable|activate/i.test(adminStatus);
    const isUp = /up|online/i.test(linkStatus);
    const statusTone = !isEnabled
      ? { className: 'bg-gray-100 dark:bg-gray-900 border border-gray-300 dark:border-slate-600', dot: 'bg-slate-400 dark:bg-slate-500', text: 'text-slate-600 dark:text-slate-300', label: 'Disabled' }
      : isUp
        ? { className: 'bg-green-50 dark:bg-green-950/40 border border-green-300 dark:border-green-800', dot: 'bg-green-500 dark:bg-green-400 animate-pulse', text: 'text-green-700 dark:text-green-400', label: 'Online' }
        : { className: 'bg-amber-50 dark:bg-amber-950/40 border border-amber-300 dark:border-amber-700', dot: 'bg-amber-500 dark:bg-amber-400', text: 'text-amber-700 dark:text-amber-300', label: 'Admin UP / Link DOWN' };

    return (
      <div className="space-y-4">
        <div className={`flex items-center gap-3 p-3 rounded-lg ${statusTone.className}`}>
          <div className={`w-3 h-3 rounded-full ${statusTone.dot}`} />
          <div>
            <div className={`text-sm font-bold ${statusTone.text}`}>{statusTone.label}</div>
            <div className="text-xs text-slate-500 dark:text-slate-400">Admin: {adminStatus} . Port: {linkStatus} . {parsed['Speed'] ?? '--'}</div>
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {(['Speed','Duplex','Flow Control','Physical Type','MTU','MAC'] as const).map(k => parsed[k] ? (
            <div key={k} className="p-2 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700">
              <div className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider">{k}</div>
              <div className="text-xs font-mono text-slate-900 dark:text-white mt-0.5 break-all">{parsed[k]}</div>
            </div>
          ) : null)}
        </div>
        <div className="flex flex-col sm:flex-row gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
          {isEnabled ? (
            <button onClick={() => doAction('disable')} disabled={actionLoading} className="px-3 py-1.5 text-xs rounded bg-red-900 hover:bg-red-800 text-white disabled:opacity-50">Disable Port</button>
          ) : (
            <button onClick={() => doAction('enable')} disabled={actionLoading} className="px-3 py-1.5 text-xs rounded bg-green-800 hover:bg-green-700 text-white disabled:opacity-50">Enable Port</button>
          )}
        </div>
      </div>
    );
  };

  const renderVlan = () => {
    const taggedVlans = (parsed['Tagged Vlan'] ?? '').split(/[\s,]+/).filter(Boolean);
    const pvid = parsed['Pvid'] ?? '';
    const mode = parsed['Mode'] ?? '--';
    return (
      <div className="space-y-4">
        {/* Info row */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
          {([['Mode', mode], ['TLS', parsed['TLS'] ?? '--']] as [string,string][]).map(([label, val]) => (
            <div key={label} className="p-2 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-center">
              <div className="text-[10px] text-slate-500 dark:text-slate-400 uppercase">{label}</div>
              <div className="text-sm font-bold text-slate-900 dark:text-white mt-0.5 break-all">{val}</div>
            </div>
          ))}
          {/* PVID with inline edit/remove */}
          <div className="p-2 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700">
            <div className="text-[10px] text-slate-500 dark:text-slate-400 uppercase mb-1">PVID (Access)</div>
            {pvid && pvid !== '--' ? (
              <div className="flex items-center gap-1.5">
                <span className="text-sm font-bold font-mono text-amber-400">{pvid}</span>
                <button onClick={() => doAction('removePvid')} disabled={actionLoading}
                  title="Remove PVID"
                  className="ml-auto px-1.5 py-0.5 text-[10px] rounded bg-red-900/60 hover:bg-red-800 text-red-300 border border-red-700 disabled:opacity-50">remove</button>
              </div>
            ) : (
              <span className="text-xs text-slate-500 italic">None</span>
            )}
          </div>
        </div>
        {/* Tagged VLANs */}
        <div>
          <div className="text-xs text-slate-500 dark:text-slate-400 mb-2 uppercase tracking-wider">Tagged VLANs</div>
          <div className="flex flex-wrap gap-1.5 min-h-8">
            {taggedVlans.length > 0 ? taggedVlans.map(v => (
              <div key={v} className="flex items-center gap-1 px-2 py-0.5 rounded bg-blue-950 border border-blue-700 text-xs font-mono text-blue-300">
                {v}
                <button onClick={() => doAction('removeVlan', { vlanId: v })} disabled={actionLoading}
                  title={`Remove VLAN ${v}`}
                  className="ml-1 w-4 h-4 flex items-center justify-center rounded-full bg-blue-900 hover:bg-red-800 text-slate-400 hover:text-red-200 leading-none font-bold text-[10px] border border-blue-700 hover:border-red-700 disabled:opacity-40 transition-colors">x</button>
              </div>
            )) : <span className="text-xs text-gray-600 italic">No tagged VLANs configured</span>}
          </div>
        </div>
        {/* Add VLAN / Set PVID */}
        <div className="flex flex-col sm:flex-row gap-2 items-stretch sm:items-center pt-3 border-t border-slate-200 dark:border-slate-800">
          <input value={newVlanId} onChange={e => setNewVlanId(e.target.value.replace(/\D/g, ''))} placeholder="VLAN ID" maxLength={4}
            className="w-full sm:w-20 px-2 py-1 text-xs rounded bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 text-slate-900 dark:text-white" />
          <select value={vlanMode} onChange={e => setVlanMode(e.target.value as 'tag' | 'access')}
            className="w-full sm:w-auto px-2 py-1 text-xs rounded bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 text-slate-900 dark:text-white">
            <option value="tag">Tagged (Trunk)</option>
            <option value="access">Set as PVID</option>
          </select>
          <button onClick={() => {
              vlanMode === 'access'
                ? doAction('setPvid', { vlanId: newVlanId })
                : doAction('addVlan', { vlanId: newVlanId, mode: 'tag' });
              setNewVlanId('');
            }}
            disabled={!newVlanId || actionLoading}
            className="px-3 py-1 text-xs rounded bg-blue-700 hover:bg-blue-600 text-white disabled:opacity-50">
            {vlanMode === 'access' ? 'Set PVID' : 'Add VLAN'}
          </button>
        </div>
      </div>
    );
  };

  const renderConfig = () => (
    <pre className="text-[11px] font-mono text-emerald-700 dark:text-emerald-300 bg-slate-50 dark:bg-slate-950 rounded-lg p-3 overflow-auto max-h-64 whitespace-pre-wrap border border-slate-200 dark:border-slate-800">
      {raw || '(No configuration data)'}
    </pre>
  );

  const renderOptical = () => {
    const metrics: [string, string, string][] = [
      ['TX Power', parsed['TX Power'] ?? '--', '#22c55e'],
      ['RX Power', parsed['RX Power'] ?? '--', '#3b82f6'],
      ['Temperature', parsed['Temperature'] ?? '--', '#f59e0b'],
      ['Supply Voltage', parsed['Supply Voltage'] ?? '--', '#a855f7'],
      ['TX Bias Current', parsed['TX Bias Current'] ?? '--', '#06b6d4'],
    ];
    const specs: [string, string][] = [
      ['Vendor', parsed['Vendor'] ?? parsed['Manufacturer Name'] ?? '--'],
      ['Part No.', parsed['Part Number'] ?? '--'],
      ['Serial No.', parsed['Serial Number'] ?? '--'],
      ['Wavelength (nm)', parsed['Wavelength'] ?? '--'],
      ['Fiber Type', parsed['Fiber Type'] ?? '--'],
      ['Connector', parsed['Connector Type'] ?? '--'],
    ];
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {metrics.map(([label, val, accent]) => (
            <div key={label} className="p-2.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700" style={{ borderLeft: `3px solid ${accent}` }}>
              <div className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wider">{label}</div>
              <div className="text-base font-bold font-mono text-slate-900 dark:text-white mt-0.5 break-all">{val}</div>
            </div>
          ))}
        </div>
        <div className="border-t border-slate-200 dark:border-slate-800 pt-3">
          <div className="text-xs text-slate-500 dark:text-slate-400 mb-2 uppercase tracking-wider">Module Specifications</div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {specs.map(([label, val]) => (
              <div key={label} className="p-1.5 rounded bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800">
                <div className="text-[9px] text-slate-500 uppercase">{label}</div>
                <div className="text-xs font-mono text-slate-800 dark:text-slate-200 mt-0.5 break-all">{val}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  };

  const TABS = ['status', 'vlan', 'config', 'optical'] as const;
  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-2 sm:p-4" style={{ background: 'rgba(2,6,23,0.72)' }}>
      <div className="w-full max-w-2xl rounded-xl shadow-2xl bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 max-h-[92vh] overflow-hidden">
        <div className="flex items-center justify-between px-4 sm:px-5 py-3.5 border-b border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
            <span className="text-sm font-bold text-slate-900 dark:text-white font-mono break-all">{port}</span>
            <span className="text-xs text-slate-500">Uplink Port</span>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-900 dark:hover:text-white text-xl leading-none">x</button>
        </div>
        <div className="grid grid-cols-4 border-b border-slate-200 dark:border-slate-800">
          {TABS.map(t => (
            <button key={t} onClick={() => setActiveTab(t)}
              className={`py-2.5 text-[11px] sm:text-xs font-semibold uppercase tracking-wider transition-colors ${activeTab === t ? 'text-blue-500 border-b-2 border-blue-500 bg-blue-50 dark:bg-blue-950/20' : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'}`}>
              {t}
            </button>
          ))}
        </div>
        <div className="p-4 sm:p-5 overflow-y-auto max-h-[calc(92vh-7rem)]">
          {loading && <div className="text-center py-8 text-slate-500 text-sm">Loading...</div>}
          {error && <div className="text-red-500 dark:text-red-400 text-sm p-3 bg-red-50 dark:bg-red-950/30 rounded-lg border border-red-200 dark:border-red-900">{error}</div>}
          {!loading && !error && tabData && (
            <>
              {activeTab === 'status'  && renderStatus()}
              {activeTab === 'vlan'    && renderVlan()}
              {activeTab === 'config'  && renderConfig()}
              {activeTab === 'optical' && renderOptical()}
            </>
          )}
          {actionMsg && (
            <div className={`mt-3 text-xs p-2 rounded ${actionMsg.startsWith('[OK]') ? 'bg-green-950/40 text-green-400 border border-green-900' : 'bg-red-950/40 text-red-400 border border-red-900'}`}>
              {actionMsg}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// -- PON Port Stat types -------------------------------------------------------
interface PONPortStat {
  slot: number;
  port: number;
  iface: string;
  adminStatus: string;
  lineProto: string;
  description?: string;
  inputBps?: number;
  outputBps?: number;
  inputPct?: number;
  outputPct?: number;
  temperature?: number;
  txPower?: number;
  rxPower?: number;
  voltage?: number;
  biasCurrent?: number;
}

// -- ZTE Chassis View ---------------------------------------------------------
function ZTEChassisView({ olt, onus: onusProp, onuLastRefresh }: { olt: OLTDetail; onus?: ONU[]; onuLastRefresh?: Date | null }) {
  const [chassisSlots, setChassisSlots] = useState<ApiChassisSlot[]>([]);
  const [selectedUplinkPort, setSelectedUplinkPort] = useState<string | null>(null);
  const [loadingChassis, setLoadingChassis] = useState(false);
  const [expandedPON, setExpandedPON] = useState<string | null>(null);
  const [ponStatCache, setPonStatCache] = useState<Record<string, PONPortStat | null>>({});
  const [loadingPON, setLoadingPON] = useState<string | null>(null);
  const [ponActing, setPonActing] = useState<string | null>(null); // portKey currently acting
  const [editingPONDesc, setEditingPONDesc] = useState<{ portKey: string; value: string } | null>(null);
  const [selectedPON, setSelectedPON] = useState<{ slot: number; port: number } | null>(null);

  const fetchChassis = useCallback(() => {
    setLoadingChassis(true);
    fetch(`/api/olt/${olt.id}/chassis`)
      .then(r => r.json())
      .then(j => { if (j.success && j.chassis) setChassisSlots(j.chassis); })
      .catch(() => {})
      .finally(() => setLoadingChassis(false));
  }, [olt.id]);

  useEffect(() => {
    fetchChassis();
  }, [fetchChassis]);

  const fetchPONStat = useCallback((portKey: string) => {
    // portKey is "slot/port" e.g. "1/1"
    if (ponStatCache[portKey] !== undefined || loadingPON === portKey) return;
    const [slot, port] = portKey.split('/');
    setLoadingPON(portKey);
    fetch(`/api/olt/${olt.id}/pon-stat?slot=${slot}&port=${port}`)
      .then(r => r.json())
      .then(j => {
        setPonStatCache(prev => ({ ...prev, [portKey]: j.success ? j.stat : null }));
      })
      .catch(() => {
        setPonStatCache(prev => ({ ...prev, [portKey]: null }));
      })
      .finally(() => setLoadingPON(null));
  }, [olt.id, ponStatCache, loadingPON]);

  const refreshPONStat = useCallback((portKey: string) => {
    setPonStatCache(prev => { const n = { ...prev }; delete n[portKey]; return n; });
  }, []);

  // Keep a ref to always have the latest fetchPONStat (avoids stale closure in setTimeout)
  const fetchPONStatRef = useRef<(portKey: string) => void>(() => {});
  useEffect(() => { fetchPONStatRef.current = fetchPONStat; }, [fetchPONStat]);

  const handlePONAction = useCallback(async (portKey: string, action: string, description?: string) => {
    const [slotStr, portStr] = portKey.split('/');
    setPonActing(portKey);
    try {
      const res = await fetch(`/api/olt/${olt.id}/pon`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot: parseInt(slotStr), port: parseInt(portStr), action, description }),
      });
      if (res.ok) {
        refreshPONStat(portKey);
        // Use ref so the latest fetchPONStat (with cleared cache) is called after React re-renders
        setTimeout(() => fetchPONStatRef.current(portKey), 1500);
      }
    } catch {
      // ignore
    } finally {
      setPonActing(null);
    }
  }, [olt.id, refreshPONStat]);

  // -- Port stats from ONU list ----------------------------------------------
  // Prefer live ONU data (auto-refreshes every 15s) over stale olt.onuStatuses
  const onusForPortMap = onusProp ?? olt.onuStatuses;
  const portStats: Record<string, { total: number; online: number; offline: number; los: number; dyingGasp: number; unregistered: number; rxPowers: number[] }> = {};
  for (const onu of onusForPortMap) {
    const key = `${onu.slot}/${onu.port}`;
    if (!portStats[key]) portStats[key] = { total: 0, online: 0, offline: 0, los: 0, dyingGasp: 0, unregistered: 0, rxPowers: [] };
    portStats[key].total++;
    if (onu.status === 'online') portStats[key].online++;
    else portStats[key].offline++;
    if (onu.status === 'auth_failed') portStats[key].unregistered++;
    if (onu.status === 'los') portStats[key].los++;
    if (onu.status === 'dying_gasp') portStats[key].dyingGasp++;
    if (onu.rxPower !== null) portStats[key].rxPowers.push(onu.rxPower);
  }

  // -- Build visible slot list -----------------------------------------------
  let visibleSlots: ApiChassisSlot[];
  if (chassisSlots.length > 0) {
    visibleSlots = chassisSlots;
  } else {
    // Fallback: derive service slots from ONU statuses
    const maxPortPerSlot: Record<number, number> = {};
    for (const onu of onusForPortMap) {
      if (maxPortPerSlot[onu.slot] === undefined || onu.port > maxPortPerSlot[onu.slot])
        maxPortPerSlot[onu.slot] = onu.port;
    }
    const serviceSlots: ApiChassisSlot[] = Object.keys(maxPortPerSlot).map(sk => {
      const slotIdx = parseInt(sk);
      const portCount = Math.max(maxPortPerSlot[slotIdx] + 1, 16);
      const cardType = portCount <= 4 ? 'GTGO' : portCount <= 8 ? 'GTGH' : 'GTGQ';
      return {
        index: slotIdx, label: `${slotIdx}`, type: 'service', present: true, cardType, portCount,
        ports: Array.from({ length: portCount }, (_, i) => ({ port: i, onuCount: 0, onlineCount: 0, hasOnus: false })),
      };
    });
    visibleSlots = [
      ...serviceSlots,
      { index: 15, label: 'UPL-A', type: 'uplink', present: true,          cardType: 'GICF',  portCount: 4, ports: Array.from({ length: 4 }, (_, i) => ({ port: i, iface: `xgei_1/15/${i + 1}`, onuCount: 0, onlineCount: 0, hasOnus: false })) },
    ].sort((a, b) => a.index - b.index);
  }

  const diagramSlots = useMemo(() => {
    const nonMcuSlots = visibleSlots.filter((slot) => slot.type !== 'mcud');
    if (nonMcuSlots.length === 0) return [] as ApiChassisSlot[];

    const slotMap = new Map(nonMcuSlots.map((slot) => [slot.index, slot]));
    const minSlot = Math.min(...nonMcuSlots.map((slot) => slot.index));
    const maxSlot = Math.max(...nonMcuSlots.map((slot) => slot.index));

    const rows: ApiChassisSlot[] = [];
    for (let slotIndex = minSlot; slotIndex <= maxSlot; slotIndex++) {
      const existing = slotMap.get(slotIndex);
      rows.push(existing ?? {
        index: slotIndex,
        label: `S${slotIndex}`,
        type: 'empty',
        present: false,
        cardType: 'empty',
        portCount: 0,
        ports: [],
      });
    }

    return rows;
  }, [visibleSlots]);

  const activeCardsCount = diagramSlots.filter((slot) => slot.present && slot.type !== 'empty').length;

  const formatUptime = (secs: number | null) => {
    if (!secs) return 'N/A';
    const d = Math.floor(secs / 86400), h = Math.floor((secs % 86400) / 3600), m = Math.floor((secs % 3600) / 60);
    return `${d}d ${h}h ${m}m`;
  };

  const portColor = (slotIdx: number, portIdx: number) => {
    const s = portStats[`${slotIdx}/${portIdx}`];
    if (!s || s.total === 0) return { bg: '#1e293b', border: '#334155', dot: '#475569' };
    if (s.online === s.total)  return { bg: '#14532d', border: '#16a34a', dot: '#4ade80' };
    if (s.los > 0)             return { bg: '#450a0a', border: '#dc2626', dot: '#f87171' };
    if (s.dyingGasp > 0)       return { bg: '#7c2d12', border: '#f97316', dot: '#fb923c' };
    if (s.unregistered > 0 && s.online === 0) return { bg: '#713f12', border: '#ca8a04', dot: '#facc15' };
    if (s.online === 0)        return { bg: '#450a0a', border: '#dc2626', dot: '#f87171' };
    return { bg: '#431407', border: '#ea580c', dot: '#fb923c' };
  };

  const portTooltip = (slotIdx: number, portIdx: number) => {
    const s = portStats[`${slotIdx}/${portIdx}`];
    const avgRx = s?.rxPowers.length ? (s.rxPowers.reduce((a, b) => a + b) / s.rxPowers.length).toFixed(1) : null;
    return s
      ? [
          `PON 0/${slotIdx}/${portIdx}`,
          `Total ONU: ${s.total}`,
          `Online: ${s.online}`,
          `Offline: ${s.offline}`,
          `LOS: ${s.los}`,
          `Dying Gasp: ${s.dyingGasp}`,
          `Unconfig: ${s.unregistered}`,
          ...(avgRx ? [`Avg RX: ${avgRx} dBm`] : []),
        ].join('\n')
      : `PON 0/${slotIdx}/${portIdx}\n(No ONU)`;
  };

  const getUplinkPortVisual = (port: ApiChassisSlot['ports'][number]) => {
    if (port.isEnabled === false) {
      return { bg: '#111827', border: '#475569', dot: '#64748b', text: '#cbd5e1', state: 'DIS' };
    }
    if (port.isEnabled && port.isLinked) {
      return { bg: '#14532d', border: '#16a34a', dot: '#4ade80', text: '#dcfce7', state: 'UP' };
    }
    if (port.isEnabled) {
      return { bg: '#451a03', border: '#f59e0b', dot: '#fbbf24', text: '#fde68a', state: 'DOWN' };
    }
    return { bg: '#0f172a', border: '#334155', dot: '#475569', text: '#94a3b8', state: 'UNK' };
  };

  const uplinkTooltip = (slot: ApiChassisSlot, port: ApiChassisSlot['ports'][number], portNumber: number) => {
    return [
      `${slot.cardType}`,
      `Port ${portNumber}: ${port.linkStatus ?? 'Unknown'}`,
      `Admin: ${port.adminStatus ?? 'Unknown'}`,
      ...(port.speed ? [`Speed: ${port.speed}`] : []),
      ...(port.physicalType ? [`Type: ${port.physicalType}`] : []),
      ...(port.iface ? [`Iface: ${port.iface}`] : []),
      ...(port.description ? [`Desc: ${port.description}`] : []),
    ].join('\n');
  };

  const renderSlotRow = (slot: ApiChassisSlot) => {
    const isUplink = slot.type === 'uplink';
    const isActive = slot.present;
    const isSmxa = isUplink && slot.cardType.toUpperCase().startsWith('SMXA');
    const rowBg = !isActive ? '#0d1117' : isUplink ? '#0c1a2e' : '#0a1a0a';
    const rowBorder = !isActive ? '#1e293b' : isUplink ? '#1d4ed8' : '#15803d';
    const labelColor = isUplink ? '#60a5fa' : '#4ade80';
    const servicePortCount = Math.max(slot.portCount || 0, 16);

    return (
      <div key={slot.index} className="flex items-center gap-0 rounded overflow-hidden select-none"
        style={{ background: rowBg, border: `1px solid ${rowBorder}`, minHeight: 48 }}>
        <div style={{ width: 4, alignSelf: 'stretch', background: isActive ? rowBorder : '#1e293b' }} />
        <div className="flex items-center justify-start px-3" style={{ minWidth: 88 }}>
          {isActive ? (
            <span className="text-xs font-bold font-mono tracking-wider" style={{ color: labelColor }}>{slot.cardType}</span>
          ) : (
            <span className="text-xs text-gray-600 font-mono">--</span>
          )}
        </div>
        <div className="flex-1 py-2 pr-3 overflow-x-auto">
          {!isActive ? (
            <div className="flex items-center h-full text-xs text-gray-700 tracking-[0.3em] justify-center">EMPTY</div>
          ) : isSmxa ? (
            <div className="flex items-center gap-1.5 flex-nowrap min-w-max">
              {(slot.ports.length > 0 ? slot.ports : (slot.uplinkIfaces ?? []).map((iface, index) => ({ port: index, iface, onuCount: 0, onlineCount: 0, hasOnus: false })) ).map((port, index) => {
                const iface = port.iface ?? (index === 0 ? `gei_1/${slot.index}` : `xgei_1/${slot.index}/${index}`);
                const isXGE = iface.startsWith('xgei');
                const shortLabel = isXGE
                  ? iface.replace(/^xgei_1\/\d+\//, 'X/')
                  : iface.replace(/^gei_1\//, 'G/');
                const visual = getUplinkPortVisual(port);

                return (
                  <button key={iface}
                    onClick={() => setSelectedUplinkPort(iface)}
                    title={uplinkTooltip(slot, port, index + 1)}
                    className="flex flex-col items-center px-2 py-1 rounded border transition-all hover:brightness-125 hover:scale-105 cursor-pointer"
                    style={{ background: visual.bg, borderColor: visual.border, minWidth: 48 }}>
                    <div className="w-1.5 h-1.5 rounded-full mb-0.5" style={{ background: visual.dot }} />
                    <span className="text-[8px] font-mono leading-none whitespace-nowrap" style={{ color: visual.text }}>{shortLabel}</span>
                    <span className="text-[7px] font-mono" style={{ color: visual.text }}>{visual.state}</span>
                  </button>
                );
              })}
              <span className="ml-1 text-[9px] text-blue-400 font-mono">{slot.cardType}</span>
            </div>
          ) : isUplink ? (
            <div className="flex items-center gap-1.5 flex-nowrap min-w-max">
              {slot.ports.map((port, index) => {
                const iface = port.iface ?? `xgei_1/${slot.index}/${index + 1}`;
                const visual = getUplinkPortVisual(port);

                return (
                  <button key={iface}
                    onClick={() => setSelectedUplinkPort(iface)}
                    title={uplinkTooltip(slot, port, index + 1)}
                    className="w-10 h-10 rounded-sm border flex flex-col items-center justify-center transition-all hover:brightness-125 hover:scale-105"
                    style={{ background: visual.bg, borderColor: visual.border }}>
                    <div className="w-1.5 h-1.5 rounded-full mb-1" style={{ background: visual.dot }} />
                    <span className="text-[7px] font-mono" style={{ color: visual.text }}>{visual.state}</span>
                  </button>
                );
              })}
              <span className="ml-2 text-[9px] text-blue-400 font-mono">{slot.cardType}</span>
            </div>
          ) : (
            <div className="flex items-center gap-1 flex-nowrap min-w-max">
              {Array.from({ length: servicePortCount }, (_, i) => {
                const c = portColor(slot.index, i + 1);
                const s = portStats[`${slot.index}/${i + 1}`];

                return (
                  <button key={i + 1}
                    className="w-4 h-4 rounded-[3px] border flex items-center justify-center cursor-pointer transition-all hover:brightness-150 hover:scale-110 hover:z-10 relative"
                    style={{ background: c.bg, borderColor: c.border }}
                    title={portTooltip(slot.index, i + 1)}
                    onClick={() => {
                      const pk = `${slot.index}/${i + 1}`;
                      setSelectedPON({ slot: slot.index, port: i + 1 });
                      refreshPONStat(pk);
                      setTimeout(() => fetchPONStatRef.current(pk), 50);
                    }}>
                    <div className="w-1 h-1 rounded-full" style={{ background: c.dot }} />
                    {s && s.unregistered > 0 && (
                      <div className="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-yellow-400 border border-yellow-600" title="Unregistered ONU" />
                    )}
                  </button>
                );
              })}
            </div>
          )}
        </div>
        <div className="px-3 text-right" style={{ minWidth: 36 }}>
          <span className="text-xs text-gray-500 font-mono">{slot.index}</span>
        </div>
        </div>
      );
    };

    return (
      <div className="space-y-4">
        {selectedUplinkPort && (
          <UplinkPortModal oltId={olt.id} port={selectedUplinkPort} onClose={() => setSelectedUplinkPort(null)} />
        )}
        {selectedPON && (
          <PONPortModal
            oltId={olt.id}
            slot={selectedPON.slot}
            port={selectedPON.port}
            stat={ponStatCache[`${selectedPON.slot}/${selectedPON.port}`]}
            loadingStat={loadingPON === `${selectedPON.slot}/${selectedPON.port}`}
            acting={ponActing === `${selectedPON.slot}/${selectedPON.port}`}
            portStats={portStats[`${selectedPON.slot}/${selectedPON.port}`]}
            onAction={(action) => handlePONAction(`${selectedPON.slot}/${selectedPON.port}`, action)}
            onRefresh={() => {
              const pk = `${selectedPON.slot}/${selectedPON.port}`;
              refreshPONStat(pk);
              setTimeout(() => fetchPONStatRef.current(pk), 50);
            }}
            onClose={() => setSelectedPON(null)}
          />
        )}

        {/* -- Main rack panel -- */}
        <div className="rounded-xl overflow-hidden shadow-xl bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800">

          {/* Header */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between px-4 sm:px-5 py-3 border-b border-slate-200 dark:border-slate-800">
            <div>
              <div className="flex items-center gap-2">
                <Server className="h-4 w-4 text-green-400" />
                <span className="font-semibold text-sm text-slate-900 dark:text-white">ZTE C320 Rack Diagram</span>
              </div>
              <div className="text-[11px] text-slate-500 mt-1">Updated: {onuLastRefresh ? onuLastRefresh.toLocaleTimeString('id-ID') : olt.lastPollAt ? new Date(olt.lastPollAt).toLocaleTimeString('id-ID') : '--'}</div>
            </div>
            <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3">
              <span className="text-xs text-slate-500 font-mono">{olt.ipAddress}</span>
              <button
                onClick={fetchChassis}
                className="inline-flex items-center justify-center gap-1.5 px-2.5 py-1 text-xs rounded border border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                disabled={loadingChassis}
              >
                <RefreshCw className={`h-3 w-3 ${loadingChassis ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>

          {/* Rack diagram body */}
          <div className="p-3 sm:p-4 flex flex-col lg:flex-row gap-3 overflow-x-auto">
            {/* FAN column */}
            <div className="flex flex-col items-center justify-between py-3 px-2 rounded select-none"
              style={{ minWidth: 56, background: '#161b22', border: '1px solid #21262d' }}>
              <div className="flex items-center gap-1">
                <Zap className="h-3 w-3 text-cyan-400" />
                <span className="text-[9px] text-gray-400 font-mono font-bold">FAN</span>
              </div>
              <div className="flex flex-col items-center gap-4 my-2">
                {[1, 2].map(n => (
                  <div key={n} className="flex flex-col items-center gap-1">
                    <div className="relative w-9 h-9 flex items-center justify-center">
                      <div className="absolute inset-0 rounded-full" style={{ border: '2px solid #16a34a' }} />
                      <div className="animate-spin" style={{ animationDuration: '2.5s' }}>
                        <svg width="20" height="20" viewBox="0 0 20 20">
                          <path d="M10 4 Q14 8 10 10 Q14 12 10 16 Q6 12 10 10 Q6 8 10 4Z" fill="#22c55e" opacity="0.8" />
                          <path d="M4 10 Q8 6 10 10 Q8 14 4 10 Q8 14 10 10 Q8 6 4 10 Q8 14 16 10 Q12 14 10 10 Q12 6 16 10Z" fill="#16a34a" opacity="0.5" />
                        </svg>
                      </div>
                    </div>
                    <span className="text-[8px] text-green-400 font-mono">{n}</span>
                  </div>
                ))}
              </div>
              <div className="text-center">
                <div className="text-[10px] font-bold text-green-400 font-mono">{olt.isOnline ? '2/2' : '0/2'}</div>
                <div className="text-[8px] text-gray-500">Active</div>
              </div>
            </div>

            {/* Slot rows */}
            <div className="flex-1 flex flex-col gap-1.5">
              {diagramSlots.map(slot => renderSlotRow(slot))}
            </div>
          </div>

          {/* Legend */}
          <div className="flex items-center gap-5 px-4 py-2.5 flex-wrap bg-slate-50 dark:bg-slate-950 border-t border-slate-200 dark:border-slate-800">
            {[
              { bg: '#14532d', border: '#16a34a', dot: '#4ade80', label: 'Online' },
              { bg: '#111827', border: '#475569', dot: '#64748b', label: 'Disabled' },
              { bg: '#451a03', border: '#f59e0b', dot: '#fbbf24', label: 'Admin UP / Port DOWN' },
              { bg: '#7c2d12', border: '#f97316', dot: '#fb923c', label: 'Dying Gasp' },
              { bg: '#450a0a', border: '#dc2626', dot: '#f87171', label: 'LOS' },
              { bg: '#713f12', border: '#ca8a04', dot: '#facc15', label: 'Unconfigured' },
            ].map(item => (
              <div key={item.label} className="flex items-center gap-1.5">
                <div className="w-4 h-4 rounded-sm border flex items-center justify-center"
                  style={{ background: item.bg, borderColor: item.border }}>
                  <div className="w-1.5 h-1.5 rounded-full" style={{ background: item.dot }} />
                </div>
                <span className="text-[9px] text-gray-400">{item.label}</span>
              </div>
            ))}
          </div>
        </div>

        {/* -- Per-port detail table -- */}
        {Object.keys(portStats).length > 0 && (
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3 text-gray-800 dark:text-gray-200">Detail Per Port PON</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
              {Object.entries(portStats)
                .sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true }))
                .map(([portKey, s]) => {
                  const pct = s.total > 0 ? (s.online / s.total) * 100 : 0;
                  const avgRx = s.rxPowers.length > 0 ? (s.rxPowers.reduce((a, b) => a + b, 0) / s.rxPowers.length).toFixed(1) : null;
                  const isExpanded = expandedPON === portKey;
                  const stat = ponStatCache[portKey];
                  const isLoading = loadingPON === portKey;

                  const handleToggle = () => {
                    if (isExpanded) {
                      setExpandedPON(null);
                    } else {
                      setExpandedPON(portKey);
                      fetchPONStat(portKey);
                    }
                  };

                  const fmtBps = (bps?: number) => {
                    if (bps == null) return '--';
                    if (bps >= 1_000_000) return (bps / 1_000_000).toFixed(1) + ' Mbps';
                    if (bps >= 1_000) return (bps / 1_000).toFixed(1) + ' Kbps';
                    return bps + ' Bps';
                  };

                  return (
                    <div key={portKey} className="border border-gray-100 dark:border-gray-800 rounded-lg overflow-hidden">
                      {/* Card header -- always visible */}
                      <button
                        onClick={handleToggle}
                        className="w-full text-left p-2.5 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-xs font-semibold text-gray-700 dark:text-gray-300">
                            PON 0/{portKey}
                          </span>
                          <div className="flex items-center gap-2">
                            <span className={`text-[10px] font-bold ${pct === 100 ? 'text-green-600' : pct === 0 ? 'text-red-600' : 'text-orange-500'}`}>
                              {s.online}/{s.total} ONU
                            </span>
                            <span className="text-[9px] text-gray-400">{isExpanded ? 'ODP' : '[vTRI]'}</span>
                          </div>
                        </div>
                        <div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-1.5 mb-1">
                          <div
                            className={`h-1.5 rounded-full ${pct === 100 ? 'bg-green-500' : pct === 0 ? 'bg-red-500' : 'bg-orange-400'}`}
                            style={{ width: `${Math.max(pct, s.total > 0 ? 5 : 0)}%` }}
                          />
                        </div>
                        <div className="flex justify-between text-[9px] text-gray-400 gap-2">
                          <span>{s.total} ONU{s.unregistered > 0 ? ` . ${s.unregistered} unreg` : ''}</span>
                          {avgRx && <span>Avg RX: {avgRx} dBm</span>}
                        </div>
                      </button>

                      {/* Expanded stats panel */}
                      {isExpanded && (
                        <div className="border-t border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-2.5">
                          {isLoading ? (
                            <div className="text-[10px] text-gray-400 text-center py-2">Memuat data...</div>
                          ) : stat === null ? (
                            <div className="text-[10px] text-red-400 text-center py-2">Gagal mengambil data Telnet</div>
                          ) : stat ? (
                            <div className="space-y-2">
                              {/* Optical module stats */}
                              <div className="grid grid-cols-2 gap-1.5">
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">Temperature</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {stat.temperature != null ? `${stat.temperature.toFixed(1)}  degC` : '--'}
                                  </div>
                                </div>
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">TX Power</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {stat.txPower != null ? `${stat.txPower.toFixed(2)} dBm` : '--'}
                                  </div>
                                </div>
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">Voltage</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {stat.voltage != null ? `${stat.voltage.toFixed(2)} V` : '--'}
                                  </div>
                                </div>
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">Bias Current</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {stat.biasCurrent != null ? `${stat.biasCurrent.toFixed(1)} mA` : '--'}
                                  </div>
                                </div>
                              </div>
                              {/* Traffic stats */}
                              <div className="grid grid-cols-2 gap-1.5">
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">UL  Upstream</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {fmtBps(stat.inputBps)}
                                  </div>
                                  {stat.inputPct != null && (
                                    <div className="text-[9px] text-gray-400">{stat.inputPct.toFixed(1)}% usage</div>
                                  )}
                                </div>
                                <div className="bg-white dark:bg-gray-900 rounded p-1.5">
                                  <div className="text-[9px] text-gray-400 mb-0.5">DL  Downstream</div>
                                  <div className="text-xs font-semibold text-gray-800 dark:text-gray-200">
                                    {fmtBps(stat.outputBps)}
                                  </div>
                                  {stat.outputPct != null && (
                                    <div className="text-[9px] text-gray-400">{stat.outputPct.toFixed(1)}% usage</div>
                                  )}
                                </div>
                              </div>
                              {stat.description && (
                                <div className="text-[9px] text-gray-400 truncate">{stat.description}</div>
                              )}
                              {/* PON port actions */}
                              {editingPONDesc?.portKey === portKey ? (
                                <div className="flex items-center gap-1 pt-1 border-t border-gray-100 dark:border-gray-800">
                                  <input
                                    autoFocus
                                    value={editingPONDesc.value}
                                    onChange={e => setEditingPONDesc({ portKey, value: e.target.value })}
                                    className="flex-1 text-[10px] border border-gray-300 dark:border-gray-600 rounded px-1.5 py-0.5 bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-200"
                                    placeholder="Description..."
                                  />
                                  <button
                                    disabled={ponActing === portKey}
                                    onClick={() => {
                                      handlePONAction(portKey, 'setDescription', editingPONDesc.value);
                                      setEditingPONDesc(null);
                                    }}
                                    className="text-[9px] px-1.5 py-0.5 rounded bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
                                  >
                                    OK
                                  </button>
                                  <button
                                    onClick={() => setEditingPONDesc(null)}
                                    className="text-[9px] px-1.5 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300"
                                  >
                                    X
                                  </button>
                                </div>
                              ) : (
                                <div className="flex items-center gap-1 pt-1 border-t border-gray-100 dark:border-gray-800">
                                  <button
                                    disabled={ponActing === portKey}
                                    onClick={() => handlePONAction(portKey,
                                      stat.adminStatus?.toLowerCase().includes('deactivate') ? 'enable' : 'disable'
                                    )}
                                    className={`flex-1 text-[9px] px-1.5 py-0.5 rounded border font-medium disabled:opacity-50 ${
                                      stat.adminStatus?.toLowerCase().includes('deactivate')
                                        ? 'border-green-400 text-green-600 hover:bg-green-50 dark:hover:bg-green-950'
                                        : 'border-red-400 text-red-500 hover:bg-red-50 dark:hover:bg-red-950'
                                    }`}
                                  >
                                    {ponActing === portKey ? '...' : stat.adminStatus?.toLowerCase().includes('deactivate') ? 'Enable' : 'Disable'}
                                  </button>
                                  <button
                                    disabled={ponActing === portKey}
                                    onClick={() => setEditingPONDesc({ portKey, value: stat.description ?? '' })}
                                    className="flex-1 text-[9px] px-1.5 py-0.5 rounded border border-blue-400 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-950 font-medium disabled:opacity-50"
                                  >
                                    Edit Desc
                                  </button>
                                </div>
                              )}
                            </div>
                          ) : null}
                        </div>
                      )}
                    </div>
                  );
                })}
            </div>
          </div>
        )}
      </div>
    );
}

type ZteServiceTemplate = 'basic' | 'zte_full' | 'huawei_full' | 'fiberhome_veip';

interface RegisterMetadata {
  onuTypes: string[];
  tcontProfiles: string[];
  trafficProfiles: string[];
  suggestedOnuId: number | null;
  detectedOnuType: string | null;
}

interface RegisterModalProps {
  oltId: string;
  onu: ONU;
  vendor: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

function ONURegisterModal({ oltId, onu, vendor, onClose, onSuccess }: RegisterModalProps) {
  const v = (vendor ?? 'zte').toLowerCase();
  const isHuawei     = v === 'huawei';
  const isFiberHome  = v === 'fiberhome';
  const isZTE        = !isHuawei && !isFiberHome;

  const [onuType,       setOnuType]       = useState(onu.onuType ?? '');
  const [vlan,          setVlan]          = useState(100);
  const [serviceTemplate, setServiceTemplate] = useState<ZteServiceTemplate>('basic');
  const [tcontProfile,  setTcontProfile]  = useState('1G');
  const [trafficProfile, setTrafficProfile] = useState('');
  const [description,   setDescription]   = useState('');
  const [onuId,         setOnuId]         = useState(onu.onuId ?? 1);
  const [primaryVlan, setPrimaryVlan] = useState(30);
  const [secondaryVlan, setSecondaryVlan] = useState(151);
  const [mgmtVlan, setMgmtVlan] = useState(1010);
  const [internetVlan, setInternetVlan] = useState(30);
  const [voipVlan, setVoipVlan] = useState(151);
  const [vlanProfile, setVlanProfile] = useState('genieacs');
  const [pppoeUsername, setPppoeUsername] = useState('');
  const [pppoePassword, setPppoePassword] = useState('');
  const [enableDualSsid, setEnableDualSsid] = useState(true);
  const [ssid1Name, setSsid1Name] = useState('');
  const [ssid1Password, setSsid1Password] = useState('12345678');
  const [ssid1Auth, setSsid1Auth] = useState('wpa2');
  const [ssid2Name, setSsid2Name] = useState('');
  const [ssid2Password, setSsid2Password] = useState('');
  const [ssid2Auth, setSsid2Auth] = useState('open');
  const [enableTr069, setEnableTr069] = useState(true);
  const [tr069Vlan, setTr069Vlan] = useState(100);
  const [acsUrl, setAcsUrl] = useState('http://192.168.54.254:7547');
  const [acsUsername, setAcsUsername] = useState('acs');
  const [acsPassword, setAcsPassword] = useState('acs');
  const [enableFirewall, setEnableFirewall] = useState(true);
  const [firewallLevel, setFirewallLevel] = useState('low');
  const [enableSecurityMgmt, setEnableSecurityMgmt] = useState(true);
  // Huawei-specific
  const [lineProfileId, setLineProfileId] = useState(1);
  const [srvProfileId,  setSrvProfileId]  = useState(1);
  // FiberHome-specific
  const [profileName,   setProfileName]   = useState('default');

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null);
  const [metadataLoading, setMetadataLoading] = useState(true);
  const [metadata, setMetadata] = useState<RegisterMetadata>({
    onuTypes: [],
    tcontProfiles: [],
    trafficProfiles: [],
    suggestedOnuId: null,
    detectedOnuType: onu.onuType ?? null,
  });

  const ponPort = onu.port + 1;
  const effectiveOnuType = onuType || metadata.detectedOnuType || metadata.onuTypes[0] || (isZTE ? 'All' : isFiberHome ? 'default' : '');
  const effectiveTrafficProfile = trafficProfile || metadata.trafficProfiles[0] || '';

  useEffect(() => {
    let active = true;
    setMetadataLoading(true);

    fetch(`/api/olt/${oltId}/onus/register?frame=${onu.frame}&slot=${onu.slot}&port=${onu.port}&onuId=${onu.onuId}&serialNumber=${encodeURIComponent(onu.serialNumber ?? '')}`)
      .then(async res => {
        const json = await res.json();
        if (!res.ok || !json.success) throw new Error(json.error ?? 'Failed to load register metadata');
        if (!active) return;

        const nextMetadata: RegisterMetadata = {
          onuTypes: json.metadata?.onuTypes ?? [],
          tcontProfiles: json.metadata?.tcontProfiles ?? [],
          trafficProfiles: json.metadata?.trafficProfiles ?? [],
          suggestedOnuId: json.metadata?.suggestedOnuId ?? null,
          detectedOnuType: json.metadata?.detectedOnuType ?? null,
        };

        setMetadata(nextMetadata);
        if (nextMetadata.suggestedOnuId) setOnuId(nextMetadata.suggestedOnuId);
        if (!onuType) setOnuType(nextMetadata.detectedOnuType ?? nextMetadata.onuTypes[0] ?? '');
        if (isZTE && nextMetadata.tcontProfiles.length > 0) setTcontProfile(nextMetadata.tcontProfiles[0]);
        if (isZTE && nextMetadata.trafficProfiles.length > 0) setTrafficProfile(nextMetadata.trafficProfiles[0]);
      })
      .catch((error: any) => {
        if (!active) return;
        setResult({ ok: false, msg: error.message });
      })
      .finally(() => {
        if (active) setMetadataLoading(false);
      });

    return () => {
      active = false;
    };
  }, [isZTE, oltId, onu.frame, onu.onuId, onu.port, onu.serialNumber, onu.slot]);

  const handleSubmit = async () => {
    setLoading(true);
    setResult(null);
    try {
      const res = await fetch(`/api/olt/${oltId}/onus/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          frame: onu.frame,
          slot:  onu.slot,
          port:  onu.port,
          onuId,
          serialNumber: onu.serialNumber,
          onuType: effectiveOnuType,
          vlan,
          description: description || undefined,
          serviceTemplate,
          // ZTE
          tcontProfile,
          trafficProfile: effectiveTrafficProfile || undefined,
          primaryVlan,
          secondaryVlan,
          mgmtVlan,
          internetVlan,
          voipVlan,
          vlanProfile,
          pppoeUsername,
          pppoePassword,
          enableDualSsid,
          ssid1Name,
          ssid1Password,
          ssid1Auth,
          ssid2Name,
          ssid2Password,
          ssid2Auth,
          enableTr069,
          tr069Vlan,
          acsUrl,
          acsUsername,
          acsPassword,
          enableFirewall,
          firewallLevel,
          enableSecurityMgmt,
          // Huawei
          lineProfileId,
          srvProfileId,
          // FiberHome
          profileName,
        }),
      });
      const data = await res.json();
      if (res.ok && data.success) {
        setResult({ ok: true, msg: data.message });
        setTimeout(() => { onSuccess(); onClose(); }, 1500);
      } else {
        setResult({ ok: false, msg: data.error ?? 'Registration failed' });
      }
    } catch (e: any) {
      setResult({ ok: false, msg: e.message });
    } finally {
      setLoading(false);
    }
  };

  // Build command preview per vendor
  const zteTemplatePreview = serviceTemplate === 'zte_full' ? [
    `interface gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    ...(description ? [`  name ${description}`, `  description ${description}`] : []),
    `  tcont 1 name VLAN${String(primaryVlan).padStart(4, '0')} profile ${tcontProfile}`,
    `  tcont 2 name VLAN${secondaryVlan} profile ${tcontProfile}`,
    '  gemport 1 tcont 1',
    ...(effectiveTrafficProfile ? [`  gemport 1 traffic-limit downstream ${effectiveTrafficProfile}`] : []),
    '  gemport 2 tcont 2',
    ...(effectiveTrafficProfile ? [`  gemport 2 traffic-limit downstream ${effectiveTrafficProfile}`] : []),
    `  service-port 1 vport 1 user-vlan ${primaryVlan} vlan ${primaryVlan}`,
    `  service-port 2 vport 2 user-vlan ${secondaryVlan} vlan ${secondaryVlan}`,
    'exit',
    `pon-onu-mng gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    `  service VLAN${String(primaryVlan).padStart(4, '0')} gemport 1 iphost 1 vlan ${primaryVlan}`,
    `  service VLAN${secondaryVlan} gemport 2 vlan ${secondaryVlan}`,
    '  vlan port veip_1 mode hybrid',
    '  vlan port veip_1 vlan 1',
    ...(pppoeUsername && pppoePassword ? [`  pppoe 1 nat enable user ${pppoeUsername} password ${pppoePassword}`] : []),
    `  vlan port eth_0/1 mode tag vlan ${primaryVlan}`,
    `  vlan port eth_0/2 mode tag vlan ${primaryVlan}`,
    `  vlan port eth_0/3 mode tag vlan ${primaryVlan}`,
    `  vlan port eth_0/4 mode tag vlan ${primaryVlan}`,
    `  vlan port wifi_0/1 mode tag vlan ${primaryVlan}`,
    ...(enableDualSsid ? [`  vlan port wifi_0/2 mode tag vlan ${secondaryVlan}`] : []),
    ...(ssid1Name ? [`  wifi ssid 1 name ${ssid1Name}`, `  wifi ssid 1 auth ${ssid1Auth === 'wpa' ? 'wpa-psk' : 'wpa2-psk'}`, ...(ssid1Password ? [`  wifi ssid 1 wpakey ${ssid1Password}`] : []), `  wifi ssid 1 bindvlan ${primaryVlan}`, '  wifi ssid 1 enable'] : []),
    ...(enableDualSsid && ssid2Name ? [`  wifi ssid 2 name ${ssid2Name}`, ...(ssid2Auth === 'open' ? ['  wifi ssid 2 auth open'] : [`  wifi ssid 2 auth ${ssid2Auth === 'wpa' ? 'wpa-psk' : 'wpa2-psk'}`]), ...(ssid2Auth !== 'open' && ssid2Password ? [`  wifi ssid 2 wpakey ${ssid2Password}`] : []), `  wifi ssid 2 bindvlan ${secondaryVlan}`, '  wifi ssid 2 enable'] : []),
    ...(enableFirewall ? [`  firewall enable level ${firewallLevel} anti-hack disable`] : []),
    ...(enableTr069 ? ['  tr069-mgmt 1 state unlock', `  tr069-mgmt 1 acs ${acsUrl} validate basic username ${acsUsername} password ${acsPassword}`] : []),
    ...(enableSecurityMgmt ? ['  security-mgmt 1 state enable mode forward'] : []),
    '  wan 1 service internet host 1',
    'exit',
  ] : serviceTemplate === 'huawei_full' ? [
    `interface gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    ...(description ? [`  name ${description}`, `  description ${description}`] : []),
    `  tcont 1 profile ${tcontProfile}`,
    '  gemport 1 tcont 1',
    ...(effectiveTrafficProfile ? [`  gemport 1 traffic-limit downstream ${effectiveTrafficProfile}`] : []),
    `  service-port 1 vport 1 user-vlan ${mgmtVlan} vlan ${mgmtVlan}`,
    `  service-port 2 vport 1 user-vlan ${internetVlan} vlan ${internetVlan}`,
    `  service-port 3 vport 1 user-vlan ${voipVlan} vlan ${voipVlan}`,
    'exit',
    `pon-onu-mng gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    '  service ServiceONU1 gemport 1',
    `  wan-ip 1 mode dhcp vlan-profile ${vlanProfile} host 1`,
    'exit',
  ] : serviceTemplate === 'fiberhome_veip' ? [
    `interface gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    ...(description ? [`  name ${description}`, `  description ${description}`] : []),
    `  tcont 1 profile ${tcontProfile}`,
    `  tcont 2 profile ${tcontProfile}`,
    `  tcont 3 profile ${tcontProfile}`,
    '  gemport 1 tcont 1',
    '  gemport 2 tcont 2',
    '  gemport 3 tcont 3',
    `  service-port 1 vport 1 user-vlan ${tr069Vlan} vlan ${tr069Vlan}`,
    `  service-port 2 vport 2 user-vlan ${internetVlan} vlan ${internetVlan}`,
    `  service-port 3 vport 3 user-vlan ${voipVlan} vlan ${voipVlan}`,
    'exit',
    `pon-onu-mng gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    `  service 1 gemport 1 vlan ${tr069Vlan}`,
    `  service 2 gemport 2 vlan ${internetVlan}`,
    `  service 3 gemport 3 vlan ${voipVlan}`,
    '  vlan port veip_1 mode hybrid',
    '  tr069-mgmt 1 state unlock',
    `  tr069-mgmt 1 acs ${acsUrl} validate basic username ${acsUsername} password ${acsPassword}`,
    `  vlan port wifi_0/1 mode tag vlan ${internetVlan}`,
    `  vlan port eth_0/1 mode tag vlan ${internetVlan}`,
    `  vlan port eth_0/2 mode tag vlan ${internetVlan}`,
    `  vlan port eth_0/3 mode tag vlan ${internetVlan}`,
    `  vlan port eth_0/4 mode tag vlan ${internetVlan}`,
    'exit',
  ] : [
    `interface gpon-onu_${onu.frame}/${onu.slot}/${ponPort}:${onuId}`,
    ...(description ? [`  name ${description}`, `  description ${description}`] : []),
    `  tcont 1 profile ${tcontProfile}`,
    '  gemport 1 tcont 1',
    `  service-port 1 vport 1 user-vlan ${vlan} vlan ${vlan}`,
    'exit',
  ];

  const cmdPreview = isHuawei ? [
    'enable',
    'config',
    `interface gpon ${onu.frame}/${onu.slot}`,
    `  ont add ${ponPort} ${onuId} sn-auth ${onu.serialNumber ?? '???'} omci ont-lineprofile-id ${lineProfileId} ont-srvprofile-id ${srvProfileId}${description ? ` desc ${description}` : ''}`,
    'quit',
    `service-port 1 vlan ${vlan} gpon ${onu.frame}/${onu.slot}/${ponPort} ont ${onuId} gemport 0 multi-service user-vlan ${vlan} tag-transform translate`,
    'quit',
  ] : isFiberHome ? [
    'enable',
    'config',
    `interface gpon-olt_${onu.frame}/${onu.slot}/${ponPort}`,
    `  onu add ${onuId} type ${effectiveOnuType || 'TYPE'} sn ${onu.serialNumber ?? '???'}`,
    `  onu ${onuId} profile ${profileName}`,
    `  onu ${onuId} vlan ${vlan} mode translate`,
    ...(description ? [`  onu ${onuId} description ${description}`] : []),
    '  commit',
    'exit',
  ] : [
    'configure terminal',
    `interface gpon-olt_${onu.frame}/${onu.slot}/${ponPort}`,
    `  onu ${onuId} type ${effectiveOnuType || 'All'} sn ${onu.serialNumber ?? '???'}`,
    'exit',
    ...zteTemplatePreview,
    'end',
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-gray-800 bg-green-50 dark:bg-green-950 flex-shrink-0">
          <div>
            <h2 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
              <Plus className="h-4 w-4 text-green-600" /> Register ONU
              <span className="text-xs font-normal text-gray-500 uppercase ml-1">{v}</span>
            </h2>
            <p className="text-xs text-gray-500 mt-0.5 font-mono">
              PON {onu.frame}/{onu.slot}/{ponPort} . ONU ID {onuId}
            </p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body -- scrollable */}
        <div className="p-5 space-y-4 overflow-y-auto">
          {/* Serial number (read-only) */}
          <div>
            <Label className="text-xs text-gray-500">Serial Number (OLT detected)</Label>
            <div className="mt-1 px-3 py-2 bg-gray-100 dark:bg-gray-800 rounded-md font-mono text-sm text-gray-700 dark:text-gray-300">
              {onu.serialNumber ?? <span className="text-yellow-600">Unknown -- no serial via SNMP (enter manually)</span>}
            </div>
          </div>

          {/* ONU ID */}
          <div>
            <Label className="text-xs text-gray-500">ONU ID (1-128)</Label>
            <Input type="number" min={1} max={128} value={onuId}
              onChange={e => setOnuId(parseInt(e.target.value) || 1)}
              className="mt-1 font-mono caret-gray-900 dark:caret-white" />
            {metadata.suggestedOnuId && (
              <p className="text-xs text-gray-400 mt-1">Suggested from OLT: {metadata.suggestedOnuId}</p>
            )}
          </div>

          {/* -- ZTE-only fields -- */}
          {isZTE && (<>
            <div>
              <Label className="text-xs text-gray-500">Config Template</Label>
              <Select value={serviceTemplate} onValueChange={(value) => setServiceTemplate(value as ZteServiceTemplate)}>
                <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="basic">Basic register</SelectItem>
                  <SelectItem value="zte_full">ZTE Full</SelectItem>
                  <SelectItem value="huawei_full">Huawei Full</SelectItem>
                  <SelectItem value="fiberhome_veip">Fiberhome VEIP</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-gray-400 mt-1">Flow diambil dari onu_register_wizard.py pada OLT C320 V2.1.1.</p>
            </div>
            <div>
              <Label className="text-xs text-gray-500">ONU Type</Label>
              {metadata.onuTypes.length > 0 ? (
                <Select value={effectiveOnuType} onValueChange={setOnuType}>
                  <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {metadata.onuTypes.map(type => (
                      <SelectItem key={type} value={type}>{type}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input value={onuType} onChange={e => setOnuType(e.target.value)} placeholder="ONU type from OLT" className="mt-1 font-mono" />
              )}
              <p className="text-xs text-gray-400 mt-1">Loaded live from OLT running-config.</p>
            </div>
            <div>
              <Label className="text-xs text-gray-500">TCONT Profile (bandwidth)</Label>
              {metadata.tcontProfiles.length > 0 ? (
                <Select value={tcontProfile} onValueChange={setTcontProfile}>
                  <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {metadata.tcontProfiles.map(profile => (
                      <SelectItem key={profile} value={profile}>{profile}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input value={tcontProfile} onChange={e => setTcontProfile(e.target.value)} placeholder="TCONT profile" className="mt-1 font-mono" />
              )}
            </div>

            {serviceTemplate === 'basic' && (
              <div>
                <Label className="text-xs text-gray-500">Service VLAN</Label>
                <Input type="number" min={1} max={4094} value={vlan}
                  onChange={e => setVlan(parseInt(e.target.value) || 100)}
                  className="mt-1 font-mono caret-gray-900 dark:caret-white" />
              </div>
            )}

            {(serviceTemplate === 'zte_full' || serviceTemplate === 'huawei_full') && (
              <div>
                <Label className="text-xs text-gray-500">Traffic Profile</Label>
                {metadata.trafficProfiles.length > 0 ? (
                  <Select value={effectiveTrafficProfile} onValueChange={setTrafficProfile}>
                    <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {metadata.trafficProfiles.map(profile => (
                        <SelectItem key={profile} value={profile}>{profile}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input value={trafficProfile} onChange={e => setTrafficProfile(e.target.value)} placeholder="Traffic profile" className="mt-1 font-mono" />
                )}
              </div>
            )}

            {serviceTemplate === 'zte_full' && (
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-4">
                <div className="text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wide">ZTE Full Template</div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-gray-500">Primary VLAN</Label>
                    <Input type="number" min={1} max={4094} value={primaryVlan} onChange={e => setPrimaryVlan(parseInt(e.target.value) || 30)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">Secondary VLAN</Label>
                    <Input type="number" min={1} max={4094} value={secondaryVlan} onChange={e => setSecondaryVlan(parseInt(e.target.value) || 151)} className="mt-1 font-mono" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-gray-500">PPPoE Username</Label>
                    <Input value={pppoeUsername} onChange={e => setPppoeUsername(e.target.value)} className="mt-1 font-mono" placeholder="optional" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">PPPoE Password</Label>
                    <Input value={pppoePassword} onChange={e => setPppoePassword(e.target.value)} className="mt-1 font-mono" placeholder="optional" />
                  </div>
                </div>
                <div className="flex items-center justify-between rounded-md bg-gray-50 dark:bg-gray-950 px-3 py-2">
                  <div>
                    <div className="text-sm text-gray-900 dark:text-gray-100">Dual SSID</div>
                    <div className="text-xs text-gray-500">Aktifkan SSID kedua di VLAN sekunder.</div>
                  </div>
                  <Switch checked={enableDualSsid} onCheckedChange={setEnableDualSsid} />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-gray-500">SSID 1 Name</Label>
                    <Input value={ssid1Name} onChange={e => setSsid1Name(e.target.value)} className="mt-1 font-mono" placeholder="optional" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">SSID 1 Password</Label>
                    <Input value={ssid1Password} onChange={e => setSsid1Password(e.target.value)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">SSID 1 Auth</Label>
                    <Select value={ssid1Auth} onValueChange={setSsid1Auth}>
                      <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="wpa2">WPA2-PSK</SelectItem>
                        <SelectItem value="wpa">WPA-PSK</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                {enableDualSsid && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <Label className="text-xs text-gray-500">SSID 2 Name</Label>
                      <Input value={ssid2Name} onChange={e => setSsid2Name(e.target.value)} className="mt-1 font-mono" placeholder="optional" />
                    </div>
                    <div>
                      <Label className="text-xs text-gray-500">SSID 2 Password</Label>
                      <Input value={ssid2Password} onChange={e => setSsid2Password(e.target.value)} className="mt-1 font-mono" placeholder="only for WPA/WPA2" />
                    </div>
                    <div>
                      <Label className="text-xs text-gray-500">SSID 2 Auth</Label>
                      <Select value={ssid2Auth} onValueChange={setSsid2Auth}>
                        <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="open">Open</SelectItem>
                          <SelectItem value="wpa2">WPA2-PSK</SelectItem>
                          <SelectItem value="wpa">WPA-PSK</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                )}
                <div className="flex items-center justify-between rounded-md bg-gray-50 dark:bg-gray-950 px-3 py-2">
                  <div>
                    <div className="text-sm text-gray-900 dark:text-gray-100">TR-069</div>
                    <div className="text-xs text-gray-500">Tambahkan ACS provisioning seperti wizard reference.</div>
                  </div>
                  <Switch checked={enableTr069} onCheckedChange={setEnableTr069} />
                </div>
                {enableTr069 && (
                  <div className="grid grid-cols-2 gap-3">
                    <div className="col-span-2">
                      <Label className="text-xs text-gray-500">ACS URL</Label>
                      <Input value={acsUrl} onChange={e => setAcsUrl(e.target.value)} className="mt-1 font-mono" />
                    </div>
                    <div>
                      <Label className="text-xs text-gray-500">ACS Username</Label>
                      <Input value={acsUsername} onChange={e => setAcsUsername(e.target.value)} className="mt-1 font-mono" />
                    </div>
                    <div>
                      <Label className="text-xs text-gray-500">ACS Password</Label>
                      <Input value={acsPassword} onChange={e => setAcsPassword(e.target.value)} className="mt-1 font-mono" />
                    </div>
                  </div>
                )}
                <div className="flex items-center justify-between rounded-md bg-gray-50 dark:bg-gray-950 px-3 py-2">
                  <div>
                    <div className="text-sm text-gray-900 dark:text-gray-100">Firewall</div>
                    <div className="text-xs text-gray-500">Tambahkan firewall enable level ... anti-hack disable.</div>
                  </div>
                  <Switch checked={enableFirewall} onCheckedChange={setEnableFirewall} />
                </div>
                {enableFirewall && (
                  <div>
                    <Label className="text-xs text-gray-500">Firewall Level</Label>
                    <Select value={firewallLevel} onValueChange={setFirewallLevel}>
                      <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="low">low</SelectItem>
                        <SelectItem value="medium">medium</SelectItem>
                        <SelectItem value="high">high</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
                <div className="flex items-center justify-between rounded-md bg-gray-50 dark:bg-gray-950 px-3 py-2">
                  <div>
                    <div className="text-sm text-gray-900 dark:text-gray-100">Security Management</div>
                    <div className="text-xs text-gray-500">Tambahkan security-mgmt forward mode.</div>
                  </div>
                  <Switch checked={enableSecurityMgmt} onCheckedChange={setEnableSecurityMgmt} />
                </div>
              </div>
            )}

            {serviceTemplate === 'huawei_full' && (
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-4">
                <div className="text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wide">Huawei Full Template</div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-gray-500">Mgmt VLAN</Label>
                    <Input type="number" min={1} max={4094} value={mgmtVlan} onChange={e => setMgmtVlan(parseInt(e.target.value) || 1010)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">Internet VLAN</Label>
                    <Input type="number" min={1} max={4094} value={internetVlan} onChange={e => setInternetVlan(parseInt(e.target.value) || 30)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">VoIP VLAN</Label>
                    <Input type="number" min={1} max={4094} value={voipVlan} onChange={e => setVoipVlan(parseInt(e.target.value) || 151)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">VLAN Profile</Label>
                    <Input value={vlanProfile} onChange={e => setVlanProfile(e.target.value || 'genieacs')} className="mt-1 font-mono" />
                  </div>
                </div>
              </div>
            )}

            {serviceTemplate === 'fiberhome_veip' && (
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-4">
                <div className="text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wide">Fiberhome VEIP Template</div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-gray-500">TR-069 VLAN</Label>
                    <Input type="number" min={1} max={4094} value={tr069Vlan} onChange={e => setTr069Vlan(parseInt(e.target.value) || 100)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">Internet VLAN</Label>
                    <Input type="number" min={1} max={4094} value={internetVlan} onChange={e => setInternetVlan(parseInt(e.target.value) || 30)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">VoIP VLAN</Label>
                    <Input type="number" min={1} max={4094} value={voipVlan} onChange={e => setVoipVlan(parseInt(e.target.value) || 151)} className="mt-1 font-mono" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="col-span-2">
                    <Label className="text-xs text-gray-500">ACS URL</Label>
                    <Input value={acsUrl} onChange={e => setAcsUrl(e.target.value)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">ACS Username</Label>
                    <Input value={acsUsername} onChange={e => setAcsUsername(e.target.value)} className="mt-1 font-mono" />
                  </div>
                  <div>
                    <Label className="text-xs text-gray-500">ACS Password</Label>
                    <Input value={acsPassword} onChange={e => setAcsPassword(e.target.value)} className="mt-1 font-mono" />
                  </div>
                </div>
              </div>
            )}
          </>)}

          {(!isZTE || serviceTemplate === 'basic') && (
            <div>
              <Label className="text-xs text-gray-500">Service VLAN</Label>
              <Input type="number" min={1} max={4094} value={vlan}
                onChange={e => setVlan(parseInt(e.target.value) || 100)}
                className="mt-1 font-mono caret-gray-900 dark:caret-white" />
            </div>
          )}

          {/* -- Huawei-only fields -- */}
          {isHuawei && (<>
            <div>
              <Label className="text-xs text-gray-500">ONT Line Profile ID</Label>
              <Input type="number" min={1} value={lineProfileId}
                onChange={e => setLineProfileId(parseInt(e.target.value) || 1)}
                className="mt-1 font-mono" />
              <p className="text-xs text-gray-400 mt-1">ont-lineprofile-id -- defines GEM/TCONT mapping</p>
            </div>
            <div>
              <Label className="text-xs text-gray-500">ONT Service Profile ID</Label>
              <Input type="number" min={1} value={srvProfileId}
                onChange={e => setSrvProfileId(parseInt(e.target.value) || 1)}
                className="mt-1 font-mono" />
              <p className="text-xs text-gray-400 mt-1">ont-srvprofile-id -- defines port/service config</p>
            </div>
          </>)}

          {/* -- FiberHome-only fields -- */}
          {isFiberHome && (<>
            <div>
              <Label className="text-xs text-gray-500">ONU Type</Label>
              {metadata.onuTypes.length > 0 ? (
                <Select value={effectiveOnuType} onValueChange={setOnuType}>
                  <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {metadata.onuTypes.map(type => (
                      <SelectItem key={type} value={type}>{type}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input value={onuType} onChange={e => setOnuType(e.target.value)} placeholder="ONU type from OLT" className="mt-1 font-mono" />
              )}
            </div>
            <div>
              <Label className="text-xs text-gray-500">Service Profile Name</Label>
              <Input value={profileName}
                onChange={e => setProfileName(e.target.value || 'default')}
                placeholder="default"
                className="mt-1 font-mono" />
              <p className="text-xs text-gray-400 mt-1">Must match a profile already configured on the OLT</p>
            </div>
          </>)}

          {/* Description */}
          <div>
            <Label className="text-xs text-gray-500">Description / Customer Name (optional)</Label>
            <Input value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="e.g. customer-name"
              className="mt-1" />
          </div>

          <div className="rounded-md border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950 p-3 text-xs text-gray-500 space-y-1">
            <div>Detected ONU type: <span className="font-mono text-gray-900 dark:text-gray-200">{metadata.detectedOnuType ?? onu.onuType ?? '-'}</span></div>
            {isZTE && <div>Template: <span className="font-mono text-gray-900 dark:text-gray-200">{serviceTemplate}</span></div>}
            <div>Metadata source: <span className="font-mono text-gray-900 dark:text-gray-200">{metadataLoading ? 'Loading from OLT...' : 'Live OLT query'}</span></div>
          </div>

          {/* Command preview */}
          <div className="bg-gray-950 dark:bg-black rounded-md border border-gray-800 p-3 text-xs font-mono text-green-400 space-y-0.5 select-text">
            <div className="text-gray-500 mb-1.5 text-[10px] uppercase tracking-widest">Telnet preview . {v}</div>
            {cmdPreview.map((line, i) => (
              <div key={i} className="leading-5">{line}</div>
            ))}
            <div className="flex items-center gap-0.5 mt-1">
              <span className="text-green-400 opacity-70">#</span>
              <span className="inline-block w-2 h-3.5 bg-green-400 opacity-70 ml-0.5 animate-pulse" />
            </div>
          </div>

          {/* Result */}
          {result && (
            <div className={`px-3 py-2 rounded-md text-sm ${result.ok
              ? 'bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-400'
              : 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400'}`}>
              {result.ok ? '[OK] ' : '[FAIL] '}{result.msg}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 px-5 py-4 border-t dark:border-gray-800 flex-shrink-0">
          <Button variant="outline" onClick={onClose} disabled={loading}>Cancel</Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || metadataLoading || !onu.serialNumber || (isZTE || isFiberHome ? !effectiveOnuType : false)}
            className="bg-green-600 hover:bg-green-700 text-white"
          >
            {loading
              ? <><RefreshCw className="h-3 w-3 mr-2 animate-spin" /> Registering...</>
              : <><Plus className="h-3 w-3 mr-1" /> Register ONU</>}
          </Button>
        </div>
      </div>
    </div>
  );
}

function ONUDetailModal({ oltId, onu, onClose }: { oltId: string; onu: ONU; onClose: () => void }) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [detail, setDetail] = useState<any>(null);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/olt/${oltId}/onus/${onu.id}/detail`)
      .then(async (res) => {
        const json = await res.json();
        if (!res.ok || !json.success) throw new Error(json.error ?? 'Failed to load ONU detail');
        setDetail(json);
      })
      .catch((e: any) => setError(e.message))
      .finally(() => setLoading(false));
  }, [oltId, onu.id]);

  const parsed = detail?.telnet?.detail?.parsed ?? {};
  const detailSummary = detail?.telnet?.detail?.summary ?? {};
  const configSummary = detail?.telnet?.config?.summary ?? {};
  const trafficSummary = detail?.telnet?.traffic?.summary ?? {};
  const customer = detail?.onu?.customer;
  const bandwidth = detail?.onu?.bandwidth;
  const tr069 = detail?.tr069;
  const oltProfiles = detail?.oltProfiles ?? {};
  const oltTcontProfiles: any[] = Array.isArray(oltProfiles.tcont) ? oltProfiles.tcont : [];
  const oltTrafficProfiles: any[] = Array.isArray(oltProfiles.traffic) ? oltProfiles.traffic : [];
  const buildScript: string = detail?.buildScript ?? '';
  // Find the specific TCONT profile used by this ONU
  const usedTcontName = Array.isArray(configSummary.tcontProfiles) && configSummary.tcontProfiles.length > 0
    ? configSummary.tcontProfiles[0] : null;
  const usedTcontProfile = usedTcontName
    ? oltTcontProfiles.find((p: any) => p.name === usedTcontName) : null;
  // Active non-default traffic (downstream) profiles
  const activeTrafficProfiles = oltTrafficProfiles.filter((p: any) => p.name !== 'default');
  const detailItems = [
    ['Interface', detail?.telnet?.interface ?? `${onu.frame}/${onu.slot}/${onu.port}:${onu.onuId}`],
    ['Serial Number', parsed['Serial number'] ?? onu.serialNumber ?? 'N/A'],
    ['Name', parsed['Name'] ?? onu.description ?? 'N/A'],
    ['Type', parsed['Type'] ?? 'N/A'],
    ['ONT Vendor', detailSummary.vendor ?? 'N/A'],
    ['State', parsed['State'] ?? onu.status],
    ['Phase', parsed['Phase state'] ?? 'N/A'],
    ['Config', parsed['Config state'] ?? 'N/A'],
    ['Distance', parsed['ONU Distance'] ?? (onu.distance !== null ? `${onu.distance}m` : 'N/A')],
    ['Online Duration', parsed['Online Duration'] ?? 'N/A'],
    ['RX Power (OLT->ONU)', onu.rxPower !== null ? `${onu.rxPower.toFixed(2)} dBm` : (parsed['Rx optical power'] ?? 'N/A')],
    ['TX Power (ONU->OLT)', onu.txPower !== null ? `${onu.txPower.toFixed(2)} dBm` : (parsed['Tx optical power'] ?? 'N/A')],
  ];
  const technicalItems = [
    ['Auth Mode', detailSummary.authenticationMode || 'N/A'],
    ['SN Bind', detailSummary.snBind || 'N/A'],
    ['Admin State', detailSummary.adminState || 'N/A'],
    ['ONU Status', detailSummary.onuStatus || 'N/A'],
    ['Current Channel', detailSummary.currentChannel || 'N/A'],
    ['Configured Channel', detailSummary.configuredChannel || 'N/A'],
    ['DBA Mode', detailSummary.dbaMode || 'N/A'],
    ['Vport Mode', detailSummary.vportMode || 'N/A'],
    ['Line Profile', detailSummary.lineProfile || 'N/A'],
    ['Service Profile', detailSummary.serviceProfile || 'N/A'],
    ['OMCI BW Profile', detailSummary.omciBwProfile || 'N/A'],
    ['FEC', detailSummary.fec || 'N/A'],
    ['Description', detailSummary.description || parsed['Description'] || 'N/A'],
    ['Multicast Encrypt', detailSummary.multicastEncryption || 'N/A'],
  ];
  const serviceVlans = Array.isArray(configSummary.serviceVlans) && configSummary.serviceVlans.length > 0
    ? configSummary.serviceVlans.join(', ')
    : 'N/A';
  const tcontProfiles = Array.isArray(configSummary.tcontProfiles) && configSummary.tcontProfiles.length > 0
    ? configSummary.tcontProfiles.join(', ')
    : 'N/A';
  const downstreamProfiles = Array.isArray(configSummary.downstreamProfiles) && configSummary.downstreamProfiles.length > 0
    ? configSummary.downstreamProfiles.join(', ')
    : '(dari TCONT)';
  const authHistory: any[] = Array.isArray(detailSummary.authHistory) ? detailSummary.authHistory : [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-gray-800">
          <div>
            <h2 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
              <Eye className="h-4 w-4 text-blue-500" /> Detail ONU
            </h2>
            <p className="text-xs text-gray-500 font-mono mt-0.5">{detail?.telnet?.interface ?? `${onu.frame}/${onu.slot}/${onu.port}:${onu.onuId}`}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"><X className="h-5 w-5" /></button>
        </div>
        <div className="p-5 overflow-y-auto space-y-4">
          {loading && <div className="py-10 text-center text-gray-400"><RefreshCw className="h-6 w-6 mx-auto animate-spin mb-2" />Loading detail...</div>}
          {error && <div className="px-3 py-2 rounded bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300 text-sm">{error}</div>}
          {!loading && !error && detail && (
            <>
              {/* ONU Basic Info */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {detailItems.map(([label, value]) => (
                  <div key={label} className="p-3 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500">{label}</div>
                    <div className="text-sm font-medium text-gray-900 dark:text-white mt-1 break-words">{value}</div>
                  </div>
                ))}
              </div>

              {/* Technical Details */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {technicalItems.map(([label, value]) => (
                  <div key={label} className="p-3 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500">{label}</div>
                    <div className="text-sm font-medium text-gray-900 dark:text-white mt-1 break-words">{value}</div>
                  </div>
                ))}
              </div>

              {/* Traffic Statistics */}
              {(trafficSummary.inputRate || trafficSummary.outputRate) && (
                <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3 space-y-3">
                  <div className="text-xs font-semibold text-gray-500 uppercase">Traffic Statistics (Realtime)</div>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
                    <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-blue-500">Downstream Rate</div>
                      <div className="mt-1 font-mono font-medium text-blue-700 dark:text-blue-300">{trafficSummary.outputRate ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-green-500">Upstream Rate</div>
                      <div className="mt-1 font-mono font-medium text-green-700 dark:text-green-300">{trafficSummary.inputRate ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">DL Bandwidth %</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.outputBandwidth ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">UL Bandwidth %</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.inputBandwidth ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">DL Peak Rate</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.outputPeakRate ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">UL Peak Rate</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.inputPeakRate ?? 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">Total DL Bytes</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.totalOutputBytes ? Number(trafficSummary.totalOutputBytes).toLocaleString() : 'N/A'}</div>
                    </div>
                    <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                      <div className="text-[10px] uppercase tracking-wide text-gray-500">Total UL Bytes</div>
                      <div className="mt-1 font-mono font-medium text-gray-700 dark:text-gray-300">{trafficSummary.totalInputBytes ? Number(trafficSummary.totalInputBytes).toLocaleString() : 'N/A'}</div>
                    </div>
                  </div>
                </div>
              )}

              {/* ONT Build Configuration */}
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3 space-y-3">
                <div className="text-xs font-semibold text-gray-500 uppercase">ONT Build Configuration</div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-sm">
                  <div className="rounded-lg bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500">Service VLANs</div>
                    <div className="mt-1 font-medium text-gray-900 dark:text-white break-words">{serviceVlans}</div>
                  </div>
                  <div className="rounded-lg bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 p-3">
                    <div className="text-[10px] uppercase tracking-wide text-orange-500">TCONT / DBA Profile (Upload)</div>
                    <div className="mt-1 font-medium text-orange-900 dark:text-orange-200 font-mono">{tcontProfiles}</div>
                    {usedTcontProfile && (
                      <div className="mt-1 text-xs text-orange-600 dark:text-orange-400">
                        Type {usedTcontProfile.bwType} -- Max {usedTcontProfile.mbwKbps >= 1000 ? `${(usedTcontProfile.mbwKbps / 1000).toFixed(0)} Mbps` : `${usedTcontProfile.mbwKbps} Kbps`}
                      </div>
                    )}
                  </div>
                  <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
                    <div className="text-[10px] uppercase tracking-wide text-blue-500">Traffic Profile (Download)</div>
                    {activeTrafficProfiles.length > 0 ? (
                      activeTrafficProfiles.map((p: any) => (
                        <div key={p.name}>
                          <div className="mt-1 font-medium text-blue-900 dark:text-blue-200 font-mono">{p.name}</div>
                          <div className="mt-0.5 text-xs text-blue-600 dark:text-blue-400">
                            SIR {p.sirKbps >= 1000 ? `${(p.sirKbps / 1000).toFixed(0)} Mbps` : `${p.sirKbps} Kbps`} / PIR {p.pirKbps >= 1000 ? `${(p.pirKbps / 1000).toFixed(0)} Mbps` : `${p.pirKbps} Kbps`}
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="mt-1 font-medium text-gray-900 dark:text-white break-words">{downstreamProfiles}</div>
                    )}
                  </div>
                </div>

                {/* Bandwidth / Traffic Profile dari PPPoE */}
                {bandwidth && (
                  <div>
                    <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">Traffic Profile (PPPoE Radius)</div>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-sm">
                      <div className="rounded-lg bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-800 p-3">
                        <div className="text-[10px] uppercase tracking-wide text-purple-500">Nama Profil</div>
                        <div className="mt-1 font-medium text-purple-900 dark:text-purple-200">{bandwidth.profileName}</div>
                      </div>
                      <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
                        <div className="text-[10px] uppercase tracking-wide text-blue-500">Download Limit</div>
                        <div className="mt-1 font-mono font-medium text-blue-700 dark:text-blue-300">
                          {bandwidth.downloadSpeed > 0
                            ? (bandwidth.downloadSpeed >= 1024
                              ? `${(bandwidth.downloadSpeed / 1024).toFixed(0)} Mbps`
                              : `${bandwidth.downloadSpeed} Kbps`)
                            : 'N/A'}
                        </div>
                      </div>
                      <div className="rounded-lg bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 p-3">
                        <div className="text-[10px] uppercase tracking-wide text-green-500">Upload Limit</div>
                        <div className="mt-1 font-mono font-medium text-green-700 dark:text-green-300">
                          {bandwidth.uploadSpeed > 0
                            ? (bandwidth.uploadSpeed >= 1024
                              ? `${(bandwidth.uploadSpeed / 1024).toFixed(0)} Mbps`
                              : `${bandwidth.uploadSpeed} Kbps`)
                            : 'N/A'}
                        </div>
                      </div>
                    </div>
                    {bandwidth.rateLimit && (
                      <div className="mt-1 text-xs text-gray-500 font-mono">Rate-limit: {bandwidth.rateLimit}</div>
                    )}
                  </div>
                )}

                {Array.isArray(configSummary.servicePorts) && configSummary.servicePorts.length > 0 && (
                  <div className="overflow-x-auto">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">Service Ports</div>
                    <table className="w-full text-sm border-separate border-spacing-y-1">
                      <thead>
                        <tr className="text-left text-[10px] uppercase tracking-wide text-gray-500">
                          <th className="px-2 py-1">Port</th>
                          <th className="px-2 py-1">VPort</th>
                          <th className="px-2 py-1">User VLAN</th>
                          <th className="px-2 py-1">VLAN</th>
                        </tr>
                      </thead>
                      <tbody>
                        {configSummary.servicePorts.map((sp: any) => (
                          <tr key={`${sp.servicePort}-${sp.vport}`} className="bg-gray-50 dark:bg-gray-950">
                            <td className="px-2 py-2 rounded-l border-y border-l border-gray-200 dark:border-gray-800 font-mono">{sp.servicePort}</td>
                            <td className="px-2 py-2 border-y border-gray-200 dark:border-gray-800 font-mono">{sp.vport}</td>
                            <td className="px-2 py-2 border-y border-gray-200 dark:border-gray-800 font-mono">{sp.userVlan}</td>
                            <td className="px-2 py-2 rounded-r border-y border-r border-gray-200 dark:border-gray-800 font-mono">{sp.vlan}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}

                {/* TCONT Table */}
                {Array.isArray(configSummary.tconts) && configSummary.tconts.length > 0 && (
                  <div className="overflow-x-auto">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">TCONT Config</div>
                    <table className="w-full text-sm border-separate border-spacing-y-1">
                      <thead>
                        <tr className="text-left text-[10px] uppercase tracking-wide text-gray-500">
                          <th className="px-2 py-1">TCONT</th>
                          <th className="px-2 py-1">Upstream Profile</th>
                        </tr>
                      </thead>
                      <tbody>
                        {configSummary.tconts.map((t: any) => (
                          <tr key={t.tcontId} className="bg-gray-50 dark:bg-gray-950">
                            <td className="px-2 py-2 rounded-l border-y border-l border-gray-200 dark:border-gray-800 font-mono">{t.tcontId}</td>
                            <td className="px-2 py-2 rounded-r border-y border-r border-gray-200 dark:border-gray-800 font-mono">{t.profile || 'N/A'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}

                {/* GEM Port Table */}
                {Array.isArray(configSummary.gemports) && configSummary.gemports.length > 0 && (
                  <div className="overflow-x-auto">
                    <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">GEM Ports</div>
                    <table className="w-full text-sm border-separate border-spacing-y-1">
                      <thead>
                        <tr className="text-left text-[10px] uppercase tracking-wide text-gray-500">
                          <th className="px-2 py-1">GEM Port</th>
                          <th className="px-2 py-1">TCONT</th>
                          <th className="px-2 py-1">DL Profile</th>
                        </tr>
                      </thead>
                      <tbody>
                        {configSummary.gemports.map((g: any) => (
                          <tr key={g.gemPort} className="bg-gray-50 dark:bg-gray-950">
                            <td className="px-2 py-2 rounded-l border-y border-l border-gray-200 dark:border-gray-800 font-mono">{g.gemPort}</td>
                            <td className="px-2 py-2 border-y border-gray-200 dark:border-gray-800 font-mono">{g.tcont}</td>
                            <td className="px-2 py-2 rounded-r border-y border-r border-gray-200 dark:border-gray-800 font-mono">{g.profile || '(dari TCONT)'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>

              {/* Auth History */}
              {authHistory.length > 0 && (
                <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3">
                  <div className="text-xs font-semibold text-gray-500 uppercase mb-2">Riwayat Koneksi ONU</div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm border-separate border-spacing-y-1">
                      <thead>
                        <tr className="text-left text-[10px] uppercase tracking-wide text-gray-500">
                          <th className="px-2 py-1">#</th>
                          <th className="px-2 py-1">Auth / Online</th>
                          <th className="px-2 py-1">Offline</th>
                          <th className="px-2 py-1">Cause</th>
                        </tr>
                      </thead>
                      <tbody>
                        {authHistory.map((h: any) => (
                          <tr key={h.index} className="bg-gray-50 dark:bg-gray-950">
                            <td className="px-2 py-1.5 rounded-l border-y border-l border-gray-200 dark:border-gray-800 font-mono">{h.index}</td>
                            <td className="px-2 py-1.5 border-y border-gray-200 dark:border-gray-800 font-mono text-green-600 dark:text-green-400">{h.authpassAt}</td>
                            <td className="px-2 py-1.5 border-y border-gray-200 dark:border-gray-800 font-mono text-red-600 dark:text-red-400">{h.offlineAt === '0000-00-00 00:00:00' ? 'Masih Online' : h.offlineAt}</td>
                            <td className="px-2 py-1.5 rounded-r border-y border-r border-gray-200 dark:border-gray-800 font-mono">{h.offlineCause || '-'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* Customer Assignment */}
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3">
                <div className="text-xs font-semibold text-gray-500 uppercase mb-2">Customer Assignment</div>
                {customer ? (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
                    <div><span className="text-gray-500">Name:</span> {customer.name}</div>
                    <div><span className="text-gray-500">Username:</span> {customer.username}</div>
                    <div><span className="text-gray-500">Phone:</span> {customer.phone ?? '-'}</div>
                    <div><span className="text-gray-500">Profile:</span> {customer.profile?.name ?? '-'}</div>
                    <div><span className="text-gray-500">Area:</span> {customer.area?.name ?? '-'}</div>
                    <div><span className="text-gray-500">ODP:</span> {customer.odpAssignment?.odp?.name ?? '-'}</div>
                    <div><span className="text-gray-500">ODP Port:</span> {customer.odpAssignment?.portNumber ?? '-'}</div>
                    <div><span className="text-gray-500">Status:</span> {customer.status}</div>
                  </div>
                ) : <div className="text-sm text-gray-400">Belum terhubung ke customer.</div>}
              </div>

              {/* TR-069 / GenieACS */}
              <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3">
                <div className="text-xs font-semibold text-gray-500 uppercase mb-2">TR-069 / GenieACS</div>
                {tr069?.configured ? (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm text-green-700 dark:text-green-400">
                      <span className="inline-block w-2 h-2 rounded-full bg-green-500"></span>
                      GenieACS terkonfigurasi -- <span className="font-mono text-xs">{tr069.host}</span>
                    </div>
                    {customer && (
                      <div className="text-xs text-gray-500">
                        Untuk melihat perangkat TR-069 customer ini ({customer.username}), kunjungi halaman GenieACS di Settings.
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <span className="inline-block w-2 h-2 rounded-full bg-gray-400"></span>
                    GenieACS belum dikonfigurasi -- fitur TR-069 memerlukan konfigurasi ACS di halaman Settings &rarr; GenieACS.
                  </div>
                )}
              </div>

              {/* Build Script */}
              {buildScript && (
                <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-3">
                  <div className="text-xs font-semibold text-gray-500 uppercase mb-2">Build Script (Reproducible Config)</div>
                  <pre className="text-[11px] font-mono bg-gray-950 text-yellow-300 rounded-lg p-3 overflow-auto max-h-96 whitespace-pre">{buildScript}</pre>
                </div>
              )}

              {/* Raw Telnet Output */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                <div>
                  <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">Detail Info (Raw)</div>
                  <pre className="text-[11px] font-mono bg-gray-950 text-green-300 rounded-lg p-3 overflow-auto max-h-64 whitespace-pre-wrap">{detail?.telnet?.detail?.raw || 'No detail output'}</pre>
                </div>
                <div>
                  <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">Running Config (Raw)</div>
                  <pre className="text-[11px] font-mono bg-gray-950 text-blue-300 rounded-lg p-3 overflow-auto max-h-64 whitespace-pre-wrap">{detail?.telnet?.config?.raw || 'No config output'}</pre>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function ONUAssignModal({ oltId, onu, onClose, onSuccess }: { oltId: string; onu: ONU; onClose: () => void; onSuccess: () => void }) {
  const [customers, setCustomers] = useState<any[]>([]);
  const [selectedCustomerId, setSelectedCustomerId] = useState(onu.customer?.id ?? '');
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadCustomers = useCallback(() => {
    setLoading(true);
    fetch(`/api/olt/${oltId}/onus/${onu.id}/assign${query ? `?q=${encodeURIComponent(query)}` : ''}`)
      .then(async (res) => {
        const json = await res.json();
        if (!res.ok || !json.success) throw new Error(json.error ?? 'Failed to load customers');
        setCustomers(json.customers ?? []);
        if (json.currentCustomer?.id) setSelectedCustomerId(json.currentCustomer.id);
      })
      .catch((e: any) => setError(e.message))
      .finally(() => setLoading(false));
  }, [oltId, onu.id, query]);

  useEffect(() => {
    const t = setTimeout(loadCustomers, 250);
    return () => clearTimeout(t);
  }, [loadCustomers]);

  const save = async (customerId: string | null) => {
    setSaving(true); setError(null);
    try {
      const res = await fetch(`/api/olt/${oltId}/onus/${onu.id}/assign`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ customerId }),
      });
      const json = await res.json();
      if (!res.ok || !json.success) throw new Error(json.error ?? 'Failed to assign customer');
      await onSuccess();
      onClose();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-lg overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-gray-800">
          <div>
            <h2 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2"><UserPlus className="h-4 w-4 text-indigo-500" /> Assign Customer</h2>
            <p className="text-xs text-gray-500 font-mono mt-0.5">{onu.serialNumber ?? `${onu.frame}/${onu.slot}/${onu.port}:${onu.onuId}`}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"><X className="h-5 w-5" /></button>
        </div>
        <div className="p-5 space-y-4">
          <div>
            <Label className="text-xs text-gray-500">Search Customer</Label>
            <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="name, username, phone, customer ID" className="mt-1" />
          </div>
          <div>
            <Label className="text-xs text-gray-500">Customer</Label>
            <Select value={selectedCustomerId || 'none'} onValueChange={(v) => setSelectedCustomerId(v === 'none' ? '' : v)}>
              <SelectTrigger className="mt-1"><SelectValue placeholder={loading ? 'Loading...' : 'Select customer'} /></SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Unassigned</SelectItem>
                {customers.map((customer) => (
                  <SelectItem key={customer.id} value={customer.id}>{customer.name} - {customer.username}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {error && <div className="px-3 py-2 rounded bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300 text-sm">{error}</div>}
        </div>
        <div className="flex justify-between gap-2 px-5 py-4 border-t dark:border-gray-800">
          <Button variant="outline" onClick={() => save(null)} disabled={saving}>Unassign</Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
            <Button onClick={() => save(selectedCustomerId || null)} disabled={saving || loading}>
              {saving ? <><RefreshCw className="h-3 w-3 mr-2 animate-spin" />Saving...</> : 'Save Assignment'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function OLTDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const searchParams = useSearchParams();
  const urlFilter = searchParams.get('filter');
  const [olt, setOlt] = useState<OLTDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [polling, setPolling] = useState(false);
  const [deletingOnu, setDeletingOnu] = useState<string | null>(null);
  const [testing, setTesting] = useState<string | null>(null);
  const [onuStatusFilter, setOnuStatusFilter] = useState(urlFilter ?? 'all');

  // Real-time ONU list (auto-refreshes every 30s)
  const [liveOnus, setLiveOnus] = useState<ONU[] | null>(null);
  const [onuLastRefresh, setOnuLastRefresh] = useState<Date | null>(null);
  const [onuRefreshSecs, setOnuRefreshSecs] = useState(0);

  // Batch reboot
  const [selectedOnus, setSelectedOnus] = useState<Set<string>>(new Set());
  const [rebootingOnu, setRebootingOnu] = useState<string | null>(null);
  const [confirmReboot, setConfirmReboot] = useState<string | null>(null);
  const [batchRebooting, setBatchRebooting] = useState(false);
  const [batchProgress, setBatchProgress] = useState<{
    current: number; total: number;
    results: { serialNumber: string; success: boolean; error?: string }[];
  } | null>(null);

  // ONU registration
  const [registeringOnu, setRegisteringOnu] = useState<ONU | null>(null);
  const [detailOnu, setDetailOnu] = useState<ONU | null>(null);
  const [assigningOnu, setAssigningOnu] = useState<ONU | null>(null);
  const [editingOnu, setEditingOnu] = useState<ONU | null>(null);

  // Clean config ONU
  const [cleaningConfigOnu, setCleaningConfigOnu] = useState<string | null>(null);

  // ODP assignment
  const [assigningOdpToOnu, setAssigningOdpToOnu] = useState<ONU | null>(null);

  // Settings state
  const [settings, setSettings] = useState({
    vendor: 'huawei',
    model: '',
    firmwareVersion: '',
    monitoringEnabled: false,
    snmpEnabled: true,
    snmpCommunity: 'public',
    snmpPort: 161,
    sshEnabled: false,
    sshPort: 22,
    telnetEnabled: false,
    telnetPort: 23,
    username: '',
    password: '',
    pollingInterval: 300,
    routerIds: [] as string[],
  });
  const [routerList, setRouterList] = useState<{ id: string; name: string; ipAddress: string }[]>([]);

  const fetchOLT = useCallback(async () => {
    try {
      const res = await fetch(`/api/olt/${id}`);
      if (res.ok) {
        const data = await res.json();
        const o = data.olt;
        o.onuStatuses = o.onuStatuses ?? [];
        o.alerts = o.alerts ?? [];
        o.routers = o.routers ?? [];
        o.monitoringLogs = o.monitoringLogs ?? [];
        o.performanceMetrics = o.performanceMetrics ?? [];
        setOlt(o);
        setSettings({
          vendor: o.vendor ?? 'huawei',
          model: o.model ?? '',
          firmwareVersion: o.firmwareVersion ?? '',
          monitoringEnabled: o.monitoringEnabled,
          snmpEnabled: o.snmpEnabled,
          snmpCommunity: o.snmpCommunity,
          snmpPort: o.snmpPort,
          sshEnabled: o.sshEnabled,
          sshPort: o.sshPort,
          telnetEnabled: o.telnetEnabled,
          telnetPort: o.telnetPort,
          username: o.username ?? '',
          password: o.password ?? '',
          pollingInterval: o.pollingInterval,
          routerIds: (o.routers ?? []).map((r: any) => r.routerId),
        });
      } else {
        router.push('/admin/olt/monitoring');
      }
    } catch (e) {
      console.error('Failed to fetch OLT', e);
    } finally {
      setLoading(false);
    }
  }, [id, router]);

  useEffect(() => { fetchOLT(); }, [fetchOLT]);

  // Live ONU list -- fetches from /onus?all=true and shapes data into ONU[]
  const fetchLiveOnus = useCallback(async () => {
    try {
      const res = await fetch(`/api/olt/${id}/onus?all=true`);
      if (!res.ok) return;
      const data = await res.json();
      const shaped: ONU[] = (data.onus ?? []).map((o: any) => ({
        id: o.id,
        frame: o.frame,
        slot: o.slot,
        port: o.port,
        onuId: o.onuId,
        onuType: o.onuType ?? null,
        serialNumber: o.serialNumber ?? null,
        macAddress: o.macAddress ?? null,
        status: o.status,
        description: o.description ?? null,
        rxPower: o.rxPower ?? null,
        txPower: o.txPower ?? null,
        temperature: o.temperature ?? null,
        distance: o.distance ?? null,
        lastSeenAt: o.lastSeenAt ?? null,
        odpId: o.odpId ?? null,
        odpName: o.odpName ?? null,
        customer: o.customerName
          ? { id: o.customerId ?? '', name: o.customerName, username: o.customerUsername ?? '', phone: o.customerPhone ?? '' }
          : null,
      }));
      setLiveOnus(shaped);
      setOnuLastRefresh(new Date());
      setOnuRefreshSecs(0);
    } catch {
      // silently skip on network error
    }
  }, [id]);

  // Start auto-refresh of ONU list every 30s
  useEffect(() => {
    fetchLiveOnus();
    const interval = setInterval(fetchLiveOnus, 15_000);
    return () => clearInterval(interval);
  }, [fetchLiveOnus]);

  // Countdown ticker: increments every second so we can show "Updated Xs ago"
  useEffect(() => {
    if (!onuLastRefresh) return;
    const tick = setInterval(() => setOnuRefreshSecs(s => s + 1), 1_000);
    return () => clearInterval(tick);
  }, [onuLastRefresh]);

  useEffect(() => {
    if (urlFilter) setOnuStatusFilter(urlFilter);
  }, [urlFilter]);

  // Fetch available routers for assignment
  useEffect(() => {
    fetch('/api/network/routers')
      .then((r) => r.json())
      .then((data) => setRouterList(data.routers ?? []))
      .catch(() => {});
  }, []);

  const handleSaveSettings = async () => {
    setSaving(true);
    try {
      const res = await fetch(`/api/olt/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      });
      if (res.ok) {
        await fetchOLT();
      }
    } catch (e) {
      console.error('Failed to save settings', e);
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async (protocol: string) => {
    setTesting(protocol);
    try {
      const res = await fetch('/api/olt/test-connection', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ oltId: id, protocol }),
      });
      const data = await res.json();
      alert(data.message || (data.success ? 'Connection successful' : 'Connection failed'));
    } catch (e) {
      alert('Test failed');
    } finally {
      setTesting(null);
    }
  };

  const handleSyncOLT = useCallback(async ({ silent = false }: { silent?: boolean } = {}) => {
    setPolling(true);
    try {
      const res = await fetch(`/api/olt/${id}/sync`, { method: 'POST' });
      const data = await res.json();
      if (!res.ok) {
        if (!silent) alert(`Sync failed: ${data.error ?? 'Unknown error'}`);
      } else if (data.background) {
        // Sync is running in background -- auto-refresh after 30s
        if (!silent) alert(data.message ?? 'Sync started -- data will refresh automatically');
        setTimeout(async () => {
          await fetchOLT();
          setPolling(false);
        }, 30_000);
        return; // don't clear polling yet -- keep button disabled during wait
      } else {
        await fetchOLT();
        if (!silent) alert(data.message ?? 'OLT sync completed');
      }
    } catch (e) {
      console.error('Sync failed', e);
      if (!silent) alert('Sync failed -- check network connection');
    } finally {
      setPolling(false);
    }
  }, [fetchOLT, id]);

  const handleRebootOnu = async (onuId: string) => {
    setRebootingOnu(onuId);
    try {
      const res = await fetch(`/api/olt/${id}/onus/${onuId}/reboot`, { method: 'POST' });
      const data = await res.json();
      if (!res.ok) {
        alert(data.error ?? 'Reboot failed');
      } else {
        setConfirmReboot(null);
        await handleSyncOLT({ silent: true });
      }
    } catch (e) {
      alert('Reboot request failed');
    } finally {
      setRebootingOnu(null);
    }
  };

  const handleDeleteOnu = async (onu: ONU) => {
    const confirmed = window.confirm(
      `Hapus ONU ${onu.serialNumber ?? `${onu.frame}/${onu.slot}/${onu.port}:${onu.onuId}`} dari OLT?\n\nAksi ini akan deregister ONU dari PON port dan hapus dari database.`
    );
    if (!confirmed) return;

    setDeletingOnu(onu.id);
    try {
      const res = await fetch(`/api/olt/${id}/onus/${onu.id}/delete`, { method: 'DELETE' });
      const data = await res.json();
      if (!res.ok) {
        alert(data.error ?? 'Delete ONU failed');
      } else {
        await fetchLiveOnus();
        await fetchOLT();
      }
    } catch (e) {
      alert('Delete ONU request failed');
    } finally {
      setDeletingOnu(null);
    }
  };

  const handleCleanConfigOnu = async (onu: ONU) => {
    const confirmed = window.confirm(
      `Clean config ONU ${onu.serialNumber ?? `${onu.frame}/${onu.slot}/${onu.port}:${onu.onuId}`}?\n\nAksi ini akan menghapus konfigurasi service (VLAN, profile) di OLT. ONU tetap terdaftar.`
    );
    if (!confirmed) return;

    setCleaningConfigOnu(onu.id);
    try {
      const res = await fetch(`/api/olt/${id}/onus/${onu.id}/clean-config`, { method: 'POST' });
      const data = await res.json();
      if (!res.ok) {
        alert(data.error ?? 'Clean config failed');
      } else {
        alert('Config ONU berhasil di-reset.');
        await fetchLiveOnus();
      }
    } catch (e) {
      alert('Clean config request failed');
    } finally {
      setCleaningConfigOnu(null);
    }
  };

  const handleBatchReboot = async () => {
    if (selectedOnus.size === 0) return;
    setBatchRebooting(true);
    const ids = Array.from(selectedOnus);
    setBatchProgress({ current: 0, total: ids.length, results: [] });
    const results: { serialNumber: string; success: boolean; error?: string }[] = [];
    for (let i = 0; i < ids.length; i++) {
      const onuId = ids[i];
      const onu = olt?.onuStatuses.find((o) => o.id === onuId);
      try {
        const res = await fetch(`/api/olt/${id}/onus/${onuId}/reboot`, { method: 'POST' });
        const data = await res.json();
        results.push({ serialNumber: onu?.serialNumber ?? onuId, success: res.ok, error: !res.ok ? data.error : undefined });
      } catch (e: any) {
        results.push({ serialNumber: onu?.serialNumber ?? onuId, success: false, error: e.message });
      }
      setBatchProgress({ current: i + 1, total: ids.length, results: [...results] });
    }
    setBatchRebooting(false);
    setSelectedOnus(new Set());
  };

  const handleExportCSV = () => {
    if (!olt) return;
    const rows = [
      ['Location', 'Serial Number', 'MAC', 'Status', 'RX Power (dBm)', 'Distance (m)', 'Customer', 'Username', 'Last Seen'],
      ...olt.onuStatuses.map((o) => [
        `${o.frame}/${o.slot}/${o.port}:${o.onuId}`,
        o.serialNumber ?? '',
        o.macAddress ?? '',
        o.status,
        o.rxPower?.toString() ?? '',
        o.distance?.toString() ?? '',
        o.customer?.name ?? '',
        o.customer?.username ?? '',
        o.lastSeenAt ? formatWIB(o.lastSeenAt, 'dd MMM yyyy HH:mm') : '',
      ]),
    ];
    const csv = rows.map((r) => r.map((c) => `"${c}"`).join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `onu-list-${olt.name}-${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleSelectOnu = (onuId: string) => {
    setSelectedOnus((prev) => {
      const next = new Set(prev);
      if (next.has(onuId)) next.delete(onuId); else next.add(onuId);
      return next;
    });
  };

  const handleSelectAll = () => {
    const onus = filteredOnus;
    if (selectedOnus.size === onus.length) {
      setSelectedOnus(new Set());
    } else {
      setSelectedOnus(new Set(onus.map((o) => o.id)));
    }
  };

  // RxPower is OLT-measured upstream power from each ONU (ZTE formula: raw/500 - 30).
  // Verified range from live device: -10 to -20 dBm (typical GPON B+ upstream).
  const getSignalQuality = (rxPower: number | null) => {
    if (rxPower === null) return { label: 'N/A', color: 'text-gray-400' };
    if (rxPower >= -14) return { label: 'Excellent', color: 'text-green-600' };
    if (rxPower >= -18) return { label: 'Good', color: 'text-blue-600' };
    if (rxPower >= -22) return { label: 'Fair', color: 'text-yellow-600' };
    return { label: 'Poor', color: 'text-red-600' };
  };

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  };

  const formatTimestamp = (ts: string) => {
    const d = new Date(ts);
    return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
  };

  const formatUptime = (seconds: number | null) => {
    if (!seconds) return 'N/A';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return `${d}d ${h}h ${m}m`;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':       return 'text-green-600';
      case 'dying_gasp':   return 'text-red-600';
      case 'los':          return 'text-orange-600';
      case 'auth_failed':  return 'text-yellow-600';
      default:             return 'text-gray-500';
    }
  };

  const filteredOnus = (liveOnus ?? olt?.onuStatuses ?? []).filter((o) =>
    onuStatusFilter === 'all' || o.status === onuStatusFilter
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!olt) return null;

  return (
    <div className="space-y-5 p-1">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link href="/admin/olt/monitoring">
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div>
            <h1 className="text-lg font-semibold flex items-center gap-2">
              {olt.isOnline
                ? <Wifi className="h-5 w-5 text-green-500" />
                : <WifiOff className="h-5 w-5 text-red-500" />}
              {olt.name}
              {olt.vendor && (
                <span className="text-xs font-normal bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 px-2 py-0.5 rounded-full uppercase">
                  {olt.vendor}
                </span>
              )}
            </h1>
            <p className="text-gray-500 dark:text-gray-400 font-mono text-xs mt-0.5">
              {olt.ipAddress}
              {olt.model && <span className="ml-2 text-gray-400">. {olt.model}</span>}
              {olt.firmwareVersion && <span className="ml-2 text-gray-400">. {olt.firmwareVersion}</span>}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button onClick={handleExportCSV} variant="outline" size="sm">
            <Download className="h-4 w-4 mr-1" />
            Export CSV
          </Button>
          <Button onClick={() => handleSyncOLT()} variant="outline" size="sm" disabled={polling}>
            <RefreshCw className={`h-4 w-4 mr-2 ${polling ? 'animate-spin' : ''}`} />
            {polling ? 'Syncing...' : 'Sync OLT'}
          </Button>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-3 gap-3">
        <Card className={`border-l-4 ${olt.isOnline ? 'border-l-green-500' : 'border-l-red-500'}`}>
          <CardContent className="pl-6 pr-4 py-4">
            <div className="flex items-center gap-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 mb-2 uppercase tracking-wide">
              <Wifi className="h-3.5 w-3.5" /> Status
            </div>
            <div className={`font-bold text-2xl leading-tight ${olt.isOnline ? 'text-green-600' : 'text-red-600'}`}>
              {olt.isOnline ? 'Online' : 'Offline'}
            </div>
            <div className="text-xs text-gray-400 mt-1">
              {olt.lastPollAt ? `Polled ${new Date(olt.lastPollAt).toLocaleTimeString('id-ID')}` : 'Not polled yet'}
            </div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-blue-500">
          <CardContent className="pl-6 pr-4 py-4">
            <div className="flex items-center gap-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 mb-2 uppercase tracking-wide">
              <Clock className="h-3.5 w-3.5" /> Uptime
            </div>
            <div className="font-bold text-2xl leading-tight text-gray-900 dark:text-white">{formatUptime(olt.uptime)}</div>
            <div className="text-xs text-gray-400 mt-1">
              {olt.vendor ?? 'OLT'} {olt.model ?? ''}
            </div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-teal-500">
          <CardContent className="pl-6 pr-4 py-4">
            <div className="flex items-center gap-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 mb-2 uppercase tracking-wide">
              <Activity className="h-3.5 w-3.5" /> ONUs
            </div>
            <div className="font-bold text-2xl leading-tight">
              <span className="text-green-600">{olt.onlineOnu}</span>
              <span className="text-gray-400 text-lg font-semibold">/{olt.totalOnu}</span>
            </div>
            <div className="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs">
              {olt.offlineOnu > 0 && <span className="text-orange-500 font-medium">{olt.offlineOnu} offline</span>}
              {(liveOnus ?? olt.onuStatuses).filter(o => o.status === 'los').length > 0 && (
                <span className="text-red-500 font-medium">{(liveOnus ?? olt.onuStatuses).filter(o => o.status === 'los').length} LOS</span>
              )}
              {(liveOnus ?? olt.onuStatuses).filter(o => o.status === 'dying_gasp').length > 0 && (
                <span className="text-orange-600 font-medium">{(liveOnus ?? olt.onuStatuses).filter(o => o.status === 'dying_gasp').length} DyingGasp</span>
              )}
              {olt.offlineOnu === 0 && <span className="text-green-500 font-medium">All online</span>}
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="onus">
        <TabsList>
          <TabsTrigger value="onus">ONU List ({olt.totalOnu})</TabsTrigger>
          <TabsTrigger value="portmap">Port Map</TabsTrigger>
          <TabsTrigger value="alerts">
            Alerts
            {olt.alerts.filter((a: any) => !a.isResolved).length > 0 && (
              <Badge variant="destructive" className="ml-2">{olt.alerts.filter((a: any) => !a.isResolved).length}</Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>

        {/* ONU List Tab */}
        <TabsContent value="onus" className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <Select value={onuStatusFilter} onValueChange={setOnuStatusFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Filter status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All ONUs</SelectItem>
                <SelectItem value="online">Online</SelectItem>
                <SelectItem value="offline">Offline</SelectItem>
                <SelectItem value="dying_gasp">Dying Gasp</SelectItem>
                <SelectItem value="los">LOS</SelectItem>
                <SelectItem value="auth_failed">Unregistered</SelectItem>
              </SelectContent>
            </Select>
            <span className="text-sm text-gray-500 self-center">{filteredOnus.length} ONUs</span>
            {onuLastRefresh && (
              <span className="text-xs text-gray-400 self-center flex items-center gap-1">
                <RefreshCw className="w-3 h-3" />
                {onuRefreshSecs < 5
                  ? 'Updated just now'
                  : `Updated ${onuRefreshSecs}s ago`}
              </span>
            )}
            {selectedOnus.size > 0 && (
              <>
                <span className="text-sm font-medium text-blue-600">{selectedOnus.size} selected</span>
                <Button
                  onClick={handleBatchReboot}
                  disabled={batchRebooting}
                  size="sm"
                  className="bg-orange-600 hover:bg-orange-700 text-white"
                >
                  <Power className="w-3 h-3 mr-1" />
                  {batchRebooting ? 'Rebooting...' : `Reboot ${selectedOnus.size} ONUs`}
                </Button>
              </>
            )}
          </div>

          {/* Batch Progress */}
          {batchProgress && (
            <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-blue-900">Processing batch reboot...</span>
                <span className="text-sm text-blue-700">{batchProgress.current} / {batchProgress.total}</span>
              </div>
              <div className="w-full bg-blue-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${(batchProgress.current / batchProgress.total) * 100}%` }}
                />
              </div>
              {batchProgress.results.length > 0 && (
                <div className="mt-3 max-h-32 overflow-y-auto space-y-0.5">
                  {batchProgress.results.map((r, i) => (
                    <div key={i} className={`text-xs py-0.5 ${r.success ? 'text-green-700' : 'text-red-700'}`}>
                      {r.success ? '[OK]' : '[FAIL]'} {r.serialNumber}{r.error && `: ${r.error}`}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-800">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b dark:border-gray-700 text-left text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/60">
                  <th className="py-2.5 pl-3 pr-2">
                    <input
                      type="checkbox"
                      checked={selectedOnus.size === filteredOnus.length && filteredOnus.length > 0}
                      onChange={handleSelectAll}
                      className="w-4 h-4 text-blue-600 rounded"
                    />
                  </th>
                  <th className="py-2.5 pr-4 font-medium">Location</th>
                  <th className="py-2.5 pr-4 font-medium">Serial / MAC</th>
                  <th className="py-2.5 pr-4 font-medium">Name</th>
                  <th className="py-2.5 pr-4 font-medium">Status</th>
                  <th className="py-2.5 pr-4 font-medium">Signal</th>
                  <th className="py-2.5 pr-4 font-medium">RX Power</th>
                  <th className="py-2.5 pr-4 font-medium">Distance</th>
                  <th className="py-2.5 pr-4 font-medium">ODP</th>
                  <th className="py-2.5 pr-4 font-medium">Customer</th>
                  <th className="py-2.5 pr-4 font-medium">Last Seen</th>
                  <th className="py-2.5 pr-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredOnus.map((onu) => {
                  const signalQuality = getSignalQuality(onu.rxPower);
                  return (
                    <tr key={onu.id} className="border-b dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                      <td className="py-2.5 pl-3 pr-2">
                        <input
                          type="checkbox"
                          checked={selectedOnus.has(onu.id)}
                          onChange={() => handleSelectOnu(onu.id)}
                          className="w-4 h-4 text-blue-600 rounded"
                        />
                      </td>
                      <td className="py-2.5 pr-4 font-mono text-xs text-gray-700 dark:text-gray-300">
                        {onu.frame}/{onu.slot}/{onu.port}:{onu.onuId}
                      </td>
                      <td className="py-2.5 pr-4">
                        <div className="font-mono text-xs text-gray-900 dark:text-gray-100">
                          {onu.serialNumber ?? onu.macAddress ?? <span className="text-yellow-600 dark:text-yellow-400">N/A</span>}
                        </div>
                      </td>
                      <td className="py-2.5 pr-4">
                        <div className="text-xs text-gray-700 dark:text-gray-300">{onu.description ?? <span className="text-gray-400">--</span>}</div>
                      </td>
                      <td className="py-2.5 pr-4">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                          onu.status === 'online' ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-400' :
                          onu.status === 'auth_failed' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-400' :
                          onu.status === 'dying_gasp' ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-400' :
                          onu.status === 'los' ? 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-400' :
                          'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                        }`}>
                          {onu.status === 'auth_failed' ? 'Unregistered' : onu.status.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td className="py-2.5 pr-4">
                        <span className={`inline-flex items-center gap-1 text-xs font-medium ${signalQuality.color}`}>
                          <Signal className="w-3 h-3" />
                          {signalQuality.label}
                        </span>
                      </td>
                      <td className="py-2.5 pr-4">
                        {onu.rxPower !== null ? (
                          <span className={`font-mono text-xs font-medium ${onu.rxPower < -27 ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}>
                            {onu.rxPower.toFixed(2)} dBm
                          </span>
                        ) : <span className="text-gray-400 text-xs">--</span>}
                      </td>
                      <td className="py-2.5 pr-4 text-xs">
                        {onu.distance !== null ? (
                          <span className="font-mono text-gray-700 dark:text-gray-300">{onu.distance} m</span>
                        ) : <span className="text-gray-400">--</span>}
                      </td>
                      <td
                        className="py-2.5 pr-4 text-xs cursor-pointer group"
                        title="Klik untuk assign/ganti ODP"
                        onClick={() => setAssigningOdpToOnu(onu)}
                      >
                        {onu.odpName
                          ? <span className="font-medium text-indigo-700 dark:text-indigo-400 group-hover:underline">{onu.odpName}</span>
                          : <span className="text-gray-400 group-hover:text-indigo-500 transition-colors">-- assign</span>}
                      </td>
                      <td className="py-2.5 pr-4">
                        {onu.customer ? (
                          <div>
                            <div className="text-xs font-medium text-gray-900 dark:text-white">{onu.customer.name}</div>
                            <div className="text-gray-500 dark:text-gray-400 text-xs">{onu.customer.username}</div>
                          </div>
                        ) : (
                          <span className="text-gray-400 text-xs">Unassigned</span>
                        )}
                      </td>
                      <td className="py-2.5 pr-4 text-xs text-gray-500 dark:text-gray-400">
                        {onu.lastSeenAt ? formatWIB(onu.lastSeenAt, 'dd MMM yyyy HH:mm') : '--'}
                      </td>
                      <td className="py-2.5 pr-3">
                        {onu.status === 'auth_failed' ? (
                          /* Unregistered ONU -- show Register button */
                          <div className="flex flex-wrap gap-1">
                            <button
                              onClick={() => setDetailOnu(onu)}
                              className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                            >
                              <Eye className="w-3 h-3" />
                              Detail
                            </button>
                            <button
                              onClick={() => setRegisteringOnu(onu)}
                              className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors"
                            >
                              <Plus className="w-3 h-3" />
                              Register
                            </button>
                          </div>
                        ) : confirmReboot === onu.id ? (
                          <div className="flex gap-1">
                            <button
                              onClick={() => handleRebootOnu(onu.id)}
                              disabled={rebootingOnu === onu.id}
                              className="px-2 py-1 text-xs bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
                            >
                              {rebootingOnu === onu.id ? 'Rebooting...' : 'Confirm'}
                            </button>
                            <button
                              onClick={() => setConfirmReboot(null)}
                              className="px-2 py-1 text-xs bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-300 dark:hover:bg-gray-600"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <div className="flex flex-wrap gap-1">
                            <button
                              onClick={() => setDetailOnu(onu)}
                              className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                            >
                              <Eye className="w-3 h-3" />
                              Detail
                            </button>
                            <button
                              onClick={() => setAssigningOnu(onu)}
                              disabled={deletingOnu === onu.id}
                              className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50 transition-colors"
                            >
                              <UserPlus className="w-3 h-3" />
                              Assign
                            </button>
                            <button
                              onClick={() => setEditingOnu(onu)}
                              disabled={deletingOnu === onu.id}
                              className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-teal-600 text-white rounded-md hover:bg-teal-700 disabled:opacity-50 transition-colors"
                            >
                              <Pencil className="w-3 h-3" />
                              Edit
                            </button>
                            <button
                              onClick={() => setConfirmReboot(onu.id)}
                              disabled={deletingOnu === onu.id || rebootingOnu !== null || batchRebooting}
                              className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-orange-500 text-white rounded-md hover:bg-orange-600 disabled:opacity-50 transition-colors"
                            >
                              <Power className="w-3 h-3" />
                              Reboot
                            </button>
                            <button
                              onClick={() => handleCleanConfigOnu(onu)}
                              disabled={cleaningConfigOnu === onu.id || deletingOnu === onu.id || rebootingOnu !== null || batchRebooting}
                              className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-yellow-600 text-white rounded-md hover:bg-yellow-700 disabled:opacity-50 transition-colors"
                              title="Reset konfigurasi service ONU di OLT (VLAN, profile). ONU tetap terdaftar."
                            >
                              {cleaningConfigOnu === onu.id ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Eraser className="w-3 h-3" />}
                              {cleaningConfigOnu === onu.id ? 'Cleaning...' : 'Clean'}
                            </button>
                            <button
                              onClick={() => handleDeleteOnu(onu)}
                              disabled={deletingOnu === onu.id || cleaningConfigOnu === onu.id || rebootingOnu !== null || batchRebooting}
                              className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50 transition-colors"
                              title="Deregister ONU dari PON port dan hapus dari database"
                            >
                              {deletingOnu === onu.id ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Trash2 className="w-3 h-3" />}
                              {deletingOnu === onu.id ? 'Deleting...' : 'Delete'}
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })}
                {filteredOnus.length === 0 && (
                  <tr>
                    <td colSpan={12} className="py-8 text-center text-gray-400">
                      No ONUs found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </TabsContent>

        {/* Alerts Tab */}
        <TabsContent value="alerts" className="space-y-3">
          {olt.alerts.filter((a: any) => !a.isResolved).length === 0 ? (
            <Card>
              <CardContent className="text-center py-8 text-gray-400">No active alerts</CardContent>
            </Card>
          ) : (
            olt.alerts.filter((a: any) => !a.isResolved).map((alert: any) => (
              <Card key={alert.id} className={alert.severity === 'critical' ? 'border-red-300' : ''}>
                <CardContent className="pt-4 pb-4">
                  <div className="flex items-start gap-3">
                    <AlertCircle className={`h-5 w-5 mt-0.5 ${alert.severity === 'critical' ? 'text-red-500' : 'text-orange-500'}`} />
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{alert.alertType.replace(/_/g, ' ')}</span>
                        <Badge variant={alert.severity === 'critical' ? 'destructive' : 'secondary'}>
                          {alert.severity}
                        </Badge>
                      </div>
                      <p className="text-gray-600 text-sm mt-1">{alert.message}</p>
                      <p className="text-gray-400 text-xs mt-1">
                        {formatWIB(alert.createdAt, 'dd MMM yyyy HH:mm')}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </TabsContent>

        {/* Settings Tab */}
        <TabsContent value="settings">
          <Card>
            <CardHeader>
              <CardTitle>Monitoring Settings</CardTitle>
              <CardDescription>Configure SNMP, SSH, Telnet and polling parameters</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* General */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <Label>Vendor</Label>
                  <Select
                    value={settings.vendor}
                    onValueChange={(v) => setSettings((s) => ({ ...s, vendor: v }))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="huawei">Huawei</SelectItem>
                      <SelectItem value="zte">ZTE</SelectItem>
                      <SelectItem value="fiberhome">FiberHome</SelectItem>
                      <SelectItem value="hioso">Hioso / C-Data</SelectItem>
                      <SelectItem value="bdcom">BDCOM</SelectItem>
                      <SelectItem value="raisecom">Raisecom</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label>Model</Label>
                  <Input
                    value={settings.model}
                    onChange={(e) => setSettings((s) => ({ ...s, model: e.target.value }))}
                    placeholder="e.g. C320"
                  />
                </div>
                <div>
                  <Label>Firmware Version</Label>
                  <Input
                    value={settings.firmwareVersion}
                    onChange={(e) => setSettings((s) => ({ ...s, firmwareVersion: e.target.value }))}
                    placeholder="e.g. V2.1.0 atau V2.2.0"
                  />
                </div>
                <div>
                  <Label>Polling Interval (seconds)</Label>
                  <Input
                    type="number"
                    value={settings.pollingInterval}
                    onChange={(e) => setSettings((s) => ({ ...s, pollingInterval: parseInt(e.target.value) || 300 }))}
                    min={60}
                    max={3600}
                  />
                </div>
              </div>

              {/* Router / NAS */}
              <div className="border rounded-lg p-4 space-y-3">
                <Label className="font-semibold">Router / NAS</Label>
                <p className="text-xs text-gray-500">Pilih router yang terhubung ke OLT ini. Digunakan untuk isolasi dan routing.</p>
                {routerList.length === 0 ? (
                  <p className="text-xs text-gray-400 italic">Tidak ada router. Tambahkan dulu di menu Router.</p>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                    {routerList.map((r) => (
                      <label key={r.id} className="flex items-center gap-2 p-2 border rounded cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800">
                        <input
                          type="checkbox"
                          checked={settings.routerIds.includes(r.id)}
                          onChange={(e) => {
                            setSettings((s) => ({
                              ...s,
                              routerIds: e.target.checked
                                ? [...s.routerIds, r.id]
                                : s.routerIds.filter((rid) => rid !== r.id),
                            }));
                          }}
                          className="w-4 h-4 text-blue-600 rounded"
                        />
                        <div>
                          <div className="text-sm font-medium">{r.name}</div>
                          <div className="text-xs text-gray-400 font-mono">{r.ipAddress}</div>
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex items-center gap-3">
                <Switch
                  checked={settings.monitoringEnabled}
                  onCheckedChange={(v) => setSettings((s) => ({ ...s, monitoringEnabled: v }))}
                />
                <Label>Enable Monitoring</Label>
              </div>

              {/* SNMP */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-3">
                  <Switch
                    checked={settings.snmpEnabled}
                    onCheckedChange={(v) => setSettings((s) => ({ ...s, snmpEnabled: v }))}
                  />
                  <Label className="font-semibold">SNMP</Label>
                </div>
                {settings.snmpEnabled && (
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label>Community String</Label>
                      <Input
                        value={settings.snmpCommunity}
                        onChange={(e) => setSettings((s) => ({ ...s, snmpCommunity: e.target.value }))}
                        placeholder="public"
                      />
                    </div>
                    <div>
                      <Label>Port</Label>
                      <Input
                        type="number"
                        value={settings.snmpPort}
                        onChange={(e) => setSettings((s) => ({ ...s, snmpPort: parseInt(e.target.value) || 161 }))}
                      />
                    </div>
                  </div>
                )}
                {settings.snmpEnabled && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTestConnection('snmp')}
                    disabled={testing === 'snmp'}
                  >
                    {testing === 'snmp' ? <RefreshCw className="h-4 w-4 animate-spin mr-1" /> : <TestTube className="h-4 w-4 mr-1" />}
                    Test SNMP
                  </Button>
                )}
              </div>

              {/* SSH */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-3">
                  <Switch
                    checked={settings.sshEnabled}
                    onCheckedChange={(v) => setSettings((s) => ({ ...s, sshEnabled: v }))}
                  />
                  <Label className="font-semibold">SSH</Label>
                </div>
                {settings.sshEnabled && (
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label>Port</Label>
                      <Input
                        type="number"
                        value={settings.sshPort}
                        onChange={(e) => setSettings((s) => ({ ...s, sshPort: parseInt(e.target.value) || 22 }))}
                      />
                    </div>
                    <div>
                      <Label>Username</Label>
                      <Input
                        value={settings.username}
                        onChange={(e) => setSettings((s) => ({ ...s, username: e.target.value }))}
                        placeholder="admin"
                      />
                    </div>
                    <div className="col-span-2">
                      <Label>Password</Label>
                      <Input
                        type="password"
                        value={settings.password}
                        onChange={(e) => setSettings((s) => ({ ...s, password: e.target.value }))}
                        placeholder="Leave blank to keep existing"
                      />
                    </div>
                  </div>
                )}
                {settings.sshEnabled && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTestConnection('ssh')}
                    disabled={testing === 'ssh'}
                  >
                    {testing === 'ssh' ? <RefreshCw className="h-4 w-4 animate-spin mr-1" /> : <TestTube className="h-4 w-4 mr-1" />}
                    Test SSH
                  </Button>
                )}
              </div>

              {/* Telnet */}
              <div className="border rounded-lg p-4 space-y-3">
                <div className="flex items-center gap-3">
                  <Switch
                    checked={settings.telnetEnabled}
                    onCheckedChange={(v) => setSettings((s) => ({ ...s, telnetEnabled: v }))}
                  />
                  <Label className="font-semibold">Telnet</Label>
                </div>
                {settings.telnetEnabled && (
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label>Port</Label>
                      <Input
                        type="number"
                        value={settings.telnetPort}
                        onChange={(e) => setSettings((s) => ({ ...s, telnetPort: parseInt(e.target.value) || 23 }))}
                      />
                    </div>
                    {!settings.sshEnabled && (
                      <>
                        <div>
                          <Label>Username</Label>
                          <Input
                            value={settings.username}
                            onChange={(e) => setSettings((s) => ({ ...s, username: e.target.value }))}
                            placeholder="admin"
                          />
                        </div>
                        <div className="col-span-2">
                          <Label>Password</Label>
                          <Input
                            type="password"
                            value={settings.password}
                            onChange={(e) => setSettings((s) => ({ ...s, password: e.target.value }))}
                            placeholder="Leave blank to keep existing"
                          />
                        </div>
                      </>
                    )}
                  </div>
                )}
                {settings.telnetEnabled && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTestConnection('telnet')}
                    disabled={testing === 'telnet'}
                  >
                    {testing === 'telnet' ? <RefreshCw className="h-4 w-4 animate-spin mr-1" /> : <TestTube className="h-4 w-4 mr-1" />}
                    Test Telnet
                  </Button>
                )}
              </div>

              <Button onClick={handleSaveSettings} disabled={saving}>
                {saving ? <RefreshCw className="h-4 w-4 animate-spin mr-2" /> : <Save className="h-4 w-4 mr-2" />}
                Save Settings
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Logs Tab */}
        <TabsContent value="logs">
          <Card>
            <CardContent className="pt-4">
              <div className="space-y-2">
                {(olt.monitoringLogs ?? []).map((log: any) => (
                  <div key={log.id} className="flex items-start gap-3 py-2 border-b last:border-0">
                    <div className={`text-xs font-mono px-1.5 py-0.5 rounded ${
                      log.severity === 'error' ? 'bg-red-100 text-red-700' :
                      log.severity === 'warning' ? 'bg-orange-100 text-orange-700' :
                      'bg-gray-100 text-gray-600'
                    }`}>
                      {log.logType}
                    </div>
                    <div className="flex-1">
                      <p className="text-sm">{log.message}</p>
                      <p className="text-xs text-gray-400">
                        {formatWIB(log.createdAt, 'dd MMM yyyy HH:mm')}
                      </p>
                    </div>
                  </div>
                ))}
                {(olt.monitoringLogs ?? []).length === 0 && (
                  <p className="text-center text-gray-400 py-8">No logs yet</p>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Port Map Tab -- Realistic ZTE C320 Chassis Diagram */}
        <TabsContent value="portmap">
          <ZTEChassisView olt={olt} onus={liveOnus ?? olt?.onuStatuses} onuLastRefresh={onuLastRefresh} />
        </TabsContent>
      </Tabs>

      {/* ONU Registration Modal */}
      {registeringOnu && (
        <ONURegisterModal
          oltId={id}
          onu={registeringOnu}
          vendor={olt?.vendor ?? null}
          onClose={() => setRegisteringOnu(null)}
          onSuccess={async () => { setRegisteringOnu(null); await handleSyncOLT({ silent: true }); }}
        />
      )}

      {detailOnu && (
        <ONUDetailModal
          oltId={id}
          onu={detailOnu}
          onClose={() => setDetailOnu(null)}
        />
      )}

      {assigningOnu && (
        <ONUAssignModal
          oltId={id}
          onu={assigningOnu}
          onClose={() => setAssigningOnu(null)}
          onSuccess={fetchOLT}
        />
      )}

      {editingOnu && (
        <ONUEditModal
          oltId={id}
          onu={editingOnu}
          onClose={() => setEditingOnu(null)}
          onSuccess={fetchOLT}
        />
      )}

      {assigningOdpToOnu && (
        <ONUOdpAssignModal
          oltId={id}
          onu={assigningOdpToOnu}
          onClose={() => setAssigningOdpToOnu(null)}
          onSuccess={() => { fetchLiveOnus(); setAssigningOdpToOnu(null); }}
        />
      )}
    </div>
  );
}

// --- ONUEditModal ------------------------------------------------------------

function ONUEditModal({
  oltId, onu, onClose, onSuccess,
}: { oltId: string; onu: ONU; onClose: () => void; onSuccess: () => Promise<void> }) {
  const [name, setName] = useState(onu.description ?? '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      const res = await fetch(`/api/olt/${oltId}/onus/${onu.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description: name }),
      });
      const j = await res.json();
      if (!res.ok || !j.success) throw new Error(j.error ?? 'Update failed');
      await onSuccess();
      onClose();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-md overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-gray-800">
          <h2 className="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <Pencil className="h-4 w-4 text-teal-500" /> Edit ONU
          </h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="p-5 space-y-4">
          <div className="text-xs text-gray-400 font-mono">
            {onu.serialNumber ?? 'No SN'} -- {onu.frame}/{onu.slot}/{onu.port}:{onu.onuId}
          </div>
          <div>
            <Label className="text-xs text-gray-500 mb-1">Name / Description</Label>
            <input
              autoFocus
              value={name}
              onChange={e => setName(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') save(); }}
              className="w-full border border-gray-300 dark:border-gray-600 rounded-md px-3 py-2 text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
              placeholder="e.g. Pelanggan Budi"
            />
          </div>
          {error && <p className="text-xs text-red-500">{error}</p>}
        </div>
        <div className="flex justify-end gap-2 px-5 py-4 border-t dark:border-gray-800">
          <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
          <Button onClick={save} disabled={saving} className="bg-teal-600 hover:bg-teal-700 text-white">
            {saving ? <RefreshCw className="h-3 w-3 mr-2 animate-spin" /> : <Save className="h-3 w-3 mr-1" />}
            Save
          </Button>
        </div>
      </div>
    </div>
  );
}

// --- PONPortModal -------------------------------------------------------------

function PONPortModal({
  oltId, slot, port, stat, loadingStat, acting, portStats, onAction, onRefresh, onClose,
}: {
  oltId: string;
  slot: number;
  port: number;
  stat: PONPortStat | null | undefined;
  loadingStat: boolean;
  acting: boolean;
  portStats: { total: number; online: number; offline: number; los: number; dyingGasp: number; unregistered: number } | undefined;
  onAction: (action: string) => Promise<void>;
  onRefresh: () => void;
  onClose: () => void;
}) {
  const iface = `gpon-olt_1/${slot}/${port}`;
  const isEnabled = stat?.adminStatus !== 'deactivate';
  const isUp = stat?.lineProto === 'up';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div
        className="bg-white dark:bg-slate-900 rounded-xl shadow-2xl border border-slate-200 dark:border-slate-700 w-full max-w-sm mx-4"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-2">
            <Signal className="w-4 h-4 text-green-400" />
            <span className="font-semibold text-slate-900 dark:text-white font-mono text-sm">{iface}</span>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="px-5 py-4 space-y-4">
          {/* Status badges */}
          <div className="flex items-center gap-2 flex-wrap">
            {loadingStat ? (
              <span className="text-xs text-slate-400 animate-pulse">Mengambil status...</span>
            ) : stat ? (
              <>
                <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full ${isEnabled ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-slate-200 text-slate-500 dark:bg-slate-700 dark:text-slate-400'}`}>
                  <Power className="w-3 h-3" />
                  {isEnabled ? 'Enabled' : 'Disabled'}
                </span>
                <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full ${isUp ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : 'bg-red-100 text-red-600 dark:bg-red-900/40 dark:text-red-400'}`}>
                  <Activity className="w-3 h-3" />
                  Link {isUp ? 'UP' : 'DOWN'}
                </span>
                {stat.description && (
                  <span className="text-xs text-slate-400 italic truncate max-w-[180px]">{stat.description}</span>
                )}
              </>
            ) : (
              <span className="text-xs text-slate-400">Status tidak tersedia</span>
            )}
          </div>

          {/* ONU stats */}
          {portStats && (
            <div className="grid grid-cols-3 gap-2">
              {[
                { label: 'Total ONU', value: portStats.total, color: 'text-slate-700 dark:text-slate-200' },
                { label: 'Online', value: portStats.online, color: 'text-green-600 dark:text-green-400' },
                { label: 'Offline', value: portStats.offline, color: 'text-red-500 dark:text-red-400' },
              ].map(item => (
                <div key={item.label} className="flex flex-col items-center p-2 rounded-lg bg-slate-50 dark:bg-slate-800">
                  <span className={`text-xl font-bold ${item.color}`}>{item.value}</span>
                  <span className="text-[10px] text-slate-400">{item.label}</span>
                </div>
              ))}
            </div>
          )}

          {/* Optical stats */}
          {stat && (stat.temperature !== undefined || stat.txPower !== undefined) && (
            <div className="flex gap-3 text-xs text-slate-500 dark:text-slate-400">
              {stat.temperature !== undefined && <span> {stat.temperature.toFixed(1)} degC</span>}
              {stat.txPower !== undefined && <span> TX {stat.txPower.toFixed(2)} dBm</span>}
            </div>
          )}

          {/* Warning when disabling */}
          {isEnabled && (
            <p className="text-[11px] text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 rounded px-3 py-2">
                Mematikan port PON akan memutus semua ONU di port ini.
            </p>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 px-5 pb-5">
          <button
            onClick={onRefresh}
            disabled={loadingStat || acting}
            className="flex items-center gap-1 px-3 py-1.5 text-xs rounded border border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50"
          >
            <RefreshCw className={`w-3 h-3 ${loadingStat ? 'animate-spin' : ''}`} />
            Refresh
          </button>

          <div className="flex-1" />

          {/* Enable / Disable */}
          {!stat || isEnabled ? (
            <button
              onClick={async () => { await onAction('disable'); onRefresh(); }}
              disabled={acting || loadingStat}
              className="flex items-center gap-1.5 px-4 py-1.5 text-xs font-semibold rounded bg-red-600 hover:bg-red-700 text-white disabled:opacity-50"
            >
              {acting ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Power className="w-3 h-3" />}
              Disable Port
            </button>
          ) : (
            <button
              onClick={async () => { await onAction('enable'); onRefresh(); }}
              disabled={acting || loadingStat}
              className="flex items-center gap-1.5 px-4 py-1.5 text-xs font-semibold rounded bg-green-600 hover:bg-green-700 text-white disabled:opacity-50"
            >
              {acting ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Power className="w-3 h-3" />}
              Enable Port
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// --- ONUOdpAssignModal --------------------------------------------------------

function ONUOdpAssignModal({
  oltId, onu, onClose, onSuccess,
}: { oltId: string; onu: ONU; onClose: () => void; onSuccess: () => void }) {
  const [odps, setOdps] = useState<{ id: string; name: string }[]>([]);
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<string | null>(onu.odpId ?? null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/network/odps')
      .then(r => r.json())
      .then((data: any[]) => setOdps(data.map((o: any) => ({ id: o.id, name: o.name }))))
      .catch(() => {});
  }, []);

  const filtered = odps.filter(o => o.name.toLowerCase().includes(search.toLowerCase()));

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      const body: Record<string, unknown> = selected ? { odpId: selected } : { clearOdp: true };
      const res = await fetch(`/api/olt/${oltId}/onus/${onu.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const d = await res.json();
        setError(d.error ?? 'Gagal menyimpan');
        return;
      }
      onSuccess();
    } catch {
      setError('Network error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-md border dark:border-gray-800">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-gray-800">
          <h2 className="font-semibold text-sm flex items-center gap-2">
            <Signal className="h-4 w-4 text-indigo-500" /> Assign ODP ke ONU
          </h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="p-5 space-y-4">
          <div className="text-xs text-gray-400 font-mono">
            {onu.serialNumber ?? 'No SN'} -- {onu.frame}/{onu.slot}/{onu.port}:{onu.onuId}
          </div>
          {onu.odpName && (
            <div className="text-xs text-indigo-600 dark:text-indigo-400">
              ODP saat ini: <strong>{onu.odpName}</strong>
            </div>
          )}
          <Input
            placeholder="Cari nama ODP..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="text-sm"
            autoFocus
          />
          <div className="max-h-56 overflow-y-auto rounded-lg border dark:border-gray-700 divide-y dark:divide-gray-800">
            {/* Clear option */}
            <button
              onClick={() => setSelected(null)}
              className={`w-full text-left px-3 py-2 text-xs flex items-center gap-2 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors ${
                selected === null ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 font-medium' : 'text-gray-500'
              }`}
            >
              <span className="w-2 h-2 rounded-full border-2 border-gray-400 flex-shrink-0" />
              -- Tidak ada ODP (hapus link)
            </button>
            {filtered.length === 0 && (
              <div className="px-3 py-4 text-xs text-gray-400 text-center">Tidak ada ODP ditemukan</div>
            )}
            {filtered.map(odp => (
              <button
                key={odp.id}
                onClick={() => setSelected(odp.id)}
                className={`w-full text-left px-3 py-2 text-xs flex items-center gap-2 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors ${
                  selected === odp.id ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 font-medium' : 'text-gray-900 dark:text-gray-200'
                }`}
              >
                <span className={`w-2 h-2 rounded-full flex-shrink-0 ${selected === odp.id ? 'bg-indigo-500' : 'bg-gray-300 dark:bg-gray-600'}`} />
                {odp.name}
              </button>
            ))}
          </div>
          {error && <p className="text-xs text-red-500">{error}</p>}
        </div>
        <div className="flex justify-end gap-2 px-5 py-4 border-t dark:border-gray-800">
          <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
          <Button onClick={save} disabled={saving} className="bg-indigo-600 hover:bg-indigo-700 text-white">
            {saving ? <RefreshCw className="h-3 w-3 mr-2 animate-spin" /> : <Save className="h-3 w-3 mr-1" />}
            Simpan
          </Button>
        </div>
      </div>
    </div>
  );
}
