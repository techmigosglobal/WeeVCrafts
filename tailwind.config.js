/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./web/**/*.templ', './internal/web/**/*.go', './internal/transport/web/templates/**/*.html'],
  theme: {
    extend: {
      colors: {
        paper: '#fffaf4',
        mist: '#eef2f6',
        ink: '#162238',
        indigo: '#1e3158',
        citrus: '#f79032',
        line: '#cdd3dc',
      },
      fontFamily: {
        display: ['Georgia', 'serif'],
      },
    },
  },
  plugins: [],
};
