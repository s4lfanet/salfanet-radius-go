'use client';
import { useState, useRef, useEffect, useCallback } from 'react';
import { Camera, X, Check, RefreshCw } from 'lucide-react';

interface CameraCaptureProps {
  onCapture: (file: File) => void;
  onClose: () => void;
  label?: string;
}

export default function CameraCapture({ onCapture, onClose, label = 'Ambil Foto' }: CameraCaptureProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const [error, setError] = useState('');
  const [captured, setCaptured] = useState<string | null>(null);
  const [facingMode, setFacingMode] = useState<'environment' | 'user'>('environment');

  const stopStream = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(t => t.stop());
      streamRef.current = null;
    }
  }, []);

  const startCamera = useCallback(async (mode: 'environment' | 'user') => {
    stopStream();
    setError('');
    setCaptured(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: mode }, width: { ideal: 1920 }, height: { ideal: 1080 } },
        audio: false,
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
    } catch (err: any) {
      if (err.name === 'NotAllowedError') {
        setError('Akses kamera ditolak. Izinkan akses kamera di browser.');
      } else if (err.name === 'NotFoundError') {
        setError('Kamera tidak ditemukan di perangkat ini.');
      } else {
        setError('Gagal mengakses kamera: ' + err.message);
      }
    }
  }, [stopStream]);

  useEffect(() => {
    startCamera(facingMode);
    return () => stopStream();
  }, [facingMode, startCamera, stopStream]);

  const handleCapture = () => {
    if (!videoRef.current || !canvasRef.current) return;
    const video = videoRef.current;
    const canvas = canvasRef.current;
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.drawImage(video, 0, 0);
    setCaptured(canvas.toDataURL('image/jpeg', 0.85));
    stopStream();
  };

  const handleRetake = () => {
    setCaptured(null);
    startCamera(facingMode);
  };

  const handleConfirm = () => {
    if (!captured || !canvasRef.current) return;
    canvasRef.current.toBlob((blob) => {
      if (!blob) return;
      const file = new File([blob], `camera-${Date.now()}.jpg`, { type: 'image/jpeg' });
      onCapture(file);
    }, 'image/jpeg', 0.85);
  };

  const handleClose = () => {
    stopStream();
    onClose();
  };

  const switchCamera = () => {
    setFacingMode(prev => prev === 'environment' ? 'user' : 'environment');
  };

  return (
    <div className="fixed inset-0 z-[9999] bg-black flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-black/80">
        <span className="text-white text-sm font-medium">{label}</span>
        <button onClick={handleClose} className="text-white/80 hover:text-white p-1">
          <X className="w-5 h-5" />
        </button>
      </div>

      {/* Camera / Preview */}
      <div className="flex-1 relative flex items-center justify-center overflow-hidden">
        {error ? (
          <div className="text-center px-6">
            <p className="text-red-400 text-sm">{error}</p>
            <button onClick={() => startCamera(facingMode)} className="mt-4 px-4 py-2 bg-white/10 text-white rounded-lg text-xs">
              Coba Lagi
            </button>
          </div>
        ) : captured ? (
          <>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={captured} alt="Captured" className="max-w-full max-h-full object-contain" />
          </>
        ) : (
          <video ref={videoRef} autoPlay playsInline muted className="max-w-full max-h-full object-contain" />
        )}
        <canvas ref={canvasRef} className="hidden" />
      </div>

      {/* Controls */}
      <div className="bg-black/80 px-4 py-4 flex items-center justify-center gap-6">
        {!captured ? (
          <>
            <button onClick={switchCamera} className="text-white/80 hover:text-white p-2" title="Ganti Kamera">
              <RefreshCw className="w-5 h-5" />
            </button>
            <button
              onClick={handleCapture}
              disabled={!!error}
              className="w-16 h-16 rounded-full border-4 border-white flex items-center justify-center disabled:opacity-30 active:scale-95 transition-transform"
            >
              <span className="w-12 h-12 rounded-full bg-white" />
            </button>
            <div className="w-9" />
          </>
        ) : (
          <>
            <button onClick={handleRetake} className="flex flex-col items-center gap-1 text-white/80 hover:text-white">
              <RefreshCw className="w-6 h-6" />
              <span className="text-[10px]">Ulangi</span>
            </button>
            <button onClick={handleConfirm} className="w-16 h-16 rounded-full bg-green-500 flex items-center justify-center active:scale-95 transition-transform">
              <Check className="w-7 h-7 text-white" />
            </button>
            <div className="w-9" />
          </>
        )}
      </div>
    </div>
  );
}
