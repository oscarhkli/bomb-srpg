import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../../test/setup';
import { firstGraphics, firstText, pointerDownOf } from '../../test/sceneHelpers';
import { START_MATCH_BUTTON_LABEL } from '../../constants';
import type { PageBounds } from './SettingsPage';
import StagePage from './StagePage';

beforeEach(() => {
  vi.clearAllMocks();
});

function bounds(): PageBounds {
  return { x: 48, y: 636, width: 1184, height: 108 };
}

describe('StagePage', () => {
  it('renders the "Stage Selection" title', () => {
    const page = new StagePage({ goNext: vi.fn(), goBack: vi.fn() });

    page.renderHeaderTitle(mockScene as never, 200, 102);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      200,
      102,
      'Stage Selection',
      expect.objectContaining({})
    );
  });

  it('renderBody is a stub: it renders nothing', () => {
    const page = new StagePage({ goNext: vi.fn(), goBack: vi.fn() });

    page.renderBody();

    expect(mockScene.add.graphics).not.toHaveBeenCalled();
    expect(mockScene.add.text).not.toHaveBeenCalled();
  });

  // p3-spec009-stage.md Non-Goal: entering MatchScene from here — StartMatchButton must not wire
  // a click handler yet.
  it('renders StartMatchButton with the right label and no click handler (Non-Goal: entering MatchScene)', () => {
    const page = new StagePage({ goNext: vi.fn(), goBack: vi.fn() });

    page.renderNav(mockScene as never, bounds());

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      START_MATCH_BUTTON_LABEL,
      expect.objectContaining({})
    );
    expect(pointerDownOf(firstGraphics())).toBeUndefined();
  });

  it('handleBack navigates back to UnitPage 2 (AC 15)', () => {
    const goBack = vi.fn();
    const page = new StagePage({ goNext: vi.fn(), goBack });

    page.handleBack();

    expect(goBack).toHaveBeenCalled();
  });

  it('destroy destroys the rendered header and nav objects', () => {
    const page = new StagePage({ goNext: vi.fn(), goBack: vi.fn() });
    page.renderHeaderTitle(mockScene as never, 200, 102);
    page.renderNav(mockScene as never, bounds());
    const title = firstText();
    const navGraphics = firstGraphics();

    page.destroy();

    expect(title.destroy).toHaveBeenCalled();
    expect(navGraphics.destroy).toHaveBeenCalled();
  });

  it('renderNav tears down a previously rendered StartMatchButton before redrawing', () => {
    const page = new StagePage({ goNext: vi.fn(), goBack: vi.fn() });
    page.renderNav(mockScene as never, bounds());
    const firstButtonGraphics = firstGraphics();

    page.renderNav(mockScene as never, bounds());

    expect(firstButtonGraphics.destroy).toHaveBeenCalled();
  });
});
