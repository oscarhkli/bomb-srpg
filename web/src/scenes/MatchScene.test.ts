import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mockGraphics, mockScene, mockText } from '../test/setup'
import { initRoom, getMatchState } from '../engine/api'
import { TERRAIN_COLORS, TERRAIN_BORDER_COLOR } from '../constants'
import MatchScene from './MatchScene'
import type { GameState, Tile, TerrainType } from '../types/api'

vi.mock('../engine/api')

function makeState(grid: Tile[][]): GameState {
  return { turn: 1, activeTeam: 0, grid, units: [], bombs: [], softBlocks: [], turnCommands: [] }
}

function plainTile(): Tile {
  return tileOf('TerrainPlain')
}

function tileOf(type: TerrainType): Tile {
  return { type, occupantType: 'OccupantNone', occupantId: 0 }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('MatchScene', () => {
  it('calls initRoom with data.roomId then getMatchState on create', async () => {
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]]))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    expect(initRoom).toHaveBeenCalledWith('room-abc')
    expect(getMatchState).toHaveBeenCalledOnce()
  })

  it('renders a 3x2 grid as 6 tiles at correct world positions', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()]
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    expect(mockGraphics.fillRect).toHaveBeenCalledTimes(6)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(1, 0, 0, 48, 48)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(2, 48, 0, 48, 48)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(3, 96, 0, 48, 48)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(4, 0, 48, 48, 48)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(5, 48, 48, 48, 48)
    expect(mockGraphics.fillRect).toHaveBeenNthCalledWith(6, 96, 48, 48, 48)
  })

  it('fills each terrain type with its TERRAIN_COLORS value', async () => {
    const types: TerrainType[] = [
      'TerrainPlain',
      'TerrainBlock',
      'TerrainTower',
      'TerrainWater',
      'TerrainLava'
    ]
    vi.mocked(getMatchState).mockResolvedValue(makeState([types.map(tileOf)]))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    types.forEach((type, i) => {
      expect(mockGraphics.fillStyle).toHaveBeenNthCalledWith(i + 1, TERRAIN_COLORS[type])
    })
  })

  it('draws a 1px black border around every tile', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()]
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    expect(mockGraphics.lineStyle).toHaveBeenCalledWith(1, TERRAIN_BORDER_COLOR)
    expect(mockGraphics.strokeRect).toHaveBeenCalledTimes(6)
    expect(mockGraphics.strokeRect).toHaveBeenNthCalledWith(1, 0, 0, 48, 48)
    expect(mockGraphics.strokeRect).toHaveBeenNthCalledWith(6, 96, 48, 48, 48)
  })

  it('centers camera on a 3x2 grid', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()]
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(72, 48)
  })

  it('centers camera on a differently-sized 5x7 grid', async () => {
    const row = (): Tile[] => Array.from({ length: 5 }, plainTile)
    vi.mocked(getMatchState).mockResolvedValue(makeState(Array.from({ length: 7 }, row)))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(120, 168)
  })

  it('shows an error message when getMatchState rejects, without rendering', async () => {
    vi.mocked(getMatchState).mockRejectedValue(new Error('network fail'))

    const scene = new MatchScene()
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] })
    await Promise.resolve()
    await Promise.resolve()

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to load match state'
    )
    expect(mockText.setOrigin).toHaveBeenCalledWith(0.5)
    expect(mockGraphics.fillRect).not.toHaveBeenCalled()
    expect(mockScene.cameras.main.centerOn).not.toHaveBeenCalled()
  })
})
