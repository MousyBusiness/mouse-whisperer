// screend-helper: native capture + OCR sidecar for screend.
//
// Speaks NDJSON over stdin/stdout — one request line in, one response line
// out, correlated by id. stdout is protocol-only; all logging goes to stderr.
// It reports raw facts (Vision-normalized bottom-left boxes + capture
// metadata); all coordinate conversion happens in Go.

import AppKit
import CoreGraphics
import Foundation
import ScreenCaptureKit
import Vision

struct HelperError: Error, CustomStringConvertible {
    let description: String
    init(_ description: String) { self.description = description }
}

struct Request: Decodable {
    let id: Int64
    let op: String
    let display: Int?
    let level: String?
    let includeImage: Bool?
    let path: String?
    let pid: Int?         // ax_query only; nil/0 = frontmost app
    let maxElements: Int? // ax_query walk cap
    let maxDepth: Int?    // ax_query depth cap

    enum CodingKeys: String, CodingKey {
        case id, op, display, level, path, pid
        case includeImage = "include_image"
        case maxElements = "max_elements"
        case maxDepth = "max_depth"
    }
}

struct Meta: Encodable {
    let imgW: Int
    let imgH: Int
    let originXPts: Double
    let originYPts: Double
    let widthPts: Double
    let heightPts: Double

    enum CodingKeys: String, CodingKey {
        case imgW = "img_w"
        case imgH = "img_h"
        case originXPts = "origin_x_pts"
        case originYPts = "origin_y_pts"
        case widthPts = "width_pts"
        case heightPts = "height_pts"
    }
}

struct Box: Encodable {
    let text: String
    let line: Int
    let lineText: String
    let conf: Double
    let x: Double
    let y: Double
    let w: Double
    let h: Double

    enum CodingKeys: String, CodingKey {
        case text, line, conf, x, y, w, h
        case lineText = "line_text"
    }
}

struct Response: Encodable {
    let id: Int64
    let ok: Bool
    var error: String? = nil
    var meta: Meta? = nil
    var observations: [Box]? = nil
    var elements: [AXElement]? = nil // ax_query only
    var axTrusted: Bool? = nil       // ping only
    var pngBase64: String? = nil

    enum CodingKeys: String, CodingKey {
        case id, ok, error, meta, observations, elements
        case axTrusted = "ax_trusted"
        case pngBase64 = "png_base64"
    }
}

func logErr(_ msg: String) {
    FileHandle.standardError.write(("screend-helper: " + msg + "\n").data(using: .utf8)!)
}

func emit(_ resp: Response) {
    guard var data = try? JSONEncoder().encode(resp) else {
        logErr("failed to encode response id \(resp.id)")
        return
    }
    data.append(0x0A)
    FileHandle.standardOutput.write(data)
}

func resolveDisplayID(_ index: Int) -> CGDirectDisplayID {
    var ids = [CGDirectDisplayID](repeating: 0, count: 16)
    var count: UInt32 = 0
    CGGetActiveDisplayList(UInt32(ids.count), &ids, &count)
    if index >= 0 && index < Int(count) {
        return ids[index]
    }
    return CGMainDisplayID()
}

func displayScale(_ displayID: CGDirectDisplayID) -> CGFloat {
    guard let mode = CGDisplayCopyDisplayMode(displayID), mode.width > 0 else { return 2.0 }
    return CGFloat(mode.pixelWidth) / CGFloat(mode.width)
}

// captureDisplay bridges ScreenCaptureKit's async capture into the
// synchronous request loop. SCScreenshotManager fails loudly when Screen
// Recording permission is missing, which we want surfaced as an error.
func captureDisplay(_ displayID: CGDirectDisplayID) throws -> CGImage {
    let sem = DispatchSemaphore(value: 0)
    var out: Result<CGImage, Error> = .failure(HelperError("capture did not complete"))
    Task {
        do {
            let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)
            guard let display = content.displays.first(where: { $0.displayID == displayID }) else {
                throw HelperError("display \(displayID) not available for capture")
            }
            let filter = SCContentFilter(display: display, excludingWindows: [])
            let config = SCStreamConfiguration()
            let scale = displayScale(displayID)
            config.width = Int(CGFloat(display.width) * scale)
            config.height = Int(CGFloat(display.height) * scale)
            config.showsCursor = false
            out = .success(try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config))
        } catch {
            out = .failure(error)
        }
        sem.signal()
    }
    sem.wait()
    return try out.get()
}

// whitespaceTokens splits on whitespace only — linguistic tokenizers would
// break screen tokens like "handler_test.go" apart.
func whitespaceTokens(_ s: String) -> [(String, Range<String.Index>)] {
    var tokens: [(String, Range<String.Index>)] = []
    var idx = s.startIndex
    while idx < s.endIndex {
        while idx < s.endIndex, s[idx].isWhitespace { idx = s.index(after: idx) }
        guard idx < s.endIndex else { break }
        let start = idx
        while idx < s.endIndex, !s[idx].isWhitespace { idx = s.index(after: idx) }
        tokens.append((String(s[start..<idx]), start..<idx))
    }
    return tokens
}

// proportionalSlice estimates a word box when Vision cannot supply one,
// assuming uniform glyph width across the recognized line.
func proportionalSlice(_ lineBox: CGRect, _ s: String, _ range: Range<String.Index>) -> CGRect {
    let total = max(1, s.count)
    let start = s.distance(from: s.startIndex, to: range.lowerBound)
    let len = s.distance(from: range.lowerBound, to: range.upperBound)
    let x = lineBox.minX + lineBox.width * CGFloat(start) / CGFloat(total)
    let w = lineBox.width * CGFloat(len) / CGFloat(total)
    return CGRect(x: x, y: lineBox.minY, width: w, height: lineBox.height)
}

func recognize(_ image: CGImage, level: String) throws -> [Box] {
    let request = VNRecognizeTextRequest()
    request.recognitionLevel = (level == "fast") ? .fast : .accurate
    // screen tokens like "ocr.go" must not get spell-corrected
    request.usesLanguageCorrection = false
    request.recognitionLanguages = ["en-US"]

    try VNImageRequestHandler(cgImage: image, options: [:]).perform([request])

    var boxes: [Box] = []
    for (lineIndex, observation) in (request.results ?? []).enumerated() {
        guard let candidate = observation.topCandidates(1).first else { continue }
        let lineText = candidate.string
        for (word, range) in whitespaceTokens(lineText) {
            let rect: CGRect
            if let boxObs = try? candidate.boundingBox(for: range) {
                rect = boxObs.boundingBox
            } else {
                rect = proportionalSlice(observation.boundingBox, lineText, range)
            }
            boxes.append(Box(text: word, line: lineIndex, lineText: lineText,
                             conf: Double(candidate.confidence),
                             x: rect.minX, y: rect.minY, w: rect.width, h: rect.height))
        }
    }
    return boxes
}

// pngAtPointSize renders the capture downscaled to logical point size so the
// Go side never rescales and overlay coordinates line up 1:1 with the image.
func pngAtPointSize(_ image: CGImage, widthPts: Int, heightPts: Int) -> String? {
    guard let ctx = CGContext(data: nil, width: widthPts, height: heightPts,
                              bitsPerComponent: 8, bytesPerRow: 0,
                              space: CGColorSpaceCreateDeviceRGB(),
                              bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return nil }
    ctx.interpolationQuality = .medium
    ctx.draw(image, in: CGRect(x: 0, y: 0, width: widthPts, height: heightPts))
    guard let scaled = ctx.makeImage() else { return nil }
    guard let png = NSBitmapImageRep(cgImage: scaled).representation(using: .png, properties: [:]) else { return nil }
    return png.base64EncodedString()
}

func captureOCR(_ req: Request) -> Response {
    do {
        let displayID = resolveDisplayID(req.display ?? 0)
        let bounds = CGDisplayBounds(displayID) // global points, top-left origin
        let t0 = Date()
        let image = try captureDisplay(displayID)
        let t1 = Date()
        let boxes = try recognize(image, level: req.level ?? "fast")
        let t2 = Date()
        let meta = Meta(imgW: image.width, imgH: image.height,
                        originXPts: bounds.minX, originYPts: bounds.minY,
                        widthPts: bounds.width, heightPts: bounds.height)
        var resp = Response(id: req.id, ok: true, meta: meta, observations: boxes)
        if req.includeImage == true {
            resp.pngBase64 = pngAtPointSize(image, widthPts: Int(bounds.width), heightPts: Int(bounds.height))
        }
        let t3 = Date()
        logErr(String(format: "capture=%.0fms ocr(%@)=%.0fms png=%.0fms words=%d",
                      t1.timeIntervalSince(t0) * 1000, req.level ?? "fast",
                      t2.timeIntervalSince(t1) * 1000, t3.timeIntervalSince(t2) * 1000, boxes.count))
        return resp
    } catch {
        return Response(id: req.id, ok: false, error: String(describing: error))
    }
}

func ocrFile(_ req: Request) -> Response {
    guard let path = req.path else {
        return Response(id: req.id, ok: false, error: "ocr_file requires path")
    }
    guard let nsImage = NSImage(contentsOfFile: path),
          let image = nsImage.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
        return Response(id: req.id, ok: false, error: "cannot load image at \(path)")
    }
    do {
        let boxes = try recognize(image, level: req.level ?? "accurate")
        let meta = Meta(imgW: image.width, imgH: image.height, originXPts: 0, originYPts: 0,
                        widthPts: Double(image.width), heightPts: Double(image.height))
        return Response(id: req.id, ok: true, meta: meta, observations: boxes)
    } catch {
        return Response(id: req.id, ok: false, error: String(describing: error))
    }
}

func handle(_ req: Request) -> Response {
    switch req.op {
    case "ping":
        var resp = Response(id: req.id, ok: true)
        resp.axTrusted = AXIsProcessTrusted()
        return resp
    case "capture_ocr":
        return captureOCR(req)
    case "ocr_file":
        return ocrFile(req)
    case "ax_query":
        return axQuery(req)
    default:
        return Response(id: req.id, ok: false, error: "unknown op \(req.op)")
    }
}

logErr("ready")
while let line = readLine(strippingNewline: true) {
    guard !line.isEmpty else { continue }
    guard let data = line.data(using: .utf8),
          let req = try? JSONDecoder().decode(Request.self, from: data) else {
        logErr("dropping undecodable request line")
        continue
    }
    emit(handle(req))
}
