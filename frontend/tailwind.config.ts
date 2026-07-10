import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "#36e0b5",
          dark: "#21bd96",
        },
        mint: "#36e0b5",
        panel: "#101719",
        "panel-raised": "#141d20",
      },
    },
  },
  plugins: [],
};

export default config;
