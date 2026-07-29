/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        blush: {
          50: '#fff1f6',
          100: '#ffe3ee',
          200: '#ffc7de',
          300: '#ff9bc4',
          400: '#ff5fa2',
          500: '#f92e83',
          600: '#e30f66',
          700: '#bd0553',
          800: '#9c0848',
          900: '#820c40',
        },
      },
      fontFamily: {
        sans: ['"PingFang TC"', '"Noto Sans TC"', 'system-ui', 'sans-serif'],
      },
      keyframes: {
        'pop-in': {
          '0%': { transform: 'scale(0.82) translateY(24px)', opacity: '0' },
          '60%': { transform: 'scale(1.03) translateY(0)', opacity: '1' },
          '100%': { transform: 'scale(1) translateY(0)', opacity: '1' },
        },
        'float-up': {
          '0%': { transform: 'translateY(0)', opacity: '1' },
          '100%': { transform: 'translateY(-48px)', opacity: '0' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
      },
      animation: {
        'pop-in': 'pop-in 0.42s cubic-bezier(0.22, 1, 0.36, 1)',
        'float-up': 'float-up 1.2s ease-out forwards',
        shimmer: 'shimmer 2.5s linear infinite',
      },
    },
  },
  plugins: [],
}
