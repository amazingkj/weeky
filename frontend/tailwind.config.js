/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Pretendard Variable"', '"Pretendard"', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
        serif: ['"Noto Serif KR"', '"Nanum Myeongjo"', 'serif'],
      },
      colors: {
        // 먹빛(ink) 기운이 도는 차가운 회색 — 서류지 위의 먹 느낌
        neutral: {
          50: '#F7F8FA',
          100: '#F0F2F5',
          200: '#E2E5EB',
          300: '#CFD4DD',
          400: '#99A0AE',
          500: '#6C7383',
          600: '#525968',
          700: '#3D4351',
          800: '#2A2F3B',
          850: '#222634',
          900: '#1B1F2A',
          925: '#141822',
          950: '#0F121A',
        },
        // 먹남색 — 주 액센트 (활성 탭, 주요 버튼, 포커스)
        ink: {
          50: '#EEF2FA',
          100: '#DEE6F4',
          200: '#BFCDE8',
          300: '#96ACD6',
          400: '#6684BD',
          500: '#4463A4',
          600: '#334F8E',
          700: '#2A4075',
          800: '#23345D',
          900: '#1D2A49',
        },
        // 인주색 — 제출완료 도장 전용
        seal: {
          50: '#FCF0ED',
          100: '#F8DDD7',
          500: '#C74634',
          600: '#B23A2A',
          700: '#973023',
        },
      },
      animation: {
        'fadeIn': 'fadeIn 0.2s ease-out',
        'slideIn': 'slideIn 0.2s ease-out',
        'slideOut': 'slideOut 0.15s ease-in forwards',
        'slideUp': 'slideUp 0.2s ease-out',
        'stamp': 'stamp 0.3s cubic-bezier(0.16, 1.2, 0.4, 1)',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideIn: {
          '0%': { opacity: '0', transform: 'translateY(-8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideOut: {
          '0%': { opacity: '1', transform: 'translateY(0)' },
          '100%': { opacity: '0', transform: 'translateY(-8px)' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(12px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        stamp: {
          '0%': { opacity: '0', transform: 'scale(1.6) rotate(-8deg)' },
          '100%': { opacity: '1', transform: 'scale(1) rotate(-3deg)' },
        },
      },
    },
  },
  plugins: [],
}
