const radixPlugin = require('tailwindcss-radix');

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      colors: {
        // Custom color system from theme.css
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        card: {
          DEFAULT: 'var(--card)',
          foreground: 'var(--card-foreground)',
        },
        popover: {
          DEFAULT: 'var(--popover)',
          foreground: 'var(--popover-foreground)',
        },
        primary: {
          DEFAULT: 'var(--primary)',
          foreground: 'var(--primary-foreground)',
        },
        secondary: {
          DEFAULT: 'var(--secondary)',
          foreground: 'var(--secondary-foreground)',
        },
        muted: {
          DEFAULT: 'var(--muted)',
          foreground: 'var(--muted-foreground)',
        },
        accent: {
          DEFAULT: 'var(--accent)',
          foreground: 'var(--accent-foreground)',
        },
        destructive: {
          DEFAULT: 'var(--destructive)',
          foreground: 'var(--destructive-foreground)',
        },
        border: 'var(--border)',
        input: 'var(--input)',
        ring: 'var(--ring)',
        // Orange color scale
        orange: {
          1: 'var(--orange-1)',
          2: 'var(--orange-2)',
          3: 'var(--orange-3)',
          4: 'var(--orange-4)',
          5: 'var(--orange-5)',
          6: 'var(--orange-6)',
          7: 'var(--orange-7)',
          8: 'var(--orange-8)',
          9: 'var(--orange-9)',
          10: 'var(--orange-10)',
          11: 'var(--orange-11)',
          12: 'var(--orange-12)',
        },
        // Gray color scale
        gray: {
          1: 'var(--gray-1)',
          2: 'var(--gray-2)',
          3: 'var(--gray-3)',
          4: 'var(--gray-4)',
          5: 'var(--gray-5)',
          6: 'var(--gray-6)',
          7: 'var(--gray-7)',
          8: 'var(--gray-8)',
          9: 'var(--gray-9)',
          10: 'var(--gray-10)',
          11: 'var(--gray-11)',
          12: 'var(--gray-12)',
        },
        // Lime color scale (dark theme)
        lime: {
          1: 'var(--lime-1)',
          2: 'var(--lime-2)',
          3: 'var(--lime-3)',
          4: 'var(--lime-4)',
          5: 'var(--lime-5)',
          6: 'var(--lime-6)',
          7: 'var(--lime-7)',
          8: 'var(--lime-8)',
          9: 'var(--lime-9)',
          10: 'var(--lime-10)',
          11: 'var(--lime-11)',
          12: 'var(--lime-12)',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      animation: {
        'fade-in': 'fadeIn 0.5s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
    },
  },
  plugins: [radixPlugin],
}
