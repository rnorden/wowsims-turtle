# OctoWoW mechanics notes and open items

This fork models Turtle WoW as run on the OctoWoW server (octowow.st), not Blizzard vanilla 1.12.
Check combat formulas against OctoWoW / Turtle sources before relying on classic references.

## Sources

- Official Turtle 1.17.2 "Class & Gameplay Changes" notes. The Turtle forum is not reachable directly, only through
  archive.org: https://web.archive.org/web/20260527132642/https://forum.turtlecraft.gg/viewtopic.php?t=15157
- OctoWoW weapon skill guide (community, Aug 2026): https://octowow.st/forum/viewtopic.php?t=1666. The numbers it
  charts are in a linked Google Sheet (export as CSV):
  https://docs.google.com/spreadsheets/d/1b5wfBLYoopE7_4r2lhrJHx395K1AlZKx8qMeu821nW4
- OctoWoW's own "Patch Notes and Changelog" forum has no weapon skill or racial changes of its own.

## Implemented: weapon skill (Turtle 1.17.2)

`sim/core/target.go` `NewAttackTable` (+ `target_test.go`), `sim/core/specializations.go`, and the hit cap shown in the
UI (`ui/core/player.ts` `getMeleeCritCapInfo`):

| | Formula (vs. a level+3 boss, defense 315) | Source |
|---|---|---|
| Miss chance | 5% + 0.2% × (315 − skill). No 1% hit suppression, so the hit cap is the miss chance (300 → 8%, 308 → 6.4%, 315 → 5%) | official |
| Glancing damage | mean 65% + 2% × (skill − 300), 95% at 315 | official (per point) + sheet |
| Skill cap | effective skill capped at level×5 + 15 = 315 | community guide only |
| Racial weapon skill | +3 instead of +5 (Human swords/maces, Orc axes, Dwarf guns, Troll bows/thrown) | official |

Against lower-level mobs, skill above the target's defense keeps the vanilla 0.1% per point.

## Open items

### Undocumented: vanilla behaviour kept

No source covers these after 1.17.2. They need in-game combat-log testing or an answer from OctoWoW staff.

- **Glance chance** per hit vs bosses (vanilla 40%) is unchanged.
- **Glancing damage range.** Sources give only the mean. The sim rolls uniformly between a min and a max interpolated
  from the vanilla endpoints: 0.55–0.75 at 300 skill, 0.91–0.99 at 315.
- **Dodge** (0.1% per skill point), **parry** and **crit suppression** are unchanged.
- **Ranged attacks** follow the same miss formula (no suppression), and they can't glance.

### Talent weapon skill (official changes, not implemented)

- Rogue **Weapon Expertise** also gives mace skill (`sim/rogue/talents.go` only covers swords/daggers/fist).
- Rogue **Mace Specialization** was renamed **Concussive Blows** and no longer gives mace skill (the sim still adds
  it).
- Warrior **Two-Handed Weapon Specialization**: 3 points, 2/4/6% damage, +1/2/3 skill in all two-handed weapons. The
  sim has 1/2/3% damage and no skill.
- Paladin **Two-Handed Weapon Specialization**: +1/2/3 skill for two-handed swords, maces and axes (the sim has damage
  only).
- High Elf (+3 bows) and Goblin (+3 daggers/maces) racials exist on Turtle, but those races aren't in the sim.

### Stat weights / EP

- Stat weights now measure a capped stat from the side that still changes DPS, instead of giving it 0. Each metric
  also reports gain-only and loss-only EP (`ep_values_gain` / `ep_values_loss`).
- The gear picker scores items relative to the equipped item and ignores gains in capped stats. This only applies to
  EP computed from a stat-weights run; manually entered or preset EP stays linear.
- Weapon skill counts toward item EP. Only hunters weigh it so far (Bow/Crossbow/Gun skill). warrior, tank_warrior,
  rogue, retribution/protection paladin, enhancement/warden shaman and feral druid would benefit from adding their
  skill pseudo stats to `epPseudoStats`.
- Still plain linear EP: random suffix EP, the item swap picker, and the Pawn / 60U exporters.
- EP can't see some effects, measured for BM hunters with `sim/hunter/ep_compare_test.go`:
  - ranged weapon speed (shots scale with weapon damage per shot: Rhok'delar +24 DPS predicted vs +4 real)
  - on-use trinkets (Earthstrike scores 0 EP but is worth about 25 DPS)
  - set bonuses (an EP-driven re-pick broke a set for −29 DPS)
  A "sim the top N candidates" action in the gear picker would cover all three.
- EP confidence intervals now come from paired per-iteration samples. An 8-seed test showed no measurable difference
  from the old formula for hunters.
- The cap-aware scoring and the lock markers in the stat-weights table have only been type-checked, not
  clicked through in the running UI.
