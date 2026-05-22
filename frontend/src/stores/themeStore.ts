import { create } from 'zustand';
import type { ThemeMode } from '../lib/theme';

const STORAGE_KEY = 'mansio-theme';

function isMode(v: unknown): v is ThemeMode {
  return v === 'system' || v === 'light' || v === 'dark';
}

function loadMode(): ThemeMode {
  if (typeof localStorage === 'undefined') return 'system';
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (isMode(v)) return v;
  } catch {
    // localStorage may be disabled (private mode, no permission)
  }
  return 'system';
}

function saveMode(mode: ThemeMode) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, mode);
  } catch {
    // ignore quota / disabled storage
  }
}

interface ThemeState {
  mode: ThemeMode;
  setMode: (m: ThemeMode) => void;
}

export const useThemeStore = create<ThemeState>((set) => ({
  mode: loadMode(),
  setMode: (mode) => {
    saveMode(mode);
    set({ mode });
  },
}));
