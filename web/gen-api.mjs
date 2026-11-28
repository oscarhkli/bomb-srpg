import fs from 'node:fs';
import ts from 'typescript';
import openapiTS, { astToString, COMMENT_HEADER } from 'openapi-typescript';

const FIELD_BRAND = {
  unitId: 'UnitID',
  ownerId: 'UnitID',
  bombId: 'BombID',
  softBlockId: 'SoftBlockID',
};
const SCHEMA_ID_BRAND = { Unit: 'UnitID', Bomb: 'BombID', SoftBlock: 'SoftBlockID' };
const PLAIN_NUMBER_IDS = [
  'Tile/occupantId',
  'SurrenderRequest/teamId',
  'MatchEndedEvent/winnerTeamId',
];

const f = ts.factory;
const brand = name =>
  f.createIntersectionTypeNode([
    f.createKeywordTypeNode(ts.SyntaxKind.NumberKeyword),
    f.createTypeLiteralNode([
      f.createPropertySignature(
        [f.createModifier(ts.SyntaxKind.ReadonlyKeyword)],
        '__brand',
        undefined,
        f.createLiteralTypeNode(f.createStringLiteral(name))
      ),
    ]),
  ]);

const problems = [];
let branded = 0;
const ast = await openapiTS(new URL('../server/api/openapi.yaml', import.meta.url), {
  transform(schema, { path = '' }) {
    const leaf = path.split('/').at(-1);
    if (schema.type !== 'integer' || !/^id$|Id$/.test(leaf)) return undefined;
    const owner = path.split('/').at(-2);
    const name = FIELD_BRAND[leaf] ?? (leaf === 'id' ? SCHEMA_ID_BRAND[owner] : undefined);
    if (name) {
      branded++;
      return brand(name);
    }
    if (!PLAIN_NUMBER_IDS.includes(`${owner}/${leaf}`)) {
      problems.push(
        `${path}: integer ID field has no brand; add it to FIELD_BRAND or SCHEMA_ID_BRAND, or to PLAIN_NUMBER_IDS, in web/gen-api.mjs`
      );
    }
    return undefined;
  },
});
if (problems.length > 0 || branded === 0) {
  console.error(
    problems.join('\n') ||
      'no ID field was branded: the openapi-typescript transform hook did not run'
  );
  process.exit(1);
}
fs.writeFileSync(
  new URL('./src/api/schema.gen.ts', import.meta.url),
  COMMENT_HEADER + astToString(ast)
);
