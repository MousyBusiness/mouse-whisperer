import Cocoa
import FlutterMacOS

@main
class AppDelegate: FlutterAppDelegate {
  private var modifierMonitor: Any?
  private var eventSink: FlutterEventSink?

  override func applicationDidFinishLaunching(_ notification: Notification) {
    let controller = mainFlutterWindow?.contentViewController as! FlutterViewController
    
    let channel = FlutterEventChannel(
      name: "mouse_whisperer/modifier_keys",
      binaryMessenger: controller.engine.binaryMessenger
    )
    channel.setStreamHandler(ModifierKeyStreamHandler(delegate: self))
  }

  func startMonitoring(sink: @escaping FlutterEventSink) {
    self.eventSink = sink
    
    // Send initial state
    sendModifierState(NSEvent.modifierFlags)
    
    // Monitor global modifier key changes
    modifierMonitor = NSEvent.addGlobalMonitorForEvents(matching: .flagsChanged) { [weak self] event in
      self?.sendModifierState(event.modifierFlags)
    }
  }

  func stopMonitoring() {
    if let monitor = modifierMonitor {
      NSEvent.removeMonitor(monitor)
      modifierMonitor = nil
    }
    eventSink = nil
  }

  private func sendModifierState(_ flags: NSEvent.ModifierFlags) {
    let state: [String: Bool] = [
      "command": flags.contains(.command),
      "control": flags.contains(.control),
      "alt": flags.contains(.option),
      "shift": flags.contains(.shift),
    ]
    eventSink?(state)
  }

  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return true
  }

  override func applicationSupportsSecureRestorableState(_ app: NSApplication) -> Bool {
    return true
  }
}
class ModifierKeyStreamHandler: NSObject, FlutterStreamHandler {
  weak var delegate: AppDelegate?
  
  init(delegate: AppDelegate) {
    self.delegate = delegate
  }
  
  func onListen(withArguments arguments: Any?, eventSink events: @escaping FlutterEventSink) -> FlutterError? {
    delegate?.startMonitoring(sink: events)
    return nil
  }
  
  func onCancel(withArguments arguments: Any?) -> FlutterError? {
    delegate?.stopMonitoring()
    return nil
  }
}

