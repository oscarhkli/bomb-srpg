import type Phaser from 'phaser';

const resolvedFrameCache = new Map<string, string>();

// Aseprite atlases auto-register a '__BASE' frame (the raw untrimmed source). Sprite creation
// must reference the atlas's real frame name instead, since its export filename isn't
// derivable from the texture key (e.g. "Fighter #Blue.aseprite").
export function firstNonBaseFrame(scene: Phaser.Scene, textureKey: string): string {
  const cached = resolvedFrameCache.get(textureKey);
  if (cached) {
    return cached;
  }
  const frames = scene.textures.get(textureKey).frames;
  const frameName = Object.keys(frames).find(name => name !== '__BASE');
  if (!frameName) {
    throw new Error(`Texture "${textureKey}" has no real frame besides '__BASE'`);
  }
  resolvedFrameCache.set(textureKey, frameName);
  return frameName;
}
