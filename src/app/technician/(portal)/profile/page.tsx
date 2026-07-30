'use client';

export const dynamic = 'force-dynamic';

import { useState, useEffect } from 'react';
import { User, Mail, Phone, Lock, Loader2, Save, Key } from 'lucide-react';
import { useToast } from '@/components/cyberpunk/CyberToast';
import { useTranslation } from '@/hooks/useTranslation';

interface Profile {
  id: string;
  username: string;
  name: string;
  email: string | null;
  phone: string | null;
  createdAt: string;
}

export default function TechnicianProfilePage() {
  const { addToast } = useToast();
  const { t } = useTranslation();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [changingPw, setChangingPw] = useState(false);
  const [form, setForm] = useState({ name: '', email: '', phone: '' });
  const [pwForm, setPwForm] = useState({ currentPassword: '', newPassword: '', confirmPassword: '' });
  const [showPwForm, setShowPwForm] = useState(false);

  useEffect(() => {
    fetch('/api/technician/profile')
      .then((r) => r.json())
      .then((data) => {
        if (data.profile) {
          setProfile(data.profile);
          setForm({ name: data.profile.name || '', email: data.profile.email || '', phone: data.profile.phone || '' });
        }
      })
      .catch(() => addToast({ type: 'error', title: t('techPortal.failedLoadProfile') }))
      .finally(() => setLoading(false));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSaveProfile = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/technician/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      const data = await res.json();
      if (res.ok) {
        addToast({ type: 'success', title: t('techPortal.profileUpdated') });
      } else {
        addToast({ type: 'error', title: data.error || t('techPortal.profileError') });
      }
    } catch {
      addToast({ type: 'error', title: t('techPortal.profileError') });
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    if (pwForm.newPassword !== pwForm.confirmPassword) {
      addToast({ type: 'error', title: t('techPortal.passwordMismatch') });
      return;
    }
    if (pwForm.newPassword.length < 6) {
      addToast({ type: 'error', title: t('techPortal.passwordMinLength') });
      return;
    }
    setChangingPw(true);
    try {
      const res = await fetch('/api/technician/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ currentPassword: pwForm.currentPassword, newPassword: pwForm.newPassword }),
      });
      const data = await res.json();
      if (res.ok) {
        addToast({ type: 'success', title: t('techPortal.passwordChanged') });
        setPwForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
        setShowPwForm(false);
      } else {
        addToast({ type: 'error', title: data.error || t('techPortal.passwordError') });
      }
    } catch {
      addToast({ type: 'error', title: t('techPortal.passwordError') });
    } finally {
      setChangingPw(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-brand-400" />
      </div>
    );
  }

  return (
    <div className="p-4 lg:p-6 w-full space-y-6">
      <h1 className="text-lg font-bold text-slate-900 dark:text-white">{t('techPortal.profile')}</h1>

      {/* Profile Card */}
      <div className="bg-white dark:bg-secondary/80 border border-slate-200 dark:border-brand-600/20 rounded-xl p-5 space-y-4">
        <div className="flex items-center gap-4 pb-4 border-b border-slate-200 dark:border-brand-600/20">
          <div className="w-14 h-14 bg-gradient-to-br from-brand-600 to-brand-400 rounded-xl flex items-center justify-center shadow-[0_0_20px_rgba(122, 90, 248,0.4)]">
            <User className="w-7 h-7 text-white" />
          </div>
          <div>
            <p className="text-base font-bold text-slate-900 dark:text-white">{profile?.name}</p>
            <p className="text-xs text-slate-500 dark:text-muted-foreground/60">@{profile?.username}</p>
          </div>
        </div>

        {/* Name */}
        <div>
          <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">{t('techPortal.name')}</label>
          <div className="relative">
            <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" />
          </div>
        </div>

        {/* Email */}
        <div>
          <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">Email</label>
          <div className="relative">
            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" />
          </div>
        </div>

        {/* Phone */}
        <div>
          <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">{t('techPortal.phone')}</label>
          <div className="relative">
            <Phone className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input type="tel" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" />
          </div>
        </div>

        <button onClick={handleSaveProfile} disabled={saving} className="w-full flex items-center justify-center gap-2 py-2.5 bg-gradient-to-r from-brand-400 to-[#00d4e6] hover:shadow-[0_0_20px_rgba(70, 95, 255,0.5)] disabled:opacity-50 text-black text-sm font-bold rounded-xl transition-all">
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          {t('techPortal.save')}
        </button>
      </div>

      {/* Change Password */}
      <div className="bg-white dark:bg-secondary/80 border border-slate-200 dark:border-brand-600/20 rounded-xl p-5 space-y-4">
        <button onClick={() => setShowPwForm(!showPwForm)} className="flex items-center gap-2 text-sm font-bold text-slate-900 dark:text-white">
          <Key className="w-4 h-4 text-brand-600" />
          {t('techPortal.changePassword')}
        </button>

        {showPwForm && (
          <div className="space-y-3 pt-2">
            <div>
              <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">{t('techPortal.currentPassword')}</label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input type="password" value={pwForm.currentPassword} onChange={(e) => setPwForm({ ...pwForm, currentPassword: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" autoComplete="current-password" />
              </div>
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">{t('techPortal.newPassword')}</label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input type="password" value={pwForm.newPassword} onChange={(e) => setPwForm({ ...pwForm, newPassword: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" autoComplete="new-password" />
              </div>
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600 dark:text-muted-foreground/70 mb-1 block">{t('techPortal.confirmPassword')}</label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input type="password" value={pwForm.confirmPassword} onChange={(e) => setPwForm({ ...pwForm, confirmPassword: e.target.value })} className="w-full pl-10 pr-3 py-2.5 bg-slate-50 dark:bg-input border border-slate-200 dark:border-brand-600/30 rounded-xl text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-[brand-400]/30 transition" autoComplete="new-password" />
              </div>
            </div>
            <div className="flex gap-2">
              <button onClick={() => { setShowPwForm(false); setPwForm({ currentPassword: '', newPassword: '', confirmPassword: '' }); }} className="flex-1 py-2.5 border border-slate-200 dark:border-brand-600/30 text-slate-700 dark:text-muted-foreground text-sm font-bold rounded-xl hover:bg-slate-50 dark:hover:bg-brand-600/10 transition">
                {t('techPortal.cancel')}
              </button>
              <button onClick={handleChangePassword} disabled={changingPw} className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-gradient-to-r from-brand-600 to-accent-foreground hover:shadow-[0_0_20px_rgba(122, 90, 248,0.5)] disabled:opacity-50 text-white text-sm font-bold rounded-xl transition-all">
                {changingPw ? <Loader2 className="w-4 h-4 animate-spin" /> : <Lock className="w-4 h-4" />}
                {t('techPortal.save')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
