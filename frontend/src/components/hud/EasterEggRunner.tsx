import { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { FaXmark } from 'react-icons/fa6';
import Button from './Button';
import './EasterEggRunner.css';
import noWalkSprite from '../../assets/images/game/player_no_walk.png';
import runSprite from '../../assets/images/game/player_run.png';
import duckSprite from '../../assets/images/game/player_duck.png';
import deadSprite from '../../assets/images/game/player_dead.png';
import playerManifest from '../../assets/images/game/player.json';
import cactusSmallSheet from '../../assets/images/game/cactus_small.png';
import cactusHighSheet from '../../assets/images/game/cactus_high.png';
import cactusManifest from '../../assets/images/game/cactus.json';
import pteraSheet from '../../assets/images/game/ptera.png';
import pteraManifest from '../../assets/images/game/ptera.json';
import cloudSmallSheet from '../../assets/images/game/cloud_small.png';
import cloudHighSheet from '../../assets/images/game/cloud_high.png';
import groundSheet from '../../assets/images/game/ground.png';

interface SpriteFrame {
  x: number;
  y: number;
  width: number;
  height: number;
}

// Pixel-art scale — both the player and the cactus are shown at their
// native sprite size (no upscaling).
const PLAYER_SCALE = 1;
const CACTUS_SCALE = 1;
const PLAYER_X = 60;
const FRAME_DURATION_MS = 90;

const RUN_FRAME = playerManifest.run[0];
const DUCK_FRAME = playerManifest.duck[0];
const PLAYER_WIDTH = RUN_FRAME.width * PLAYER_SCALE;
const PLAYER_HEIGHT = RUN_FRAME.height * PLAYER_SCALE;
const DUCK_WIDTH = DUCK_FRAME.width * PLAYER_SCALE;
const DUCK_HEIGHT = DUCK_FRAME.height * PLAYER_SCALE;

// Every player sprite frame is opaque edge-to-edge (no transparent padding
// to trim, unlike the very first placeholder sprite this replaced), so the
// hitbox is just the frame itself shrunk a bit for fairness — a slightly
// forgiving hitbox reads as fair; a strict one doesn't.
const HITBOX_MARGIN = 0.15;
const PLAYER_HITBOX = {
  left: PLAYER_WIDTH * HITBOX_MARGIN,
  right: PLAYER_WIDTH * (1 - HITBOX_MARGIN),
  top: PLAYER_HEIGHT * HITBOX_MARGIN,
  bottom: PLAYER_HEIGHT * (1 - HITBOX_MARGIN),
};
const PLAYER_DUCK_HITBOX = {
  left: DUCK_WIDTH * HITBOX_MARGIN,
  right: DUCK_WIDTH * (1 - HITBOX_MARGIN),
  top: DUCK_HEIGHT * HITBOX_MARGIN,
  bottom: DUCK_HEIGHT * (1 - HITBOX_MARGIN),
};

const CANVAS_WIDTH = 760;
const CANVAS_HEIGHT = 240;
const GROUND_Y = CANVAS_HEIGHT - 40;

// Generous jump: gives a real forgiving window to clear a cactus, not a
// single frame-perfect instant at the peak.
const GRAVITY = 0.003; // px/ms^2
const JUMP_VELOCITY = -0.95; // px/ms

// Ptera flies at one of three altitudes, like the official game: low (sits
// on the ground like a cactus, must be jumped), mid (overlaps the standing
// hitbox but clears the duck hitbox, must be ducked), and high (clears the
// standing hitbox entirely — safe to just run under, though a mistimed jump
// can still clank into it, same as the original).
const STANDING_TOP_ABOVE_GROUND = PLAYER_HEIGHT - PLAYER_HITBOX.top;
const DUCK_TOP_ABOVE_GROUND = DUCK_HEIGHT - PLAYER_DUCK_HITBOX.top;
const PTERA_ALTITUDES = {
  low: 0,
  mid: DUCK_TOP_ABOVE_GROUND + 4,
  high: STANDING_TOP_ABOVE_GROUND + 6,
} as const;
const PTERA_FRAME_MS = 150;

// Obstacle spacing, adapted from the actual formula the official Chrome
// dino game uses (`minGap = obstacleWidth * speed + minGapConstant`,
// `maxGap = minGap * 1.5`, then a random pick in between) — gaps naturally
// tighten as speed increases and widen for bigger/clustered obstacles,
// instead of a hand-tuned "difficulty" timer. `speed` there is in px/frame
// at ~60fps, so ours (px/ms) is converted to the same unit for the formula
// to behave the same way.
const CHROME_FRAME_MS = 1000 / 60;
const GAP_MIN_CONSTANT = 150;
const GAP_MAX_COEFFICIENT = 1.5;
const CACTUS_CLUSTER_GAP = 2; // px between cacti within the same cluster

// Clouds are purely decorative background parallax, exactly like the
// official game: they drift at a fixed slow speed regardless of the current
// game speed (so they visibly lag behind the ground/obstacles), float in
// the upper sky band, and never take part in collision.
const CLOUD_SPEED = 0.05; // px/ms, constant
const CLOUD_MIN_GAP_PX = 100;
const CLOUD_MAX_GAP_PX = 400;
const CLOUD_MIN_Y = 15;
const CLOUD_MAX_Y = GROUND_Y - 90;
const CLOUD_SMALL_WIDTH = 46;
const CLOUD_SMALL_HEIGHT = 13;
const CLOUD_HIGH_WIDTH = 92;
const CLOUD_HIGH_HEIGHT = 27;

// The ground texture is a fixed-width strip that has to be tiled
// side-by-side and scrolled in lockstep with the obstacles to read as an
// infinite floor.
const GROUND_IMG_WIDTH = 1200;

const BEST_SCORE_STORAGE_KEY = 'easterEggRunner.bestScore';

type ObstacleKind = 'cactus' | 'ptera';
type CactusFamily = 'small' | 'high';

interface Obstacle {
  x: number;
  width: number;
  height: number;
  kind: ObstacleKind;
  /** Sprite sheet's own bottom-up sprite index — cactus variant, unused for ptera. */
  variant: number;
  /** Small or large cactus sheet. Only used by 'cactus'. */
  family: CactusFamily;
  /** How high off the ground the obstacle's bottom edge sits. 0 for every cactus; one of PTERA_ALTITUDES for ptera. */
  bottomAboveGround: number;
}

type CloudVariant = 'small' | 'high';

interface Cloud {
  x: number;
  y: number;
  width: number;
  height: number;
  variant: CloudVariant;
}

type GameState = 'ready' | 'playing' | 'gameover';

interface EasterEggRunnerProps {
  onClose: () => void;
}

/**
 * Hidden "dino run"-style mini-game, unlocked by clicking the home page
 * logo repeatedly (see HomePage). Runs entirely on a <canvas>. The player
 * has four poses (assets/images/game/player_*.png + player.json): idle
 * before the game starts, a 2-frame run/duck cycle while playing, and a
 * 2-frame death animation that plays once (not looping) on game over.
 * Ground obstacles are cactus sprites (small/large, clustered like the
 * official game) to jump over. A flying ptera (ptera.png + .json, 2-frame
 * flap) shows up at one of 3 altitudes: low (jump it, like a cactus), mid
 * (duck under it), or high (clears the standing hitbox — safe to run
 * under).
 * Space/click to jump, arrow down to duck, (re)start on click/space,
 * Escape to close.
 */
export default function EasterEggRunner({ onClose }: EasterEggRunnerProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const context = canvas.getContext('2d');
    if (!context) return;
    // Re-bound to a non-null-typed const: nested function declarations below
    // don't retain the `if (!context) return` narrowing on their own.
    const ctx: CanvasRenderingContext2D = context;
    ctx.imageSmoothingEnabled = false;

    const noWalk = new Image();
    noWalk.src = noWalkSprite;
    const run = new Image();
    run.src = runSprite;
    const duck = new Image();
    duck.src = duckSprite;
    const dead = new Image();
    dead.src = deadSprite;
    const cactusSmall = new Image();
    cactusSmall.src = cactusSmallSheet;
    const cactusHigh = new Image();
    cactusHigh.src = cactusHighSheet;
    const ptera = new Image();
    ptera.src = pteraSheet;
    const cloudSmall = new Image();
    cloudSmall.src = cloudSmallSheet;
    const cloudHigh = new Image();
    cloudHigh.src = cloudHighSheet;
    const ground = new Image();
    ground.src = groundSheet;

    let state: GameState = 'ready';
    let playerY = 0; // 0 = on the ground, negative = airborne offset
    let velocityY = 0;
    let onGround = true;
    let ducking = false;
    let frameIndex = 0;
    let frameTimer = 0;
    let deadFrameIndex = 0;
    let deadFrameTimer = 0;
    let deadAnimationDone = false;
    let obstacles: Obstacle[] = [];
    let spawnTimer = 0;
    let nextSpawnIn = 900;
    let clouds: Cloud[] = [];
    let cloudSpawnTimer = 0;
    let cloudNextSpawnIn = 0;
    let groundScrollX = 0;
    let speed = 0.32; // px/ms
    let distance = 0;
    let bestScore = Number(localStorage.getItem(BEST_SCORE_STORAGE_KEY)) || 0;
    let lastTime = 0;
    let animationFrame = 0;
    let running = true;

    function reset() {
      playerY = 0;
      velocityY = 0;
      onGround = true;
      ducking = false;
      frameIndex = 0;
      frameTimer = 0;
      deadFrameIndex = 0;
      deadFrameTimer = 0;
      deadAnimationDone = false;
      obstacles = [];
      spawnTimer = 0;
      nextSpawnIn = 900;
      groundScrollX = 0;
      speed = 0.32;
      distance = 0;
    }

    function jump() {
      if (state === 'ready') {
        state = 'playing';
        return;
      }
      if (state === 'gameover') {
        reset();
        state = 'playing';
        return;
      }
      if (onGround) {
        velocityY = JUMP_VELOCITY;
        onGround = false;
      }
    }

    function spawnCloud() {
      const variant: CloudVariant = Math.random() < 0.6 ? 'small' : 'high';
      const width = variant === 'small' ? CLOUD_SMALL_WIDTH : CLOUD_HIGH_WIDTH;
      const height = variant === 'small' ? CLOUD_SMALL_HEIGHT : CLOUD_HIGH_HEIGHT;
      const y = CLOUD_MIN_Y + Math.random() * (CLOUD_MAX_Y - CLOUD_MIN_Y);
      clouds.push({ x: CANVAS_WIDTH + 10, y, width, height, variant });
    }

    // A cactus sprite noticeably wider than the rest of its family (e.g. a
    // multi-cactus cluster sprite in cactus.json) is treated as rarer,
    // appearing far less often than a plain uniform pick would give it.
    function pickCactusVariant(frames: readonly SpriteFrame[]): number {
      if (frames.length <= 1) return 0;
      const avgWidth = frames.reduce((sum, f) => sum + f.width, 0) / frames.length;
      const weights = frames.map((f) => (f.width > avgWidth * 1.4 ? 0.15 : 1));
      const total = weights.reduce((a, b) => a + b, 0);
      let r = Math.random() * total;
      for (let i = 0; i < frames.length; i++) {
        r -= weights[i];
        if (r <= 0) return i;
      }
      return frames.length - 1;
    }

    // Spawns one obstacle "event" — a single barrier, or a cluster of 1-3
    // cacti packed tightly together, mirroring the real Chrome dino's
    // grouped cactus obstacles — and returns the total width it occupies so
    // the caller can size the gap to the next spawn accordingly.
    function spawnObstacle(): number {
      // Ptera (flying, one of 3 altitudes) shows up less often than cacti,
      // same as the official game.
      if (Math.random() < 0.2) {
        const roll = Math.random();
        const altitude = roll < 0.34 ? 'low' : roll < 0.67 ? 'mid' : 'high';
        const frame = pteraManifest.pos[0];
        obstacles.push({
          x: CANVAS_WIDTH + 10,
          width: frame.width,
          height: frame.height,
          kind: 'ptera',
          variant: 0,
          family: 'small',
          bottomAboveGround: PTERA_ALTITUDES[altitude],
        });
        return frame.width;
      }

      // Small cacti cluster in groups of 1-3, large ones in 1-2, matching
      // the real game's grouping per cactus size.
      const family: CactusFamily = Math.random() < 0.65 ? 'small' : 'high';
      const frames = cactusManifest[family];
      const clusterRoll = Math.random();
      const clusterSize =
        family === 'small'
          ? clusterRoll < 0.5
            ? 1
            : clusterRoll < 0.8
              ? 2
              : 3
          : clusterRoll < 0.65
            ? 1
            : 2;

      const startX = CANVAS_WIDTH + 10;
      let x = startX;
      for (let i = 0; i < clusterSize; i++) {
        const variant = pickCactusVariant(frames);
        const frame = frames[variant];
        const width = frame.width * CACTUS_SCALE;
        obstacles.push({
          x,
          width,
          height: frame.height * CACTUS_SCALE,
          kind: 'cactus',
          variant,
          family,
          bottomAboveGround: 0,
        });
        x += width + CACTUS_CLUSTER_GAP;
      }
      return x - CACTUS_CLUSTER_GAP - startX;
    }

    // The official Chrome dino's own spacing formula: the gap to the next
    // obstacle scales with both the current speed and the width of what was
    // just spawned (a wider cluster demands more room), picked randomly
    // between that minimum and 1.5x it.
    function scheduleNextSpawn(obstacleWidth: number) {
      const frameSpeed = speed * CHROME_FRAME_MS;
      const minGapPx = obstacleWidth * frameSpeed + GAP_MIN_CONSTANT;
      const maxGapPx = minGapPx * GAP_MAX_COEFFICIENT;
      const gapPx = minGapPx + Math.random() * (maxGapPx - minGapPx);
      nextSpawnIn = gapPx / speed;
    }

    function drawCactus(baseX: number, baseY: number, width: number, height: number, variant: number, family: CactusFamily) {
      const frames = cactusManifest[family];
      const frame = frames[variant % frames.length];
      const sheet = family === 'small' ? cactusSmall : cactusHigh;
      // The tall cactus sprite sits a couple px too high above the ground
      // line at its native crop — nudge it down to sit flush.
      const groundOffset = family === 'high' ? 2 : 0;
      if (sheet.complete) {
        ctx.drawImage(sheet, frame.x, frame.y, frame.width, frame.height, baseX, baseY - height + groundOffset, width, height);
      }
    }

    function drawGround() {
      if (!ground.complete) return;
      let x = -(groundScrollX % GROUND_IMG_WIDTH);
      // Purely visual nudge — the ground texture sits 10px too low against
      // its native crop; everything else (player, obstacles, GROUND_Y
      // itself) is untouched.
      while (x < CANVAS_WIDTH) {
        ctx.drawImage(ground, x, GROUND_Y - 10);
        x += GROUND_IMG_WIDTH;
      }
    }

    function drawCloud(c: Cloud) {
      const sheet = c.variant === 'small' ? cloudSmall : cloudHigh;
      if (sheet.complete) {
        ctx.drawImage(sheet, c.x, c.y, c.width, c.height);
      }
    }

    function drawPtera(baseX: number, width: number, height: number, bottomAboveGround: number) {
      const frameIdx = Math.floor(performance.now() / PTERA_FRAME_MS) % pteraManifest.pos.length;
      const frame = pteraManifest.pos[frameIdx];
      const baseY = GROUND_Y - bottomAboveGround;
      if (ptera.complete) {
        ctx.drawImage(ptera, frame.x, frame.y, frame.width, frame.height, baseX, baseY - height, width, height);
      }
    }

    function drawObstacle(o: Obstacle) {
      const baseX = o.x;

      if (o.kind === 'cactus') {
        drawCactus(baseX, GROUND_Y, o.width, o.height, o.variant, o.family);
      } else {
        drawPtera(baseX, o.width, o.height, o.bottomAboveGround);
      }
    }

    function collides(o: Obstacle) {
      const isDucking = ducking && onGround;
      const poseHeight = isDucking ? DUCK_HEIGHT : PLAYER_HEIGHT;
      const hitbox = isDucking ? PLAYER_DUCK_HITBOX : PLAYER_HITBOX;
      const drawY = GROUND_Y - poseHeight + playerY;
      const pLeft = PLAYER_X + hitbox.left;
      const pRight = PLAYER_X + hitbox.right;
      const pTop = drawY + hitbox.top;
      const pBottom = drawY + hitbox.bottom;

      const oLeft = o.x;
      const oRight = o.x + o.width;
      const oBottom = GROUND_Y - o.bottomAboveGround;
      const oTop = oBottom - o.height;

      return pRight > oLeft && pLeft < oRight && pBottom > oTop && pTop < oBottom;
    }

    function step(timestamp: number) {
      if (!running) return;
      const delta = lastTime ? Math.min(timestamp - lastTime, 40) : 16;
      lastTime = timestamp;

      ctx.clearRect(0, 0, CANVAS_WIDTH, CANVAS_HEIGHT);

      for (const c of clouds) drawCloud(c);
      drawGround();

      if (state === 'playing') {
        distance += delta * speed * 0.06;
        // Ramps up further and caps later than before — it was reaching its
        // ceiling too early (around a score of 1100) and staying flat for
        // the rest of the run.
        speed = Math.min(1.05, 0.32 + distance / 6000);
        groundScrollX += speed * delta;

        if (Math.floor(distance) > bestScore) {
          bestScore = Math.floor(distance);
          localStorage.setItem(BEST_SCORE_STORAGE_KEY, String(bestScore));
        }

        velocityY += GRAVITY * delta;
        playerY += velocityY * delta;
        if (playerY >= 0) {
          playerY = 0;
          velocityY = 0;
          onGround = true;
        }

        if (onGround) {
          frameTimer += delta;
          if (frameTimer >= FRAME_DURATION_MS) {
            frameTimer = 0;
            frameIndex += 1;
          }
        }

        spawnTimer += delta;
        if (spawnTimer >= nextSpawnIn) {
          spawnTimer = 0;
          const width = spawnObstacle();
          scheduleNextSpawn(width);
        }

        // Clouds drift at their own fixed slow speed, independent of the
        // ramping game speed, for a parallax background layer.
        cloudSpawnTimer += delta;
        if (cloudSpawnTimer >= cloudNextSpawnIn) {
          cloudSpawnTimer = 0;
          spawnCloud();
          cloudNextSpawnIn = (CLOUD_MIN_GAP_PX + Math.random() * (CLOUD_MAX_GAP_PX - CLOUD_MIN_GAP_PX)) / CLOUD_SPEED;
        }
        for (const c of clouds) c.x -= CLOUD_SPEED * delta;
        clouds = clouds.filter((c) => c.x + c.width > -10);

        for (const o of obstacles) o.x -= speed * delta;
        obstacles = obstacles.filter((o) => o.x + o.width > -10);

        for (const o of obstacles) {
          if (collides(o)) {
            state = 'gameover';
            deadFrameIndex = 0;
            deadFrameTimer = 0;
            deadAnimationDone = false;
            break;
          }
        }
      } else if (state === 'gameover' && !deadAnimationDone) {
        deadFrameTimer += delta;
        if (deadFrameTimer >= FRAME_DURATION_MS) {
          deadFrameTimer = 0;
          if (deadFrameIndex < playerManifest.dead[0].pos.length - 1) {
            deadFrameIndex += 1;
          } else {
            deadAnimationDone = true;
          }
        }
      }

      for (const o of obstacles) drawObstacle(o);

      // Pick which pose to draw: idle before the game starts, the death
      // animation (played once) on game over, otherwise run/duck.
      let activeSheet = run;
      let frame: SpriteFrame = RUN_FRAME;
      let poseWidth = PLAYER_WIDTH;
      let poseHeight = PLAYER_HEIGHT;

      if (state === 'ready') {
        activeSheet = noWalk;
        frame = playerManifest.no_walk[0];
      } else if (state === 'gameover') {
        activeSheet = dead;
        frame = playerManifest.dead[0].pos[deadFrameIndex];
      } else if (ducking && onGround) {
        activeSheet = duck;
        frame = playerManifest.duck[frameIndex % playerManifest.duck.length];
        poseWidth = DUCK_WIDTH;
        poseHeight = DUCK_HEIGHT;
      } else {
        frame = playerManifest.run[frameIndex % playerManifest.run.length];
      }

      const drawY = GROUND_Y - poseHeight + playerY;
      if (activeSheet.complete) {
        ctx.drawImage(activeSheet, frame.x, frame.y, frame.width, frame.height, PLAYER_X, drawY, poseWidth, poseHeight);
      }

      ctx.fillStyle = 'rgba(234, 246, 248, 0.9)';
      ctx.font = '14px "Share Tech Mono", monospace';
      ctx.textAlign = 'left';
      ctx.fillText(`HI ${bestScore}   SCORE ${Math.floor(distance)}`, 12, 24);

      if (state !== 'playing') {
        ctx.fillStyle = 'rgba(5, 8, 13, 0.55)';
        ctx.fillRect(0, 0, CANVAS_WIDTH, CANVAS_HEIGHT);
        ctx.textAlign = 'center';
        ctx.fillStyle = '#eaf6f8';
        ctx.font = '700 18px "Orbitron", sans-serif';
        ctx.fillText(
          state === 'ready' ? 'PRESS SPACE OR CLICK TO START' : `GAME OVER — SCORE ${Math.floor(distance)}`,
          CANVAS_WIDTH / 2,
          CANVAS_HEIGHT / 2,
        );
        ctx.font = '12px "Share Tech Mono", monospace';
        ctx.fillText(
          state === 'gameover' ? 'PRESS SPACE OR CLICK TO RETRY' : 'HOLD ARROW DOWN TO DUCK UNDER LOW FLYERS',
          CANVAS_WIDTH / 2,
          CANVAS_HEIGHT / 2 + 26,
        );
      }

      animationFrame = requestAnimationFrame(step);
    }

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === ' ' || e.key === 'ArrowUp') {
        e.preventDefault();
        jump();
      } else if (e.key === 'ArrowDown' || e.key === 's' || e.key === 'S') {
        e.preventDefault();
        ducking = true;
      } else if (e.key === 'Escape') {
        onClose();
      }
    }

    function handleKeyUp(e: KeyboardEvent) {
      if (e.key === 'ArrowDown' || e.key === 's' || e.key === 'S') {
        ducking = false;
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    canvas.addEventListener('click', jump);
    animationFrame = requestAnimationFrame(step);

    return () => {
      running = false;
      cancelAnimationFrame(animationFrame);
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
      canvas.removeEventListener('click', jump);
    };
  }, [onClose]);

  return createPortal(
    <div className="easter-egg-overlay">
      <div className="easter-egg-box">
        <canvas ref={canvasRef} width={CANVAS_WIDTH} height={CANVAS_HEIGHT} className="easter-egg-canvas" />
        <Button
          variant="ghost"
          iconOnly
          icon={<FaXmark />}
          aria-label="Close"
          className="easter-egg-close"
          onClick={onClose}
        />
      </div>
    </div>,
    document.body,
  );
}
