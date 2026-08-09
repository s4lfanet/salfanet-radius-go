"use client";

import { useEffect, useState } from "react";
import { Plus, Loader2, Pencil, Trash2, Code2, Cpu, RefreshCw, CheckCircle2, AlertCircle } from "lucide-react";
import { SimpleModal, ModalHeader, ModalTitle, ModalDescription, ModalBody, ModalFooter, ModalButton, ModalInput, ModalLabel, ModalTextarea } from '@/components/cyberpunk/SimpleModal';
import { useToast } from '@/components/cyberpunk/CyberToast';
import { useTranslation } from "@/hooks/useTranslation";

interface GenieVp {
  _id: string;
  script: string;
}

export default function VirtualParametersPage() {
  const { t } = useTranslation();
  const { addToast, confirm } = useToast();
  const [items, setItems] = useState<GenieVp[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<GenieVp | null>(null);
  const [form, setForm] = useState({ name: "", script: "" });

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/settings/genieacs/virtual-parameters", { cache: "no-store" });
      const data = await res.json();
      if (data.success) {
        setItems(data.data || []);
      }
    } catch (error) {
      console.error("Failed to load virtual parameters", error);
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setForm({ name: "", script: "" });
    setEditing(null);
  };

  const openCreate = () => {
    resetForm();
    setShowForm(true);
  };

  const openEdit = (vp: GenieVp) => {
    setEditing(vp);
    setForm({ name: vp._id, script: vp.script });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const name = form.name.trim();
    if (!name) {
      addToast({ type: 'error', title: t('common.error'), description: 'Name is required' });
      return;
    }
    setSaving(true);
    try {
      const payload = { name, script: form.script };

      const endpoint = editing
        ? `/api/settings/genieacs/virtual-parameters/${encodeURIComponent(editing._id)}`
        : "/api/settings/genieacs/virtual-parameters";
      const method = editing ? "PUT" : "POST";

      const res = await fetch(endpoint, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await res.json();

      if (!res.ok || !data.success) {
        throw new Error(data.error || data.details || t('genieacs.failedSaveParam'));
      }

      setShowForm(false);
      resetForm();
      fetchData();
      addToast({ type: 'success', title: t('common.success'), description: t('genieacs.paramSaved') });
    } catch (error: any) {
      addToast({ type: 'error', title: t('common.error'), description: error?.message || t('genieacs.failedSaveParam') });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (vp: GenieVp) => {
    if (!await confirm({
      title: t('genieacs.deleteParamConfirm'),
      message: t('genieacs.deleteParamWarning').replace('{name}', vp._id),
      confirmText: t('genieacs.yesDeleteIt'),
      cancelText: t('common.cancel'),
      variant: 'danger',
    })) return;

    setDeletingId(vp._id);
    try {
      const res = await fetch(`/api/settings/genieacs/virtual-parameters/${encodeURIComponent(vp._id)}`, { method: "DELETE" });
      const data = await res.json();
      if (!res.ok || !data.success) throw new Error(data.error || t('genieacs.failedDeleteParam'));
      setItems((prev) => prev.filter((item) => item._id !== vp._id));
      addToast({ type: 'success', title: t('common.success'), description: 'Virtual parameter deleted from GenieACS' });
    } catch (error: any) {
      addToast({ type: 'error', title: t('common.error'), description: error?.message || t('genieacs.failedDeleteParam') });
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="bg-background relative">
      <div className="relative z-10 max-w-6xl mx-auto space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-foreground dark:text-transparent dark:bg-clip-text dark:bg-gradient-to-r dark:from-brand-400 dark:via-white dark:to-accent-foreground dark:drop-shadow-[0_0_30px_rgba(70, 95, 255,0.5)] flex items-center gap-2">
              <Cpu className="w-6 h-6 text-brand-500 dark:text-brand-400 dark:drop-shadow-[0_0_20px_rgba(70, 95, 255,0.6)]" />
              {t('genieacs.virtualParamsTitle')}
            </h1>
            <p className="text-xs sm:text-sm text-muted-foreground mt-1">
              {t('genieacs.virtualParamsSubtitle')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={fetchData}
              className="inline-flex items-center gap-1 px-3 py-2 rounded-lg border border-border text-foreground hover:bg-muted/50 text-sm"
              title={t('common.refresh')}
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={openCreate}
              className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-primary text-white text-sm font-medium hover:bg-primary/90 shadow"
            >
              <Plus className="w-4 h-4" />
              {t('genieacs.addVirtualParam')}
            </button>
          </div>
        </div>

        {/* Sync Status Banner */}
        <div className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-3 flex items-center gap-3">
          <CheckCircle2 className="w-5 h-5 text-emerald-500 flex-shrink-0" />
          <div className="flex-1">
            <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
              Synced with GenieACS Server
            </p>
            <p className="text-xs text-emerald-600 dark:text-emerald-400">
              {items.length} virtual parameters loaded from GenieACS. Changes here are synced directly to the ACS server.
            </p>
          </div>
        </div>

        {/* Help Banner */}
        <div className="bg-primary/10 border border-primary/30 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <Code2 className="w-5 h-5 text-primary dark:text-primary" />
            </div>
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-100 mb-1">
                {t('genieacs.whatIsVirtualParams')}
              </h3>
              <p className="text-xs text-blue-800 dark:text-blue-200 mb-2">
                {t('genieacs.virtualParamsDesc')}
              </p>
              <details className="text-xs text-blue-700 dark:text-blue-300">
                <summary className="cursor-pointer font-medium hover:text-blue-900 dark:hover:text-blue-100">
                  {t('genieacs.viewExamples')}
                </summary>
                <div className="mt-2 space-y-2 pl-4 border-l-2 border-blue-300 dark:border-blue-700">
                  <div>
                    <p className="font-mono text-[11px] text-blue-900 dark:text-blue-100">VirtualParameters.uptime</p>
                    <p className="text-[11px] text-primary dark:text-primary">{`\u2192`} {t('genieacs.calcUptimeDesc')}</p>
                  </div>
                  <div>
                    <p className="font-mono text-[11px] text-blue-900 dark:text-blue-100">VirtualParameters.redaman</p>
                    <p className="text-[11px] text-primary dark:text-primary">{`\u2192`} {t('genieacs.getSignalDesc')}</p>
                  </div>
                  <div>
                    <p className="font-mono text-[11px] text-blue-900 dark:text-blue-100">VirtualParameters.pppUsername</p>
                    <p className="text-[11px] text-primary dark:text-primary">{`\u2192`} {t('genieacs.getPppoeDesc')}</p>
                  </div>
                </div>
              </details>
            </div>
          </div>
        </div>

        <div className="bg-card rounded-lg border border-border">
          <div className="px-4 py-3 border-b border-border flex items-center justify-between">
            <div>
              <p className="text-sm font-semibold text-foreground dark:text-gray-100">{t('genieacs.parameterList')}</p>
              <p className="text-xs text-muted-foreground">
                {items.length} {t('genieacs.parameters')}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <a
                href="https://docs.genieacs.com/"
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-muted-foreground dark:text-muted-foreground hover:text-primary inline-flex items-center gap-1"
                title={t('common.docs')}
              >
                <Code2 className="w-3 h-3" />
                {t('common.docs')}
              </a>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-10">
              <Loader2 className="w-5 h-5 animate-spin text-brand-500 dark:text-brand-400 dark:drop-shadow-[0_0_20px_rgba(70, 95, 255,0.6)]" />
            </div>
          ) : items.length === 0 ? (
            <div className="py-10 text-center">
              <Code2 className="w-12 h-12 mx-auto text-muted-foreground mb-3" />
              <p className="text-sm text-muted-foreground mb-2">{t('genieacs.noVirtualParams')}</p>
              <p className="text-xs text-muted-foreground mb-4">{t('genieacs.createFirstParam')}</p>
              <button
                onClick={openCreate}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90"
              >
                <Plus className="w-4 h-4" />
                {t('genieacs.addFirstParam')}
              </button>
            </div>
          ) : (
            <>
              {/* Mobile Card View */}
              <div className="block md:hidden space-y-3 p-3">
                {items.map((vp) => (
                  <div key={vp._id} className="bg-card/80 backdrop-blur-xl rounded-xl border border-brand-600/20 p-3">
                    <div className="flex items-start justify-between mb-2">
                      <span className="text-sm font-semibold text-foreground font-mono">{vp._id}</span>
                    </div>
                    <div className="space-y-1 mb-2">
                      <p className="text-xs text-muted-foreground line-clamp-3 font-mono bg-muted/50 dark:bg-input/50 rounded p-2">
                        {vp.script}
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2 pt-2 border-t border-border">
                      <button
                        onClick={() => openEdit(vp)}
                        className="inline-flex items-center gap-1 p-2 rounded border border-border text-primary hover:bg-primary/10 text-xs"
                      >
                        <Pencil className="w-4 h-4" />
                        {t('common.edit')}
                      </button>
                      <button
                        onClick={() => handleDelete(vp)}
                        disabled={deletingId === vp._id}
                        className="inline-flex items-center gap-1 p-2 rounded border border-destructive/30 text-destructive hover:bg-destructive/10 disabled:opacity-50 text-xs"
                      >
                        {deletingId === vp._id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                        {t('common.delete')}
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              {/* Desktop List View */}
              <div className="divide-y divide-gray-200 dark:divide-gray-800 hidden md:block">
                {items.map((vp) => (
                  <div key={vp._id} className="px-4 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div className="space-y-1 flex-1 min-w-0">
                      <span className="text-sm font-semibold text-foreground font-mono">{vp._id}</span>
                      <p className="text-xs text-muted-foreground dark:text-muted-foreground line-clamp-2 font-mono bg-muted/30 dark:bg-input/30 rounded px-2 py-1">
                        {vp.script}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 text-xs flex-shrink-0">
                      <button
                        onClick={() => openEdit(vp)}
                        className="inline-flex items-center gap-1 px-2 py-1 rounded border border-border text-primary hover:bg-primary/10 dark:text-blue-300 dark:hover:bg-blue-900/30"
                      >
                        <Pencil className="w-4 h-4" />
                        {t('common.edit')}
                      </button>
                      <button
                        onClick={() => handleDelete(vp)}
                        disabled={deletingId === vp._id}
                        className="inline-flex items-center gap-1 px-2 py-1 rounded border border-destructive/30 text-destructive hover:bg-destructive/10 disabled:opacity-50"
                      >
                        {deletingId === vp._id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                        {t('common.delete')}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        <SimpleModal isOpen={showForm} onClose={() => setShowForm(false)} size="lg">
          <ModalHeader>
            <ModalTitle className="flex items-center gap-2">
              <Code2 className="w-4 h-4 text-brand-400" />
              {editing ? `Edit: ${editing._id}` : t('genieacs.addVirtualParamTitle')}
            </ModalTitle>
            <ModalDescription>{t('genieacs.fillPathExpression')}</ModalDescription>
          </ModalHeader>

          <form onSubmit={handleSubmit}>
            <ModalBody className="space-y-3">
              <div>
                <ModalLabel required>{t('genieacs.paramNameLabel')}</ModalLabel>
                <ModalInput
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder={t('genieacs.paramNamePlaceholder')}
                  className="font-mono"
                  required
                  disabled={!!editing}
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  {t('genieacs.paramNameHint')} — This is the VirtualParameter name in GenieACS (e.g. <code className="font-mono">pppoeIP</code>, <code className="font-mono">redaman</code>)
                </p>
              </div>

              <div>
                <ModalLabel required>Script</ModalLabel>
                <ModalTextarea
                  value={form.script}
                  onChange={(e) => setForm({ ...form, script: e.target.value })}
                  className="text-xs font-mono resize-y"
                  rows={12}
                  placeholder="// GenieACS virtual parameter script&#10;let result = '';&#10;return {writable: false, value: [result, &quot;xsd:string&quot;]};"
                  required
                />
                <div className="mt-1 space-y-1">
                  <p className="text-[11px] text-muted-foreground">{t('genieacs.expressionHint')}</p>
                  <details className="text-[11px] text-muted-foreground">
                    <summary className="cursor-pointer text-brand-400 hover:underline">{t('genieacs.viewExpressionExamples')}</summary>
                    <div className="mt-2 p-2 bg-muted/50 dark:bg-input/50 rounded border border-brand-600/20 space-y-2">
                      <div>
                        <p className="font-semibold text-foreground">1. PPPoE IP:</p>
                        <pre className="text-[10px] overflow-x-auto text-brand-400/80">{`let result = '';
let keys = ['InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress'];
result = getParameterValue(keys);
return {writable: false, value: [result, "xsd:string"]};`}</pre>
                      </div>
                      <div>
                        <p className="font-semibold text-foreground">2. RxPower (redaman):</p>
                        <pre className="text-[10px] overflow-x-auto text-brand-400/80">{`let huawei = declare("InternetGatewayDevice.WANDevice.*.X_GponInterafceConfig.RXPower", {value: Date.now()});
let m = "N/A";
if (huawei.size) { m = huawei.value[0]; }
return {writable: false, value: [m, "xsd:string"]};`}</pre>
                      </div>
                    </div>
                  </details>
                </div>
              </div>

              {/* Info Section */}
              <div className="bg-[#ff8c00]/10 border border-[#ff8c00]/30 rounded-lg p-3">
                <p className="text-xs font-semibold text-[#ff8c00] mb-1"><AlertCircle className="w-3.5 h-3.5 inline mr-1" />Note</p>
                <p className="text-[11px] text-[#ff8c00]/80">
                  This virtual parameter will be created/updated directly in the GenieACS server. Changes are immediately active for all devices on next inform.
                </p>
              </div>
            </ModalBody>
            <ModalFooter>
              <ModalButton variant="secondary" type="button" onClick={() => setShowForm(false)}>
                {t('common.cancel')}
              </ModalButton>
              <ModalButton variant="primary" type="submit" disabled={saving} className="flex items-center gap-2">
                {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                {editing ? t('genieacs.saveChanges') : t('common.save')}
              </ModalButton>
            </ModalFooter>
          </form>
        </SimpleModal>
      </div>
    </div>
  );
}


