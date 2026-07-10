import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { ERROR_PANEL_PADDING } from '../constants';
import ErrorPanel from './ErrorPanel';

beforeEach(() => {
  vi.clearAllMocks();
});

function textCalls(): [number, number, string, ...unknown[]][] {
  return mockScene.add.text.mock.calls as unknown as [number, number, string, ...unknown[]][];
}

function textResult(index: number): ReturnType<typeof mockScene.add.text> {
  return mockScene.add.text.mock.results[index]!.value as ReturnType<typeof mockScene.add.text>;
}

describe('ErrorPanel', () => {
  it('lazily creates the background once and adds the message text on show', () => {
    const panel = new ErrorPanel(mockScene as never);

    panel.show('boom');

    expect(mockScene.add.graphics).toHaveBeenCalledOnce();
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'boom',
      expect.objectContaining({})
    );
  });

  it('reuses the single background graphics across multiple shows', () => {
    const panel = new ErrorPanel(mockScene as never);

    panel.show('first');
    panel.show('second');

    expect(mockScene.add.graphics).toHaveBeenCalledOnce();
  });

  it('stacks each message below the previous by its actual rendered height plus padding', () => {
    const panel = new ErrorPanel(mockScene as never);

    panel.show('first');
    panel.show('second');

    const firstY = textCalls().find(c => c[2] === 'first')![1];
    const secondY = textCalls().find(c => c[2] === 'second')![1];
    // createMockText() reports height 16 — the gap is that height plus panel padding.
    expect(secondY - firstY).toBe(textResult(0).height + ERROR_PANEL_PADDING);
  });

  it('destroys all texts and the background on clear, and recreates the background on the next show', () => {
    const panel = new ErrorPanel(mockScene as never);

    panel.show('first');
    const firstText = textResult(0);
    const bg = mockScene.add.graphics.mock.results[0]!.value as ReturnType<
      typeof mockScene.add.graphics
    >;

    panel.clear();

    expect(firstText.destroy).toHaveBeenCalled();
    expect(bg.destroy).toHaveBeenCalled();

    panel.show('again');
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(2); // fresh bg after clear
  });

  it('resets the vertical offset after clear so the next message starts back at the top', () => {
    const panel = new ErrorPanel(mockScene as never);

    panel.show('first');
    const firstY = textCalls().find(c => c[2] === 'first')![1];
    panel.show('second'); // pushes the offset down
    panel.clear();
    panel.show('third');

    const thirdY = textCalls().find(c => c[2] === 'third')![1];
    expect(thirdY).toBe(firstY);
  });
});
