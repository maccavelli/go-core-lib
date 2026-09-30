#!/bin/sh
# Verify a staged self-update release directory against the canonical
# product/platform/extra matrix. Performs no publication.
set -eu

usage() {
	echo "usage: verify-selfupdate-release.sh --dir DIR --products JSON --platforms JSON --extras JSON [--tag TAG]" >&2
	exit 2
}

DIR=""
PRODUCTS_JSON=""
PLATFORMS_JSON=""
EXTRAS_JSON="[]"
TAG=""

while [ $# -gt 0 ]; do
	case "$1" in
	--dir)
		DIR="${2:-}"
		shift 2
		;;
	--products)
		PRODUCTS_JSON="${2:-}"
		shift 2
		;;
	--platforms)
		PLATFORMS_JSON="${2:-}"
		shift 2
		;;
	--extras)
		EXTRAS_JSON="${2:-}"
		shift 2
		;;
	--tag)
		TAG="${2:-}"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

if [ -z "$DIR" ] || [ -z "$PRODUCTS_JSON" ] || [ -z "$PLATFORMS_JSON" ]; then
	usage
fi
[ -d "$DIR" ] || {
	echo "verify-selfupdate-release: staging directory $DIR is missing" >&2
	exit 1
}

python3 - "$DIR" "$PRODUCTS_JSON" "$PLATFORMS_JSON" "$EXTRAS_JSON" "$TAG" <<'PY'
import hashlib, json, os, re, stat, sys, unicodedata

dirpath, products_raw, platforms_raw, extras_raw, tag = sys.argv[1:6]
tag_re = re.compile(r"^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")
hex_re = re.compile(r"^[0-9a-fA-F]{64}$")
os_arch_re = re.compile(r"^[a-z0-9][a-z0-9_]*$")
product_re = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")

def fail(msg):
    print("verify-selfupdate-release: " + msg, file=sys.stderr)
    sys.exit(1)

try:
    products = json.loads(products_raw)
    platforms = json.loads(platforms_raw)
    extras = json.loads(extras_raw)
except json.JSONDecodeError as e:
    fail("invalid JSON: %s" % e)

if not isinstance(products, list) or not products:
    fail("products-json must be a non-empty JSON array")
if not isinstance(platforms, list) or not platforms:
    fail("platforms-json must be a non-empty JSON array")
if not isinstance(extras, list):
    fail("extra-assets-json must be a JSON array")

seen_products = set()
for p in products:
    if not isinstance(p, str) or not product_re.fullmatch(p):
        fail("invalid product %r" % p)
    if p in seen_products:
        fail("duplicate product %s" % p)
    seen_products.add(p)

seen_plats = set()
for plat in platforms:
    if not isinstance(plat, dict) or set(plat.keys()) != {"os", "arch"}:
        fail("platform objects must have only os and arch")
    osname, arch = plat["os"], plat["arch"]
    if not isinstance(osname, str) or not os_arch_re.fullmatch(osname):
        fail("invalid platform os %r" % osname)
    if not isinstance(arch, str) or not os_arch_re.fullmatch(arch):
        fail("invalid platform arch %r" % arch)
    key = (osname, arch)
    if key in seen_plats:
        fail("duplicate platform %s/%s" % key)
    seen_plats.add(key)

seen_extras = set()
for extra in extras:
    # The product-name character class: never a path, never a shell glob or
    # word separator, so the upload step can pass names safely
    # (0003-MADR D5).
    if not isinstance(extra, str) or not product_re.fullmatch(extra):
        fail("invalid extra asset %r" % extra)
    if extra in seen_extras or extra == "SHA256SUMS" or extra.startswith("SHA256SUMS-"):
        fail("invalid or duplicate extra asset %r" % extra)
    seen_extras.add(extra)

def asset_name(product, osname, arch):
    name = "%s-%s-%s" % (product, osname, arch)
    if osname == "windows":
        name += ".exe"
    return name

canonical = []
for product in products:
    for osname, arch in seen_plats:
        canonical.append(asset_name(product, osname, arch))

expected = set(canonical)
expected.add("SHA256SUMS")
expected.update(seen_extras)

if tag and not tag_re.fullmatch(tag):
    fail("tag %r is not a strict stable tag" % tag)

present = []
for name in os.listdir(dirpath):
    path = os.path.join(dirpath, name)
    mode = os.lstat(path).st_mode
    if stat.S_ISDIR(mode):
        fail("unexpected directory %s in staging" % name)
    if not stat.S_ISREG(mode):
        fail("staged entry %s is not a regular file (0003-MADR D6)" % name)
    present.append(name)

present_set = set(present)
if present_set != expected:
    missing = sorted(expected - present_set)
    extra = sorted(present_set - expected)
    fail("file set mismatch missing=%s extra=%s" % (missing, extra))

MAX_CHECKSUM_LINE = 4096  # selfupdate/checksums.go maxChecksumLine


def go_isspace(c):
    # Go's unicode.IsSpace, which differs from str.isspace (that also
    # counts U+001C..U+001F).
    return c in "\t\n\v\f\r \x85\xa0" or unicodedata.category(c) in ("Zs", "Zl", "Zp")


def go_trim_space(s):
    start, end = 0, len(s)
    while start < end and go_isspace(s[start]):
        start += 1
    while end > start and go_isspace(s[end - 1]):
        end -= 1
    return s[start:end]


def go_fields(s):
    fields, cur = [], []
    for c in s:
        if go_isspace(c):
            if cur:
                fields.append("".join(cur))
                cur = []
        else:
            cur.append(c)
    if cur:
        fields.append("".join(cur))
    return fields


def parse_sums(path):
    """Mirror parseSHA256SUMS in selfupdate/checksums.go (0003-MADR D1)."""
    base = os.path.basename(path)
    with open(path, "rb") as f:
        raw = f.read()
    # bufio.ScanLines splits on \n only; a final empty segment is no line.
    segments = raw.split(b"\n")
    if segments and segments[-1] == b"":
        segments.pop()
    entries = {}
    for i, seg in enumerate(segments, 1):
        # The scanner's 4096-byte buffer must hold the line and its newline.
        if len(seg) >= MAX_CHECKSUM_LINE:
            fail("%s line %d: longer than the client accepts" % (base, i))
        line = seg.decode("utf-8", errors="surrogateescape").rstrip("\r")
        trimmed = go_trim_space(line)
        if trimmed == "" or trimmed.startswith("#"):
            continue
        fields = go_fields(line)
        if len(fields) != 2:
            fail("%s line %d: want exactly two fields" % (base, i))
        digest, name = fields
        if name.startswith("*"):
            name = name[1:]
        if name == "" or "*" in name:
            fail("%s line %d: malformed filename" % (base, i))
        if not hex_re.fullmatch(digest) or not digest.isascii():
            fail("%s line %d: malformed digest" % (base, i))
        if name in (".", "..") or "/" in name or "\\" in name or os.path.basename(name) != name:
            fail("%s line %d: filename is not a basename" % (base, i))
        if name in entries:
            fail("%s duplicate filename %s" % (base, name))
        entries[name] = digest.lower()
    if not entries:
        fail("%s has no entries" % base)
    return entries

sums = parse_sums(os.path.join(dirpath, "SHA256SUMS"))
if set(sums) != set(canonical):
    fail("SHA256SUMS must contain exactly the canonical binaries")

def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

for name in canonical:
    path = os.path.join(dirpath, name)
    got = sha256_file(path)
    if got != sums[name]:
        fail("SHA256SUMS mismatch for %s" % name)

print("verify-selfupdate-release: ok")
PY
