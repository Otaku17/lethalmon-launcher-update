import { useEffect, useRef, useState } from 'react';
import './ThemedImage.css';

export interface ThemedImageProps {
  src: string;
  alt: string;
  /** Color for the image's dark areas. Accepts any CSS color, including a
   *  variable (`var(--bg)`). */
  lowColor?: string;
  /** Color for the image's light areas. */
  highColor?: string;
  /** 0 = original image, 1 = full duotone. Values in between blend the two. */
  intensity?: number;
  /** Adds a subtle scanline effect on top of the image. */
  scanlines?: boolean;
  /** Adds a cyan glow around the image once loaded. */
  glow?: boolean;
  className?: string;
}

/**
 * Recolors any image to adopt the theme's palette (by default: shadows ->
 * --bg, highlights -> --cyan), similar to a "duotone" shader. Processing
 * happens on a <canvas>, pixel by pixel.
 *
 * <ThemedImage src="/screenshots/cockpit.png" alt="Ship cockpit" />
 * <ThemedImage src="/photo.jpg" alt="..." lowColor="#0a1119" highColor="var(--magenta)" intensity={0.7} />
 *
 * ⚠️ If the image comes from another domain, that domain must allow CORS
 * (otherwise reading pixels is blocked by the browser). Local/same-origin
 * images work with no configuration needed.
 * In Next.js (App Router), add `'use client'` at the top of the file that
 * uses this component.
 * `.svg` files are detected automatically and rasterized at a comfortable
 * resolution (≥1200px on the longest side) instead of their default
 * "naturalWidth", which is often tiny, to avoid blur/aliasing.
 */
export default function ThemedImage({
  src,
  alt,
  lowColor = 'var(--bg)',
  highColor = 'var(--cyan)',
  intensity = 1,
  scanlines = true,
  glow = true,
  className = '',
}: ThemedImageProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoaded(false);
    setFailed(false);

    const canvas = canvasRef.current;
    if (!canvas) return;

    const img = new Image();
    img.crossOrigin = 'anonymous';

    img.onload = () => {
      if (cancelled) return;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      // An SVG loaded into an <img> often has a tiny naturalWidth/
      // naturalHeight (the browser picks an arbitrary default size, e.g.
      // 300x150), since the vector has no inherent "resolution". If we draw
      // the canvas at that size, the SVG gets rasterized at low resolution
      // then stretched via CSS -> blur/aliasing, whereas an SVG should stay
      // sharp at any size.
      // We detect this case and force a comfortable render size, preserving
      // the original aspect ratio.
      const isSvg = /\.svg(\?.*)?$/i.test(src) || src.startsWith('data:image/svg');
      const MIN_SVG_SIZE = 1200;

      let renderWidth = img.naturalWidth;
      let renderHeight = img.naturalHeight;

      if (isSvg) {
        const aspect = img.naturalWidth / img.naturalHeight || 1;
        if (aspect >= 1) {
          renderWidth = Math.max(renderWidth, MIN_SVG_SIZE);
          renderHeight = renderWidth / aspect;
        } else {
          renderHeight = Math.max(renderHeight, MIN_SVG_SIZE);
          renderWidth = renderHeight * aspect;
        }
      }

      canvas.width = Math.round(renderWidth);
      canvas.height = Math.round(renderHeight);
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);

      try {
        const [lr, lg, lb] = resolveCssColor(lowColor);
        const [hr, hg, hb] = resolveCssColor(highColor);
        const amount = Math.min(1, Math.max(0, intensity));

        const frame = ctx.getImageData(0, 0, canvas.width, canvas.height);
        const data = frame.data;

        for (let i = 0; i < data.length; i += 4) {
          const r = data[i];
          const g = data[i + 1];
          const b = data[i + 2];
          const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;

          const dr = lr + (hr - lr) * luminance;
          const dg = lg + (hg - lg) * luminance;
          const db = lb + (hb - lb) * luminance;

          data[i] = r + (dr - r) * amount;
          data[i + 1] = g + (dg - g) * amount;
          data[i + 2] = b + (db - b) * amount;
        }

        ctx.putImageData(frame, 0, 0);
        setLoaded(true);
      } catch {
        // Tainted canvas (cross-origin image without CORS headers):
        // show the original image without recoloring instead of crashing.
        setFailed(true);
        setLoaded(true);
      }
    };

    img.onerror = () => {
      if (!cancelled) {
        setFailed(true);
        setLoaded(true);
      }
    };

    img.src = src;

    return () => { cancelled = true; };
  }, [src, lowColor, highColor, intensity]);

  const classes = [
    'hud-themed-image',
    scanlines ? 'has-scanlines' : '',
    glow ? 'has-glow' : '',
    className,
  ].filter(Boolean).join(' ');

  return (
    <div className={classes}>
      {failed ? (
        <img className="hud-themed-image__fallback is-loaded" src={src} alt={alt} />
      ) : (
        <canvas
          ref={canvasRef}
          className={`hud-themed-image__canvas ${loaded ? 'is-loaded' : ''}`}
          role="img"
          aria-label={alt}
        />
      )}
      {!loaded && <div className="hud-themed-image__placeholder" />}
    </div>
  );
}

/** Resolves any CSS color (hex, rgb, name, var(--x)...) to [r,g,b]. */
function resolveCssColor(color: string): [number, number, number] {
  if (typeof document === 'undefined') return [0, 0, 0];
  const el = document.createElement('span');
  el.style.color = color;
  el.style.display = 'none';
  document.body.appendChild(el);
  const computed = getComputedStyle(el).color; // "rgb(r, g, b)" once resolved
  document.body.removeChild(el);
  const match = computed.match(/[\d.]+/g);
  if (!match || match.length < 3) return [0, 0, 0];
  return [Number(match[0]), Number(match[1]), Number(match[2])];
}
