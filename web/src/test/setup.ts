// Mock Path2D globally for jsdom (not implemented by jsdom)
;(globalThis as Record<string, unknown>).Path2D ??= class MockPath2D {
  addPath(): void { /* stub */ }
}

// Mock Phaser globals for unit tests (no real canvas/WebGL)
const mockGameObjectFactory = {
  sprite: vi.fn(),
  graphics: vi.fn(),
  text: vi.fn(),
  container: vi.fn(),
  renderTexture: vi.fn(),
  tileSprite: vi.fn(),
  bitmapText: vi.fn()
}

const mockScene = {
  add: mockGameObjectFactory,
  make: mockGameObjectFactory,
  cameras: { main: { width: 1280, height: 720 } },
  scale: { width: 1280, height: 720 },
  sys: { game: { config: { width: 1280, height: 720 } } },
  load: {
    image: vi.fn(),
    spritesheet: vi.fn(),
    atlas: vi.fn(),
    audio: vi.fn(),
    json: vi.fn(),
    on: vi.fn(),
    once: vi.fn()
  },
  tweens: { add: vi.fn() },
  time: { addEvent: vi.fn() },
  events: { on: vi.fn(), off: vi.fn(), emit: vi.fn() },
  input: {
    on: vi.fn(),
    off: vi.fn(),
    keyboard: { addKey: vi.fn(), createCursorKeys: vi.fn(() => ({})) }
  },
  children: { getAll: vi.fn(() => []) },
  game: { config: { width: 1280, height: 720 } }
}

// @ts-expect-error - global mock for tests
// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
global.Phaser = {
  Game: vi.fn(),
  Scene: class { constructor() { Object.assign(this, mockScene) } },
  AUTO: 0,
  CANVAS: 1,
  WEBGL: 2,
  Scale: { FIT: 3, CENTER_BOTH: 4 },
  Physics: { Arcade: {} },
  Tilemaps: {},
  Math: {
    Between: (min: number, max: number) => Math.floor(Math.random() * (max - min + 1)) + min,
    Clamp: (v: number, min: number, max: number) => Math.max(min, Math.min(max, v)),
    Linear: (a: number, b: number, t: number) => a + (b - a) * t
  },
  Geom: {
    Rectangle: class { constructor(public x: number, public y: number, public width: number, public height: number) {} },
    Circle: class { constructor(public x: number, public y: number, public radius: number) {} },
    Point: class { constructor(public x: number, public y: number) {} }
  },
  Display: { Color: { ValueToColor: vi.fn() } }
}

// Mock canvas for jsdom
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
  canvas: { width: 1280, height: 720 },
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
  getTransform: vi.fn(() => new DOMMatrix())
}

HTMLCanvasElement.prototype.getContext = vi.fn((contextId: string) => {
  if (contextId === '2d') return mockContext as unknown as CanvasRenderingContext2D
  return null
}) as unknown as HTMLCanvasElement['getContext']
