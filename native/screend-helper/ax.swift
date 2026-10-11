// ax.swift: accessibility (AXUIElement) queries for screend-helper.
//
// ax_query walks the frontmost app's focused window and reports actionable
// elements. Frames are global logical points, top-left origin — already the
// space robotgo consumes, so unlike Vision observations they must NOT go
// through the Go coordinate conversion.

import AppKit
import ApplicationServices
import Foundation

struct AXElement: Encodable {
    let role: String
    let subrole: String?
    let title: String
    let enabled: Bool
    let x: Double
    let y: Double
    let w: Double
    let h: Double
    let actions: [String]?
}

private let axActionableRoles: Set<String> = [
    "AXButton", "AXLink", "AXTextField", "AXTextArea", "AXCheckBox",
    "AXRadioButton", "AXPopUpButton", "AXComboBox", "AXMenuItem", "AXMenuButton",
]

// axProbeRoles are container-ish roles that sometimes carry a press action
// (custom controls, table cells). Probing action names costs one IPC per
// element, so it is limited to these instead of every node.
private let axProbeRoles: Set<String> = [
    "AXCell", "AXRow", "AXGroup", "AXImage", "AXStaticText",
]

private let axDefaultMaxElements = 2000
private let axDefaultMaxDepth = 30

private func axString(_ el: AXUIElement, _ attr: String) -> String? {
    var ref: CFTypeRef?
    guard AXUIElementCopyAttributeValue(el, attr as CFString, &ref) == .success else { return nil }
    return ref as? String
}

private func axElementAttr(_ el: AXUIElement, _ attr: String) -> AXUIElement? {
    var ref: CFTypeRef?
    guard AXUIElementCopyAttributeValue(el, attr as CFString, &ref) == .success,
          let v = ref, CFGetTypeID(v) == AXUIElementGetTypeID() else { return nil }
    return unsafeBitCast(v, to: AXUIElement.self)
}

private func axChildren(_ el: AXUIElement) -> [AXUIElement] {
    var ref: CFTypeRef?
    guard AXUIElementCopyAttributeValue(el, kAXChildrenAttribute as CFString, &ref) == .success,
          let arr = ref as? [AnyObject] else { return [] }
    return arr.compactMap {
        CFGetTypeID($0) == AXUIElementGetTypeID() ? unsafeBitCast($0, to: AXUIElement.self) : nil
    }
}

private func axFrame(_ el: AXUIElement) -> CGRect? {
    var posRef: CFTypeRef?
    var sizeRef: CFTypeRef?
    guard AXUIElementCopyAttributeValue(el, kAXPositionAttribute as CFString, &posRef) == .success,
          AXUIElementCopyAttributeValue(el, kAXSizeAttribute as CFString, &sizeRef) == .success,
          let pv = posRef, CFGetTypeID(pv) == AXValueGetTypeID(),
          let sv = sizeRef, CFGetTypeID(sv) == AXValueGetTypeID() else { return nil }
    var p = CGPoint.zero
    var s = CGSize.zero
    guard AXValueGetValue(unsafeBitCast(pv, to: AXValue.self), .cgPoint, &p),
          AXValueGetValue(unsafeBitCast(sv, to: AXValue.self), .cgSize, &s) else { return nil }
    return CGRect(origin: p, size: s)
}

private func axActions(_ el: AXUIElement) -> [String] {
    var ref: CFArray?
    guard AXUIElementCopyActionNames(el, &ref) == .success, let arr = ref as? [String] else { return [] }
    return arr
}

private func axEnabled(_ el: AXUIElement) -> Bool {
    var ref: CFTypeRef?
    guard AXUIElementCopyAttributeValue(el, kAXEnabledAttribute as CFString, &ref) == .success,
          let b = ref as? Bool else { return true }
    return b
}

// axTitle resolves the spoken name of an element: title, then description,
// then (for text-bearing roles) a short value or placeholder, then the
// linked title element. Empty means the element is real but unnameable —
// the Go side decides what to do with those.
private func axTitle(_ el: AXUIElement, role: String) -> String {
    if let t = axString(el, kAXTitleAttribute), !t.isEmpty { return t }
    if let d = axString(el, kAXDescriptionAttribute), !d.isEmpty { return d }
    if role == "AXTextField" || role == "AXTextArea" || role == "AXComboBox" {
        if let v = axString(el, kAXValueAttribute), !v.isEmpty, v.count <= 100 { return v }
        if let ph = axString(el, "AXPlaceholderValue"), !ph.isEmpty { return ph }
    }
    if let titleEl = axElementAttr(el, kAXTitleUIElementAttribute) {
        if let t = axString(titleEl, kAXTitleAttribute), !t.isEmpty { return t }
        if let v = axString(titleEl, kAXValueAttribute), !v.isEmpty { return v }
    }
    return ""
}

func axQuery(_ req: Request) -> Response {
    guard AXIsProcessTrusted() else {
        return Response(id: req.id, ok: false,
                        error: "accessibility permission not granted (AXIsProcessTrusted=false)")
    }

    let pid: pid_t
    if let p = req.pid, p > 0 {
        pid = pid_t(p)
    } else if let front = NSWorkspace.shared.frontmostApplication {
        pid = front.processIdentifier
    } else {
        return Response(id: req.id, ok: false, error: "no frontmost application")
    }

    let app = AXUIElementCreateApplication(pid)
    // an unresponsive app degrades to an error response instead of wedging
    // the helper (which the engine would kill and respawn, paying backoff)
    AXUIElementSetMessagingTimeout(app, 0.5)
    // Chromium/Electron apps publish an empty tree until this is set; on
    // everything else the attribute doesn't exist and the call is a no-op
    AXUIElementSetAttributeValue(app, "AXManualAccessibility" as CFString, kCFBooleanTrue)

    var window = axElementAttr(app, kAXFocusedWindowAttribute)
    if window == nil {
        var ref: CFTypeRef?
        if AXUIElementCopyAttributeValue(app, kAXWindowsAttribute as CFString, &ref) == .success,
           let arr = ref as? [AnyObject], let first = arr.first,
           CFGetTypeID(first) == AXUIElementGetTypeID() {
            window = unsafeBitCast(first, to: AXUIElement.self)
        }
    }
    guard let win = window else {
        return Response(id: req.id, ok: false, error: "app pid \(pid) has no focused window")
    }

    let maxElements = (req.maxElements ?? 0) > 0 ? req.maxElements! : axDefaultMaxElements
    let maxDepth = (req.maxDepth ?? 0) > 0 ? req.maxDepth! : axDefaultMaxDepth

    let t0 = Date()
    var visited = 0
    var out: [AXElement] = []

    func walk(_ el: AXUIElement, _ depth: Int) {
        if depth > maxDepth || visited >= maxElements { return }
        visited += 1
        let role = axString(el, kAXRoleAttribute) ?? ""
        var actions: [String] = []
        var actionable = axActionableRoles.contains(role)
        if !actionable, axProbeRoles.contains(role) {
            actions = axActions(el)
            actionable = actions.contains("AXPress")
        }
        if actionable, let frame = axFrame(el), frame.width >= 1, frame.height >= 1 {
            if actions.isEmpty { actions = axActions(el) }
            out.append(AXElement(
                role: role,
                subrole: axString(el, kAXSubroleAttribute),
                title: axTitle(el, role: role),
                enabled: axEnabled(el),
                x: frame.origin.x, y: frame.origin.y,
                w: frame.width, h: frame.height,
                actions: actions.isEmpty ? nil : actions))
        }
        for child in axChildren(el) {
            walk(child, depth + 1)
        }
    }
    walk(win, 0)

    // reading order: the Go ranker's phrase matching requires items supplied
    // adjacently in visual order
    out.sort { a, b in a.y != b.y ? a.y < b.y : a.x < b.x }

    let bounds = CGDisplayBounds(resolveDisplayID(req.display ?? 0)) // global points, top-left origin
    let meta = Meta(imgW: Int(bounds.width), imgH: Int(bounds.height),
                    originXPts: bounds.minX, originYPts: bounds.minY,
                    widthPts: bounds.width, heightPts: bounds.height)
    logErr(String(format: "ax_query pid=%d visited=%d elements=%d walk=%.0fms",
                  pid, visited, out.count, Date().timeIntervalSince(t0) * 1000))
    var resp = Response(id: req.id, ok: true, meta: meta)
    resp.elements = out
    return resp
}
