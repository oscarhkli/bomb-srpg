export interface SpriteManifestEntry {
  key: string;
  png: string;
  json: string;
}

export const SPRITE_ASSET_BASE = 'assets/sprites/';

export const UNIT_SPRITE_MANIFEST: SpriteManifestEntry[] = [
  { key: 'unit_fighter_blue', png: 'units/Fighter-Blue.png', json: 'units/Fighter-Blue.json' },
  { key: 'unit_fighter_red', png: 'units/Fighter-Red.png', json: 'units/Fighter-Red.json' },
  { key: 'unit_king_blue', png: 'units/King-Blue.png', json: 'units/King-Blue.json' },
  { key: 'unit_king_red', png: 'units/King-Red.png', json: 'units/King-Red.json' },
  { key: 'unit_bandit_blue', png: 'units/Bandit-Blue.png', json: 'units/Bandit-Blue.json' },
  { key: 'unit_bandit_red', png: 'units/Bandit-Red.png', json: 'units/Bandit-Red.json' },
  { key: 'unit_witch_blue', png: 'units/Witch-Blue.png', json: 'units/Witch-Blue.json' },
  { key: 'unit_witch_red', png: 'units/Witch-Red.png', json: 'units/Witch-Red.json' },
  { key: 'unit_prologue_red', png: 'units/Prologue-Red.png', json: 'units/Prologue-Red.json' },
];

export const BUTTON_SPRITE_MANIFEST: SpriteManifestEntry[] = [
  { key: 'button_neutral', png: 'buttons/Button-Neutral.png', json: 'buttons/Button-Neutral.json' },
  {
    key: 'button_selected',
    png: 'buttons/Button-Selected.png',
    json: 'buttons/Button-Selected.json',
  },
  { key: 'button_clicked', png: 'buttons/Button-Clicked.png', json: 'buttons/Button-Clicked.json' },
  { key: 'button_label_move', png: 'buttons/Button-Move.png', json: 'buttons/Button-Move.json' },
  { key: 'button_label_bomb', png: 'buttons/Button-Bomb.png', json: 'buttons/Button-Bomb.json' },
  { key: 'button_label_back', png: 'buttons/Button-Back.png', json: 'buttons/Button-Back.json' },
  { key: 'button_label_yes', png: 'buttons/Button-Yes.png', json: 'buttons/Button-Yes.json' },
  { key: 'button_label_no', png: 'buttons/Button-No.png', json: 'buttons/Button-No.json' },
];

export const SPRITE_MANIFEST: SpriteManifestEntry[] = [
  ...UNIT_SPRITE_MANIFEST,
  ...BUTTON_SPRITE_MANIFEST,
  { key: 'bomb', png: 'Bomb.png', json: 'Bomb.json' },
  { key: 'soft_block', png: 'SoftBlock.png', json: 'SoftBlock.json' },
  { key: 'stage_plain', png: 'stages/Stage-Plain.png', json: 'stages/Stage-Plain.json' },
  { key: 'stage_standard', png: 'stages/Stage-Standard.png', json: 'stages/Stage-Standard.json' },
  { key: 'stage_divided', png: 'stages/Stage-Divided.png', json: 'stages/Stage-Divided.json' },
];
