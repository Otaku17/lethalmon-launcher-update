import { useEffect, useRef } from 'react';
import './ParticleBackground.css';

interface GridNode {
  x: number;
  y: number;
  phase: number;
  speed: number;
  magenta: boolean;
  baseRadius: number;
}

interface Connection {
  a: number;
  b: number;
}

const CELL = 68; // grid cell size, in CSS px
const NODE_CHANCE = 0.05; // odds that a grid intersection becomes a node
const MAX_NODES = 46;
const CONNECTION_DIST = CELL * 1.6;
const CONNECTION_CHANCE = 0.5;

// Animated canvas background: a faint drifting grid with glowing nodes,
// circuit-board-style connections between nearby nodes, and a slow scanning
// band. Purely decorative — pauses itself while the tab is hidden.
function ParticleBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d', { alpha: true });
    if (!ctx) return;

    let width = 0;
    let height = 0;
    let dpr = Math.min(window.devicePixelRatio || 1, 2);
    let nodes: GridNode[] = [];
    let connections: Connection[] = [];
    let gridOffset = 0;
    let scanY = 0;
    let lastTime = 0;
    let animationFrame = 0;
    let running = true;

    const resize = () => {
      const parent = canvas.parentElement;
      if (!parent) return;
      width = parent.clientWidth;
      height = parent.clientHeight;
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      // Nodes placed on grid intersections.
      const cols = Math.ceil(width / CELL) + 1;
      const rows = Math.ceil(height / CELL) + 1;
      nodes = [];
      for (let gx = 0; gx <= cols; gx++) {
        for (let gy = 0; gy <= rows; gy++) {
          if (Math.random() < NODE_CHANCE) {
            nodes.push({
              x: gx * CELL,
              y: gy * CELL,
              phase: Math.random() * Math.PI * 2,
              speed: 0.0015 + Math.random() * 0.002,
              magenta: Math.random() < 0.18,
              baseRadius: 1.4 + Math.random() * 1.6,
            });
          }
        }
      }

      // On a large screen, cap the node count to stay lightweight.
      if (nodes.length > MAX_NODES) {
        nodes = nodes.sort(() => Math.random() - 0.5).slice(0, MAX_NODES);
      }

      // Connects nearby nodes to each other, circuit-board style.
      connections = [];
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[i].x - nodes[j].x;
          const dy = nodes[i].y - nodes[j].y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < CONNECTION_DIST && Math.random() < CONNECTION_CHANCE) {
            connections.push({ a: i, b: j });
          }
        }
      }

      scanY = -height * 0.2;
    };

    const drawGrid = () => {
      ctx.strokeStyle = 'rgba(77, 238, 234, 0.05)';
      ctx.lineWidth = 1;
      const offsetY = gridOffset % CELL;

      ctx.beginPath();
      // Fixed verticals.
      for (let x = 0; x <= width; x += CELL) {
        ctx.moveTo(x + 0.5, 0);
        ctx.lineTo(x + 0.5, height);
      }
      // Horizontals that drift slowly -> subtle "data flow" effect.
      for (let y = -CELL; y <= height + CELL; y += CELL) {
        const yy = y + offsetY;
        ctx.moveTo(0, yy + 0.5);
        ctx.lineTo(width, yy + 0.5);
      }
      ctx.stroke();
    };

    const drawConnections = () => {
      ctx.lineWidth = 1;
      for (const c of connections) {
        const a = nodes[c.a];
        const b = nodes[c.b];
        if (!a || !b) continue;
        const alpha = 0.04 + 0.05 * (0.5 + 0.5 * Math.sin(a.phase + b.phase));
        ctx.strokeStyle = `rgba(77, 238, 234, ${alpha})`;
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.lineTo(b.x, b.y);
        ctx.stroke();
      }
    };

    const drawNodes = () => {
      for (const n of nodes) {
        const pulse = 0.5 + 0.5 * Math.sin(n.phase);
        const alpha = 0.25 + pulse * 0.55;
        const radius = n.baseRadius * (0.8 + pulse * 0.5);
        const color = n.magenta ? '255, 61, 129' : '77, 238, 234';

        const glow = ctx.createRadialGradient(
          n.x,
          n.y,
          0,
          n.x,
          n.y,
          radius * 5,
        );
        glow.addColorStop(0, `rgba(${color}, ${alpha * 0.35})`);
        glow.addColorStop(1, `rgba(${color}, 0)`);
        ctx.fillStyle = glow;
        ctx.beginPath();
        ctx.arc(n.x, n.y, radius * 5, 0, Math.PI * 2);
        ctx.fill();

        ctx.fillStyle = `rgba(${color}, ${alpha})`;
        ctx.beginPath();
        ctx.arc(n.x, n.y, radius, 0, Math.PI * 2);
        ctx.fill();
      }
    };

    const drawScan = () => {
      const bandHeight = height * 0.22;
      const gradient = ctx.createLinearGradient(
        0,
        scanY - bandHeight,
        0,
        scanY + bandHeight,
      );
      gradient.addColorStop(0, 'rgba(77, 238, 234, 0)');
      gradient.addColorStop(0.5, 'rgba(77, 238, 234, 0.05)');
      gradient.addColorStop(1, 'rgba(77, 238, 234, 0)');
      ctx.fillStyle = gradient;
      ctx.fillRect(0, scanY - bandHeight, width, bandHeight * 2);

      ctx.strokeStyle = 'rgba(77, 238, 234, 0.35)';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(0, scanY);
      ctx.lineTo(width, scanY);
      ctx.stroke();
    };

    const step = (timestamp: number) => {
      if (!running) return;
      const delta = lastTime ? timestamp - lastTime : 16;
      lastTime = timestamp;

      ctx.clearRect(0, 0, width, height);

      gridOffset += delta * 0.006;
      scanY += delta * 0.035;
      if (scanY > height * 1.2) scanY = -height * 0.2;

      for (const n of nodes) n.phase += n.speed * delta;

      drawGrid();
      drawConnections();
      drawScan();
      drawNodes();

      animationFrame = requestAnimationFrame(step);
    };

    const handleVisibility = () => {
      if (document.hidden) {
        running = false;
        cancelAnimationFrame(animationFrame);
      } else if (!running) {
        running = true;
        lastTime = 0;
        animationFrame = requestAnimationFrame(step);
      }
    };

    resize();
    animationFrame = requestAnimationFrame(step);

    const resizeObserver = new ResizeObserver(resize);
    if (canvas.parentElement) resizeObserver.observe(canvas.parentElement);
    document.addEventListener('visibilitychange', handleVisibility);

    return () => {
      running = false;
      cancelAnimationFrame(animationFrame);
      resizeObserver.disconnect();
      document.removeEventListener('visibilitychange', handleVisibility);
    };
  }, []);

  return (
    <div className="particle-background-wrap">
      <canvas ref={canvasRef} className="particle-background" />
    </div>
  );
}

export default ParticleBackground;
