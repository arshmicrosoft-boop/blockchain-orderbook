import { useEffect, useState } from 'react';

/**
 * Custom hook for theme management
 * Handles dark/light mode with localStorage persistence
 */
export function useTheme() {
  const [isDark, setIsDark] = useState(true);

  // Apply theme to DOM
  useEffect(() => {
    const root = document.documentElement;
    if (isDark) {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    try {
      localStorage.setItem('theme', isDark ? 'dark' : 'light');
    } catch {
      // Ignore localStorage errors
    }
  }, [isDark]);

  // Load theme from localStorage on mount
  useEffect(() => {
    try {
      const saved = localStorage.getItem('theme');
      if (saved) {
        setIsDark(saved === 'dark');
      }
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  const toggleTheme = () => setIsDark((prev) => !prev);

  return { isDark, toggleTheme };
}