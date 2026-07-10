/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export served by nginx in production.
  output: "export",
  // API base URL is read at runtime from the browser (defaults to same origin),
  // so the bundle is environment-agnostic.
  images: { unoptimized: true },
  trailingSlash: true,
};

export default nextConfig;
