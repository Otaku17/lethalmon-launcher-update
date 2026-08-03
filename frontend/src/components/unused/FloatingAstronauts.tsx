import { useEffect, useRef, useState } from 'react';
import sprite0 from '../assets/images/sprite_float_0.png';
import sprite1 from '../assets/images/sprite_float_1.png';
import sprite2 from '../assets/images/sprite_float_2.png';
import sprite3 from '../assets/images/sprite_float_3.png';
import sprite4 from '../assets/images/sprite_float_4.png';
import sprite5 from '../assets/images/sprite_float_5.png';
import sprite6 from '../assets/images/sprite_float_6.png';
import './FloatingAstronauts.css';

const SPRITES = [sprite0, sprite1, sprite2, sprite3, sprite4, sprite5, sprite6];
const SIZE = 40;
const MIN_SPEED = 20;
const MAX_SPEED = 110;
const BASE_SPEED = 45;

function FloatingAstronauts() {
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [sprite] = useState(
    () => SPRITES[Math.floor(Math.random() * SPRITES.length)],
  );

  useEffect(() => {
    const container = containerRef.current;
    const img = imgRef.current;
    if (!container || !img) return;

    let width = container.clientWidth;
    let height = container.clientHeight;

    let x = Math.random() * Math.max(width - SIZE, 0);
    let y = Math.random() * Math.max(height - SIZE, 0);
    const startAngle = Math.random() * Math.PI * 2;
    let vx = Math.cos(startAngle) * BASE_SPEED;
    let vy = Math.sin(startAngle) * BASE_SPEED;
    let rotation = 0;
    let rotationSpeed = (Math.random() < 0.5 ? -1 : 1) * (50 + Math.random() * 80);
    let nextSpeedChange = 2 + Math.random() * 3;
    let running = true;
    let lastTime = performance.now();
    let animationFrame = 0;

    const resize = () => {
      width = container.clientWidth;
      height = container.clientHeight;
    };

    const jitterBounce = () => {
      const speed = Math.hypot(vx, vy);
      const angle = Math.atan2(vy, vx);
      const jitter = (Math.random() - 0.5) * (Math.PI / 2); // ±45°
      const newAngle = angle + jitter;
      vx = Math.cos(newAngle) * speed;
      vy = Math.sin(newAngle) * speed;
      rotationSpeed = (Math.random() < 0.5 ? -1 : 1) * (50 + Math.random() * 80);
    };

    const step = (time: number) => {
      if (!running) return;
      const dt = Math.min((time - lastTime) / 1000, 0.05);
      lastTime = time;

      nextSpeedChange -= dt;
      if (nextSpeedChange <= 0) {
        const currentSpeed = Math.hypot(vx, vy);
        const targetSpeed = Math.max(
          MIN_SPEED,
          Math.min(MAX_SPEED, currentSpeed * (0.6 + Math.random() * 0.9)),
        );
        const angle = Math.atan2(vy, vx);
        vx = Math.cos(angle) * targetSpeed;
        vy = Math.sin(angle) * targetSpeed;
        nextSpeedChange = 2 + Math.random() * 3;
      }

      x += vx * dt;
      y += vy * dt;
      rotation += rotationSpeed * dt;

      let bounced = false;
      const maxX = Math.max(width - SIZE, 0);
      const maxY = Math.max(height - SIZE, 0);

      if (x <= 0) {
        x = 0;
        vx = Math.abs(vx);
        bounced = true;
      } else if (x >= maxX) {
        x = maxX;
        vx = -Math.abs(vx);
        bounced = true;
      }

      if (y <= 0) {
        y = 0;
        vy = Math.abs(vy);
        bounced = true;
      } else if (y >= maxY) {
        y = maxY;
        vy = -Math.abs(vy);
        bounced = true;
      }

      if (bounced) jitterBounce();

      img.style.transform = `translate(${x}px, ${y}px) rotate(${rotation}deg)`;

      animationFrame = requestAnimationFrame(step);
    };

    const handleVisibility = () => {
      if (document.hidden) {
        running = false;
        cancelAnimationFrame(animationFrame);
      } else if (!running) {
        running = true;
        lastTime = performance.now();
        animationFrame = requestAnimationFrame(step);
      }
    };

    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(container);
    document.addEventListener('visibilitychange', handleVisibility);
    animationFrame = requestAnimationFrame(step);

    return () => {
      running = false;
      cancelAnimationFrame(animationFrame);
      resizeObserver.disconnect();
      document.removeEventListener('visibilitychange', handleVisibility);
    };
  }, []);

  return (
    <div className="floating-astronauts" ref={containerRef}>
      <img
        ref={imgRef}
        src={sprite}
        alt=""
        className="floating-astronaut"
        style={{ width: SIZE }}
      />
    </div>
  );
}

export default FloatingAstronauts;
