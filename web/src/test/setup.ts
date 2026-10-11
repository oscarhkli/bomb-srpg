import { boardOffset } from '../rendering/boardOffset';

// Reset the shared boardOffset singleton so one test's grid size can't leak into the next.
beforeEach(() => {
  boardOffset.x = 0;
  boardOffset.y = 0;
});

// jsdom does not implement Path2D.
(globalThis as Record<string, unknown>).Path2D ??= class MockPath2D {
  addPath(): void {
    /* stub */
  }
};

export function createMockGraphics() {
  return {
    x: 0,
    y: 0,
    fillStyle: vi.fn().mockReturnThis(),
    fillRect: vi.fn().mockReturnThis(),
    clear: vi.fn().mockReturnThis(),
    fillCircle: vi.fn().mockReturnThis(),
    fillPoints: vi.fn().mockReturnThis(),
    fillRoundedRect: vi.fn().mockReturnThis(),
    lineStyle: vi.fn().mockReturnThis(),
    strokeRect: vi.fn().mockReturnThis(),
    strokeRoundedRect: vi.fn().mockReturnThis(),
    strokePoints: vi.fn().mockReturnThis(),
    strokeCircle: vi.fn().mockReturnThis(),
    setInteractive: vi.fn().mockReturnThis(),
    disableInteractive: vi.fn().mockReturnThis(),
    on: vi.fn().mockReturnThis(),
    off: vi.fn().mockReturnThis(),
    setDepth: vi.fn().mockReturnThis(),
    setScrollFactor: vi.fn().mockReturnThis(),
    destroy: vi.fn(),
  };
}

export function createMockText() {
  return {
    x: 0,
    y: 0,
    height: 16,
    width: 100,
    setInteractive: vi.fn().mockReturnThis(),
    on: vi.fn().mockReturnThis(),
    setOrigin: vi.fn().mockReturnThis(),
    setDepth: vi.fn().mockReturnThis(),
    setScrollFactor: vi.fn().mockReturnThis(),
    setText: vi.fn().mockReturnThis(),
    setColor: vi.fn().mockReturnThis(),
    setVisible: vi.fn().mockReturnThis(),
    disableInteractive: vi.fn().mockReturnThis(),
    destroy: vi.fn(),
  };
}

export function createMockContainer(x = 0, y = 0) {
  return {
    x,
    y,
    add: vi.fn().mockReturnThis(),
    setSize: vi.fn().mockReturnThis(),
    setDepth: vi.fn().mockReturnThis(),
    setScrollFactor: vi.fn().mockReturnThis(),
    setInteractive: vi.fn().mockReturnThis(),
    disableInteractive: vi.fn().mockReturnThis(),
    enableFilters: vi.fn().mockReturnThis(),
    filters: {
      internal: {
        addColorMatrix: vi.fn(() => ({ colorMatrix: { desaturate: vi.fn() } })),
        clear: vi.fn(),
      },
    },
    on: vi.fn().mockReturnThis(),
    off: vi.fn().mockReturnThis(),
    destroy: vi.fn(),
  };
}

export function createMockImage(x = 0, y = 0) {
  return {
    x,
    y,
    setOrigin: vi.fn().mockReturnThis(),
    setDepth: vi.fn().mockReturnThis(),
    setScrollFactor: vi.fn().mockReturnThis(),
    setTexture: vi.fn().mockReturnThis(),
    destroy: vi.fn(),
  };
}

export function createMockSprite(x = 0, y = 0) {
  return {
    x,
    y,
    width: 64,
    height: 64,
    setOrigin: vi.fn().mockReturnThis(),
    setDepth: vi.fn().mockReturnThis(),
    setScrollFactor: vi.fn().mockReturnThis(),
    setInteractive: vi.fn().mockReturnThis(),
    disableInteractive: vi.fn().mockReturnThis(),
    setTexture: vi.fn().mockReturnThis(),
    setFrame: vi.fn().mockReturnThis(),
    setDisplaySize: vi.fn().mockReturnThis(),
    on: vi.fn().mockReturnThis(),
    off: vi.fn().mockReturnThis(),
    destroy: vi.fn(),
  };
}

export function createMockCamera(x = 0, y = 0, width = 640, height = 360, name = '') {
  return {
    x,
    y,
    width,
    height,
    name,
    scrollX: 0,
    scrollY: 0,
    setScroll: vi.fn().mockReturnThis(),
    centerOn: vi.fn(),
    fadeOut: vi.fn(),
    fadeIn: vi.fn(),
    once: vi.fn(),
  };
}

const mockGameObjectFactory = {
  sprite: vi.fn((x?: number, y?: number) => createMockSprite(x, y)),
  image: vi.fn((x?: number, y?: number) => createMockImage(x, y)),
  graphics: vi.fn(() => createMockGraphics()),
  text: vi.fn(() => createMockText()),
  container: vi.fn((x?: number, y?: number) => createMockContainer(x, y)),
  renderTexture: vi.fn(),
  tileSprite: vi.fn(),
  bitmapText: vi.fn(),
};

const mockCameraManager = {
  main: createMockCamera(),
  add: vi.fn(
    (
      x?: number,
      y?: number,
      width?: number,
      height?: number,
      makeMain?: boolean,
      name?: string
    ) => {
      const cam = createMockCamera(x, y, width, height, name);
      if (makeMain) {
        mockCameraManager.main = cam;
      }
      return cam;
    }
  ),
  remove: vi.fn(),
};

export const mockScene = {
  add: mockGameObjectFactory,
  make: mockGameObjectFactory,
  cameras: mockCameraManager,
  scale: { width: 640, height: 360 },
  scene: { restart: vi.fn(), start: vi.fn() },
  sys: { game: { config: { width: 640, height: 360 } } },
  load: {
    font: vi.fn(),
    image: vi.fn(),
    spritesheet: vi.fn(),
    atlas: vi.fn(),
    aseprite: vi.fn(),
    audio: vi.fn(),
    json: vi.fn(),
    on: vi.fn(),
    once: vi.fn(),
  },
  textures: {
    get: vi.fn((key: string) => ({ frames: { __BASE: {}, [`${key}-frame`]: {} } })),
    exists: vi.fn((_key: string) => false),
    getFrame: vi.fn(() => ({
      x: 0,
      y: 0,
      width: 100,
      height: 100,
      realWidth: 100,
      realHeight: 100,
    })),
  },
  tweens: { add: vi.fn() },
  time: { addEvent: vi.fn(), delayedCall: vi.fn() },
  events: { on: vi.fn(), once: vi.fn(), off: vi.fn(), emit: vi.fn() },
  input: {
    on: vi.fn(),
    off: vi.fn(),
    keyboard: { addKey: vi.fn(), createCursorKeys: vi.fn(() => ({})) },
  },
  children: { getAll: vi.fn(() => []) },
  game: { config: { width: 640, height: 360 } },
};

// @ts-expect-error - global mock for tests
// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
global.Phaser = {
  Game: vi.fn(),
  Scene: class {
    constructor() {
      Object.assign(this, mockScene);
    }
  },
  AUTO: 0,
  CANVAS: 1,
  WEBGL: 2,
  Scale: { NONE: 0, FIT: 3, CENTER_BOTH: 4, NO_ZOOM: 1, ZOOM_2X: 2 },
  Physics: { Arcade: {} },
  Tilemaps: {},
  Math: {
    Vector2: class {
      constructor(
        public x: number,
        public y: number
      ) {}
    },
    Between: (min: number, max: number) => Math.floor(Math.random() * (max - min + 1)) + min,
    Clamp: (v: number, min: number, max: number) => Math.max(min, Math.min(max, v)),
    Linear: (a: number, b: number, t: number) => a + (b - a) * t,
  },
  Geom: {
    Rectangle: class {
      static Contains = vi.fn();
      constructor(
        public x: number,
        public y: number,
        public width: number,
        public height: number
      ) {}
    },
    Circle: class {
      constructor(
        public x: number,
        public y: number,
        public radius: number
      ) {}
    },
    Point: class {
      constructor(
        public x: number,
        public y: number
      ) {}
    },
  },
  Display: { Color: { ValueToColor: vi.fn() } },
};

const mockContext = {
  fillRect: vi.fn(),
  clearRect: vi.fn(),
  drawImage: vi.fn(),
  getImageData: vi.fn(() => ({ data: new Uint8ClampedArray(4) })),
  putImageData: vi.fn(),
  createImageData: vi.fn(),
  setTransform: vi.fn(),
  resetTransform: vi.fn(),
  translate: vi.fn(),
  scale: vi.fn(),
  rotate: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  arc: vi.fn(),
  fill: vi.fn(),
  stroke: vi.fn(),
  closePath: vi.fn(),
  fillText: vi.fn(),
  measureText: vi.fn(() => ({ width: 0 })),
  canvas: { width: 640, height: 360 },
  globalAlpha: 1,
  globalCompositeOperation: 'source-over',
  clip: vi.fn(),
  isPointInPath: vi.fn(),
  isPointInStroke: vi.fn(),
  fillRule: 'nonzero',
  imageSmoothingEnabled: true,
  imageSmoothingQuality: 'low',
  shadowBlur: 0,
  shadowColor: 'rgba(0, 0, 0, 0)',
  shadowOffsetX: 0,
  shadowOffsetY: 0,
  font: '10px sans-serif',
  textAlign: 'start',
  textBaseline: 'alphabetic',
  direction: 'inherit',
  lineWidth: 1,
  lineCap: 'butt',
  lineJoin: 'miter',
  miterLimit: 10,
  lineDashOffset: 0,
  strokeStyle: '#000000',
  fillStyle: '#000000',
  createLinearGradient: vi.fn(),
  createRadialGradient: vi.fn(),
  createPattern: vi.fn(),
  createConicGradient: vi.fn(),
  drawFocusIfNeeded: vi.fn(),
  scrollPathIntoView: vi.fn(),
  ellipse: vi.fn(),
  roundRect: vi.fn(),
  getLineDash: vi.fn(() => []),
  setLineDash: vi.fn(),
  path: new Path2D(),
  filter: 'none',
  drawWidgetAsOnScreen: vi.fn(),
  drawWindow: vi.fn(),
  demote: vi.fn(),
  getTransform: vi.fn(() => new DOMMatrix()),
};

HTMLCanvasElement.prototype.getContext = vi.fn((contextId: string) => {
  if (contextId === '2d') {
    return mockContext as unknown as CanvasRenderingContext2D;
  }
  return null;
}) as unknown as HTMLCanvasElement['getContext'];

// Redirects scenes' `import Phaser from 'phaser'` to the mocked global.Phaser above.
vi.mock('phaser', () => ({
  default: (globalThis as Record<string, unknown>).Phaser,
}));
