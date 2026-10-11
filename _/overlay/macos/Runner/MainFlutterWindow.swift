import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSPanel {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    self.contentViewController = flutterViewController

    // Find the largest screen (matching Go backend logic which uses largest display)
    if let screen = NSScreen.screens.max(by: { $0.frame.width * $0.frame.height < $1.frame.width * $1.frame.height }) {
        print("Using largest screen: \(screen.frame.size.width) x \(screen.frame.size.height)")
        self.setFrame(screen.frame, display: true)
    }
    self.collectionBehavior = [
        .canJoinAllSpaces,
        .fullScreenAuxiliary,
        .stationary,
    ];
    self.backgroundColor = .clear
    self.isOpaque = false
    self.ignoresMouseEvents = true
    self.isOpaque = false;
    self.hasShadow = false;
    self.styleMask = [.borderless, .nonactivatingPanel]; 

    flutterViewController.backgroundColor = .clear
    self.level = NSPanel.Level.screenSaver
    
    self.orderFrontRegardless();

    // Make the overlay visible on all Spaces/Desktops
    // self.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
    
    RegisterGeneratedPlugins(registry: flutterViewController)
    
    super.awakeFromNib()
    self.makeKey();
  }
    
  override func constrainFrameRect(_ frameRect: NSRect, to screen: NSScreen?) -> NSRect {
      return screen?.frame ?? frameRect
  }
}
