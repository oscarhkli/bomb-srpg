import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { TEAM_COLORS, TURN_PANEL_SUDDEN_DEATH_COLOR } from '../constants';
import TurnPanel from './TurnPanel';

beforeEach(() => {
  vi.clearAllMocks();
});

function headerGraphics(): ReturnType<typeof mockScene.add.graphics> {
  return mockScene.add.graphics.mock.results[0]!.value as ReturnType<typeof mockScene.add.graphics>;
}

describe('TurnPanel', () => {
  it('fills the header with TEAM_COLORS[activeTeam]', () => {
    const panel = new TurnPanel(mockScene as never);

    panel.update(2, 30, 1);

    expect(headerGraphics().fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
  });

  it('re-fills the header with the new team color when activeTeam changes', () => {
    const panel = new TurnPanel(mockScene as never);
    panel.update(2, 30, 1);
    vi.clearAllMocks();

    panel.update(3, 30, 2);

    expect(headerGraphics().fillStyle).toHaveBeenCalledWith(TEAM_COLORS[2]);
  });

  it('renders the turn number in the sudden-death color when turn exceeds maxTurns', () => {
    const panel = new TurnPanel(mockScene as never);

    panel.update(31, 30, 1);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '31',
      expect.objectContaining({
        color: `#${TURN_PANEL_SUDDEN_DEATH_COLOR.toString(16).padStart(6, '0')}`,
      })
    );
  });

  it('pins every panel element to the camera viewport (scrollFactor 0) so it stays in place regardless of camera scroll', () => {
    const panel = new TurnPanel(mockScene as never);

    panel.update(2, 30, 1);

    expect(headerGraphics().setScrollFactor).toHaveBeenCalledWith(0);
    mockScene.add.text.mock.results.forEach(r => {
      expect(
        (r.value as ReturnType<typeof mockScene.add.text>).setScrollFactor
      ).toHaveBeenCalledWith(0);
    });
  });
});
