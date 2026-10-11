// space-helper: reports the Mission Control space layout for the main display.
//
// One-shot: prints a single JSON object to stdout and exits. All errors go
// to stderr with a non-zero exit. Uses the private SkyLight framework
// (CGSCopyManagedDisplaySpaces) — read-only, the same enumeration calls
// WhichSpace and yabai rely on. Space IDs are stable while spaces exist;
// array order is Mission Control order (what Ctrl+Left/Right walks).
//
// Single-display only by design: reads the first managed display.

import Foundation

func fail(_ msg: String) -> Never {
    FileHandle.standardError.write(("space-helper: " + msg + "\n").data(using: .utf8)!)
    exit(1)
}

guard let skylight = dlopen("/System/Library/PrivateFrameworks/SkyLight.framework/SkyLight", RTLD_NOW) else {
    fail("failed to load SkyLight framework")
}

typealias MainConnectionFn = @convention(c) () -> Int32
typealias CopySpacesFn = @convention(c) (Int32) -> Unmanaged<CFArray>?

guard let mainConnSym = dlsym(skylight, "CGSMainConnectionID"),
      let copySpacesSym = dlsym(skylight, "CGSCopyManagedDisplaySpaces")
else {
    fail("missing SkyLight symbols")
}

let mainConnection = unsafeBitCast(mainConnSym, to: MainConnectionFn.self)
let copyManagedDisplaySpaces = unsafeBitCast(copySpacesSym, to: CopySpacesFn.self)

guard let displaysRef = copyManagedDisplaySpaces(mainConnection()) else {
    fail("CGSCopyManagedDisplaySpaces returned nil")
}
guard let displays = displaysRef.takeRetainedValue() as? [[String: Any]],
      let display = displays.first
else {
    fail("no managed displays")
}

func spaceID(_ dict: [String: Any]) -> UInt64? {
    if let n = dict["ManagedSpaceID"] as? NSNumber { return n.uint64Value }
    if let n = dict["id64"] as? NSNumber { return n.uint64Value }
    return nil
}

struct SpaceOut: Encodable {
    let id: UInt64
    let type: Int
}

struct Out: Encodable {
    let activeID: UInt64
    let spaces: [SpaceOut]

    enum CodingKeys: String, CodingKey {
        case activeID = "active_id"
        case spaces
    }
}

guard let current = display["Current Space"] as? [String: Any],
      let activeID = spaceID(current)
else {
    fail("missing Current Space")
}
guard let rawSpaces = display["Spaces"] as? [[String: Any]], !rawSpaces.isEmpty else {
    fail("missing Spaces list")
}

var spaces: [SpaceOut] = []
for s in rawSpaces {
    guard let id = spaceID(s) else { continue }
    spaces.append(SpaceOut(id: id, type: (s["type"] as? NSNumber)?.intValue ?? 0))
}

guard let data = try? JSONEncoder().encode(Out(activeID: activeID, spaces: spaces)),
      let json = String(data: data, encoding: .utf8)
else {
    fail("failed to encode output")
}
print(json)
