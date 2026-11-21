#!/usr/bin/env python3
"""Extracts item drop sources from a Turtle WoW (vmangos-based) world database SQL dump.

Usage:
    python3 -I tools/database/loot/extract_loot_sources.py <tw_world.sql> <out_dir> [scripts_dir]

`scripts_dir` is optional: the server's src/scripts directory (only the
dungeons/ subtree is needed). It is used to find the instance of creatures that
have no static spawn (e.g. summoned bosses like Ragnaros), by looking at which
instance directory registers the creature's script_name.

Writes:
    <out_dir>/LootSources.csv  item_id,npc_id,object_id,name,zone_id,chance
    <out_dir>/LootZones.csv    zone_id,name

`chance` is the percent chance that a single kill/loot of the source yields at
least one copy of the item, following the server's LootMgr roll rules:
  - ungrouped entries roll independently,
  - grouped entries share one roll: explicitly chanced entries take their
    chance cumulatively, equal-chanced (0) entries split the remainder,
  - references roll their own chance and are then processed `maxcount` times,
    restricted to a single group of the referenced template when `groupid` is set.

Only the top sources per item are written. Items dropped by many creatures
(shared world-drop tables) get a single chance-less "World Drop" row, kept after
any specific source with at least a 1% chance.
"""

import csv
import glob
import os
import re
import sys
from collections import Counter, defaultdict

MAX_SOURCES_PER_ITEM = 3
WORLD_DROP_MIN_SOURCES = 20
WORLD_DROP_SPECIFIC_MIN_CHANCE = 1.0  # percent
MIN_CHANCE = 0.0001  # percent

INSERT_RE = re.compile(r"^INSERT INTO `(\w+)` VALUES \((.*)\);\s*$")

WANTED_TABLES = {
    'creature_template',
    'creature',
    'creature_loot_template',
    'reference_loot_template',
    'gameobject_template',
    'gameobject',
    'gameobject_loot_template',
    'map_template',
    'area_template',
    'creature_ai_events',
    'creature_ai_scripts',
    'gameobject_scripts',
}

SCRIPT_COMMAND_SUMMON_CREATURE = 10

GO_TYPE_CHEST = 3

# Instance creatures (creature entry -> map id) that are spawned by C++ scripts or
# other means we cannot trace from the world database.
NPC_MAP_OVERRIDES = {
    643: 36,     # Sneed - Deadmines
    3654: 43,    # Mutanus the Devourer - Wailing Caverns
    4627: 33,    # Arugal's Voidwalker - Shadowfang Keep
    4978: 48,    # Aku'mai Servant - Blackfathom Deeps
    12876: 48,   # Baron Aquanis - Blackfathom Deeps
    7361: 90,    # Grubbis - Gnomeregan
    7355: 129,   # Tuten'kash - Razorfen Downs
    7356: 129,   # Plaguemaw the Rotting - Razorfen Downs
    7273: 209,   # Gahz'rilla - Zul'Farrak
    7275: 209,   # Shadowpriest Sezz'ziz - Zul'Farrak
    7276: 209,   # Zul'Farrak Dead Hero - Zul'Farrak
    7286: 209,   # Zul'Farrak Zombie - Zul'Farrak
    7796: 209,   # Nekrum Gutchewer - Zul'Farrak
    8179: 209,   # Greater Healing Ward - Zul'Farrak
    8443: 109,   # Avatar of Hakkar - Sunken Temple
    9029: 230,   # Eviscerator - Blackrock Depths
    9030: 230,   # Ok'thor the Breaker - Blackrock Depths
    9032: 230,   # Hedrum the Creeper - Blackrock Depths
    9596: 229,   # Bannok Grimaxe - Blackrock Spire
    10264: 229,  # Solakar Flamewreath - Blackrock Spire
    10268: 229,  # Gizrul the Slavener - Blackrock Spire
    10339: 229,  # Gyth - Blackrock Spire
    10584: 229,  # Urok Doomhowl - Blackrock Spire
    10506: 289,  # Kirtonos the Herald - Scholomance
    14516: 289,  # Death Knight Darkreaver - Scholomance
    10813: 329,  # Balnazzar - Stratholme
    11664: 409,  # Flamewaker Elite - Molten Core
    14351: 429,  # Gordok Bushwacker - Dire Maul
}

SCRIPT_NAME_RE = re.compile(r'Name\s*=\s*"(\w+)"')


def norm_name(name):
    name = name.lower().replace("'", '')
    return name[4:] if name.startswith('the ') else name


def script_instances(scripts_dir):
    """Maps a registered script name to the instance script registered in the same directory."""
    res = {}
    if not scripts_dir:
        return res
    for inst_file in glob.glob(os.path.join(scripts_dir, '**', 'instance_*.cpp'), recursive=True):
        d = os.path.dirname(inst_file)
        with open(inst_file, encoding='utf-8', errors='replace') as f:
            inst_names = [n for n in SCRIPT_NAME_RE.findall(f.read()) if n.startswith('instance_')]
        if not inst_names:
            continue
        for cpp in glob.glob(os.path.join(d, '*.cpp')):
            with open(cpp, encoding='utf-8', errors='replace') as f:
                for n in SCRIPT_NAME_RE.findall(f.read()):
                    res.setdefault(n, inst_names[0])
    return res


def parse_values(s):
    """Parses the inside of a single SQL VALUES (...) tuple."""
    out = []
    i, n = 0, len(s)
    while i < n:
        c = s[i]
        if c in ' ,':
            i += 1
            continue
        if c == "'":
            i += 1
            buf = []
            while i < n:
                c = s[i]
                if c == '\\' and i + 1 < n:
                    buf.append(s[i + 1])
                    i += 2
                    continue
                if c == "'":
                    if i + 1 < n and s[i + 1] == "'":
                        buf.append("'")
                        i += 2
                        continue
                    i += 1
                    break
                buf.append(c)
                i += 1
            out.append(''.join(buf))
        else:
            j = s.find(',', i)
            if j == -1:
                j = n
            tok = s[i:j].strip()
            out.append(None if tok == 'NULL' else tok)
            i = j
    return out


def load_tables(path):
    tables = defaultdict(list)
    with open(path, encoding='utf-8', errors='replace') as f:
        for line in f:
            if not line.startswith('INSERT INTO'):
                continue
            m = INSERT_RE.match(line)
            if not m or m.group(1) not in WANTED_TABLES:
                continue
            tables[m.group(1)].append(parse_values(m.group(2)))
    return tables


class LootTemplate:
    def __init__(self):
        self.entries = []  # ungrouped items and all references: (item, chance, group, mincountOrRef, maxcount)
        self.groups = defaultdict(list)  # group id -> [(item, chance)]


def build_templates(rows):
    templates = defaultdict(LootTemplate)
    for r in rows:
        entry, item, chance, group, mincount_or_ref, maxcount = (
            int(r[0]), int(r[1]), abs(float(r[2])), int(r[3]), int(r[4]), int(r[5]))
        t = templates[entry]
        if mincount_or_ref < 0 or group == 0:
            t.entries.append((item, chance, group, mincount_or_ref, maxcount))
        else:
            t.groups[group].append((item, chance))
    return templates


def combine(a, b):
    """Probability of at least one of two independent events (fractions)."""
    return 1 - (1 - a) * (1 - b)


def group_item_chances(group):
    res = {}
    explicit = [(i, c) for i, c in group if c > 0]
    equal = [i for i, c in group if c == 0]
    remaining = 1.0
    for item, chance in explicit:
        p = min(chance / 100, remaining)
        remaining -= p
        res[item] = combine(res.get(item, 0), p)
    if equal and remaining > 0:
        p = remaining / len(equal)
        for item in equal:
            res[item] = combine(res.get(item, 0), p)
    return res


def template_item_chances(templates, refs, entry, group_id=0, memo=None, depth=0):
    """Returns {item_id: probability (fraction) of at least one drop} for one processing of a template."""
    if memo is None:
        memo = {}
    key = (id(templates), entry, group_id)
    if key in memo:
        return memo[key]
    t = templates.get(entry)
    res = {}
    if t is None or depth > 10:
        return res

    if group_id:
        res = group_item_chances(t.groups.get(group_id, []))
        memo[key] = res
        return res

    for item, chance, group, mincount_or_ref, maxcount in t.entries:
        p_roll = min(chance, 100) / 100
        if mincount_or_ref < 0:
            sub = template_item_chances(refs, refs, -mincount_or_ref, group, memo, depth + 1)
            loops = max(maxcount, 1)
            for sub_item, p in sub.items():
                p_any = 1 - (1 - p) ** loops
                res[sub_item] = combine(res.get(sub_item, 0), p_roll * p_any)
        else:
            res[item] = combine(res.get(item, 0), p_roll)

    for group in t.groups.values():
        for item, p in group_item_chances(group).items():
            res[item] = combine(res.get(item, 0), p)

    memo[key] = res
    return res


def main():
    if len(sys.argv) not in (3, 4):
        print(__doc__)
        sys.exit(1)
    sql_path, out_dir = sys.argv[1], sys.argv[2]
    scripts_dir = sys.argv[3] if len(sys.argv) == 4 else None
    tables = load_tables(sql_path)

    # Zones: instance maps resolve to their top-level area. Open-world spawns get zone 0
    # since we have no position -> area lookup.
    areas = {}
    top_areas_by_map = defaultdict(list)
    for r in tables['area_template']:
        entry, map_id, zone_id, name = int(r[0]), int(r[1]), int(r[2]), r[6]
        areas[entry] = name
        if zone_id == 0:
            top_areas_by_map[map_id].append(entry)
    map_zone = {}
    map_by_instance_script = {}
    for r in tables['map_template']:
        map_id, map_type, linked_zone, map_name, script = int(r[0]), int(r[2]), int(r[3]), r[10], r[11]
        if map_type not in (1, 2):  # dungeon, raid
            continue
        if script:
            map_by_instance_script[script] = map_id
        # linked_zone and the map's own top-level areas are both unreliable on their own
        # (e.g. Deadmines has no linked zone and its lowest area is "Westfall"), so prefer a name match.
        candidates = ([linked_zone] if linked_zone in areas else []) + sorted(top_areas_by_map[map_id])
        target = norm_name(map_name)
        zone = next((a for a in candidates if norm_name(areas[a]) == target), 0) \
            or next((a for a in candidates if norm_name(areas[a]) in target or target in norm_name(areas[a])), 0) \
            or (candidates[0] if candidates else 0)
        if zone:
            map_zone[map_id] = zone
    script_instance = script_instances(scripts_dir)

    def zone_for(map_counter):
        if not map_counter:
            return 0
        map_id = map_counter.most_common(1)[0][0]
        return map_zone.get(map_id, 0)

    creature_maps = defaultdict(Counter)
    for r in tables['creature']:
        map_id = int(r[5])
        for cid in r[1:5]:
            if cid and int(cid):
                creature_maps[int(cid)][map_id] += 1
    object_maps = defaultdict(Counter)
    object_guid_map = {}
    for r in tables['gameobject']:
        object_maps[int(r[1])][int(r[2])] += 1
        object_guid_map[int(r[0])] = int(r[2])

    # Creatures without a static spawn inherit the maps of whatever summons them:
    # EventAI scripts of spawned creatures, or scripts of spawned gameobjects (keyed by guid).
    ai_script_owner = defaultdict(set)
    for r in tables['creature_ai_events']:
        for script_id in r[11:14]:
            if script_id and int(script_id):
                ai_script_owner[int(script_id)].add(int(r[1]))
    summoned_by_creature = defaultdict(set)  # summoned entry -> summoner creature entries
    for r in tables['creature_ai_scripts']:
        if int(r[3]) == SCRIPT_COMMAND_SUMMON_CREATURE:
            summoned_by_creature[int(r[4])] |= ai_script_owner.get(int(r[0]), set())
    for r in tables['gameobject_scripts']:
        if int(r[3]) == SCRIPT_COMMAND_SUMMON_CREATURE and int(r[0]) in object_guid_map:
            summoned = int(r[4])
            if summoned not in creature_maps:
                creature_maps.setdefault(summoned, Counter())[object_guid_map[int(r[0])]] += 1
    for _ in range(3):  # follow summon chains a few levels deep
        for summoned, summoners in summoned_by_creature.items():
            if summoned in creature_maps:
                continue
            maps = Counter()
            for summoner in summoners:
                maps.update(creature_maps.get(summoner, Counter()))
            if maps:
                creature_maps[summoned] = maps

    refs = build_templates(tables['reference_loot_template'])
    creature_loot = build_templates(tables['creature_loot_template'])
    object_loot = build_templates(tables['gameobject_loot_template'])

    # item -> list of (chance%, npc_id, object_id, name, zone_id)
    sources = defaultdict(list)
    memo_c, memo_o = {}, {}

    seen_npcs = set()
    for r in tables['creature_template']:
        entry, name, loot_id, script = int(r[0]), r[6], int(r[46]), r[79]
        if not loot_id or entry in seen_npcs:
            continue
        seen_npcs.add(entry)
        chances = template_item_chances(creature_loot, refs, loot_id, memo=memo_c)
        if not chances:
            continue
        zone = zone_for(creature_maps.get(entry))
        if not zone and entry not in creature_maps and script in script_instance:
            zone = map_zone.get(map_by_instance_script.get(script_instance[script]), 0)
        for item, p in chances.items():
            if p * 100 >= MIN_CHANCE:
                sources[item].append((p * 100, entry, 0, name, zone))

    seen_objects = set()
    for r in tables['gameobject_template']:
        entry, go_type, name, loot_id = int(r[0]), int(r[1]), r[3], int(r[8])
        if go_type != GO_TYPE_CHEST or not loot_id or entry in seen_objects:
            continue
        # Only objects that actually exist in the world.
        if entry not in object_maps:
            continue
        seen_objects.add(entry)
        chances = template_item_chances(object_loot, refs, loot_id, memo=memo_o)
        zone = zone_for(object_maps.get(entry))
        for item, p in chances.items():
            if p * 100 >= MIN_CHANCE:
                sources[item].append((p * 100, 0, entry, name, zone))

    # Last resort for unspawned creatures: if the other sources of the items they drop
    # mostly share one instance zone (e.g. a summoned boss sharing a trash table), use it.
    item_zone_votes = {}
    for item, srcs in sources.items():
        votes = Counter(zone for _, npc, _, _, zone in srcs if zone)
        if votes:
            item_zone_votes[item] = votes
    npc_items = defaultdict(list)
    for item, srcs in sources.items():
        for _, npc, obj, _, zone in srcs:
            if npc and not zone and npc not in creature_maps:
                npc_items[npc].append(item)
    npc_zone_fix = {}
    for npc, items in npc_items.items():
        votes = Counter()
        for item in items:
            for zone, n in item_zone_votes.get(item, {}).items():
                votes[zone] += 1  # one vote per item, not per source
        if votes:
            zone, n = votes.most_common(1)[0]
            if n * 2 > sum(votes.values()):
                npc_zone_fix[npc] = zone
    for npc, map_id in NPC_MAP_OVERRIDES.items():
        if map_id in map_zone:
            npc_zone_fix[npc] = map_zone[map_id]
    for item, srcs in sources.items():
        sources[item] = [(c, npc, obj, name, npc_zone_fix.get(npc, zone) if npc and not zone else zone)
                         for c, npc, obj, name, zone in srcs]
    unresolved = Counter((npc, name) for srcs in sources.values() for _, npc, _, name, zone in srcs
                         if npc and not zone and npc not in creature_maps and npc not in npc_zone_fix)
    print(f'{len(unresolved)} unspawned creatures without a zone, most items first: '
          + ', '.join(f'{name} ({npc})' for (npc, name), _ in unresolved.most_common(40)), file=sys.stderr)

    used_zones = set()
    os.makedirs(out_dir, exist_ok=True)
    with open(os.path.join(out_dir, 'LootSources.csv'), 'w', newline='', encoding='utf-8') as f:
        w = csv.writer(f)
        w.writerow(['item_id', 'npc_id', 'object_id', 'name', 'zone_id', 'chance'])
        for item in sorted(sources):
            srcs = sources[item]
            # Collapse duplicates with the same name/zone (e.g. several creature entries for one boss).
            best = {}
            for chance, npc, obj, name, zone in srcs:
                k = (name, zone, bool(obj))
                if k not in best or chance > best[k][0]:
                    best[k] = (chance, npc, obj, name, zone)
            srcs = list(best.values())
            srcs.sort(key=lambda s: (-s[0], s[3]))
            world_drop = len(srcs) >= WORLD_DROP_MIN_SOURCES
            if world_drop:
                # Keep notable specific sources (e.g. world bosses), then one chance-less "World Drop" row.
                srcs = [s for s in srcs if s[0] >= WORLD_DROP_SPECIFIC_MIN_CHANCE][:MAX_SOURCES_PER_ITEM - 1]
            for chance, npc, obj, name, zone in srcs[:MAX_SOURCES_PER_ITEM]:
                if zone:
                    used_zones.add(zone)
                w.writerow([item, npc, obj, name, zone, f'{chance:.4g}'])
            if world_drop:
                w.writerow([item, 0, 0, 'World Drop', 0, 0])

    with open(os.path.join(out_dir, 'LootZones.csv'), 'w', newline='', encoding='utf-8') as f:
        w = csv.writer(f)
        w.writerow(['zone_id', 'name'])
        for zone in sorted(used_zones):
            w.writerow([zone, areas.get(zone, '')])

    print(f'{len(sources)} items with sources, {len(used_zones)} zones', file=sys.stderr)


if __name__ == '__main__':
    main()
