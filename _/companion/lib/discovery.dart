import 'dart:async';
import 'dart:io';

import 'package:bonsoir/bonsoir.dart';
import 'package:flutter/foundation.dart';

/// Represents a discovered Mouse Whisperer service.
class DiscoveredService {
  final String name;
  final String ip;
  final int port;

  DiscoveredService({required this.name, required this.ip, required this.port});

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DiscoveredService &&
          runtimeType == other.runtimeType &&
          name == other.name &&
          ip == other.ip &&
          port == other.port;

  @override
  int get hashCode => name.hashCode ^ ip.hashCode ^ port.hashCode;

  @override
  String toString() => 'DiscoveredService($name, $ip:$port)';
}

/// Discovers Mouse Whisperer servers on the local network using Bonjour/mDNS.
class ServiceDiscovery {
  // Note: Using _tcp for mDNS service type (standard convention), even though the actual service uses UDP
  static const String serviceType = '_mousewhisperer._tcp';

  late BonsoirDiscovery _discovery;
  StreamSubscription? _subscription;

  final _servicesController = StreamController<Set<DiscoveredService>>.broadcast();
  final Set<DiscoveredService> _services = {};

  /// Stream of discovered services. Emits whenever services are added or removed.
  Stream<Set<DiscoveredService>> get servicesStream => _servicesController.stream;

  /// Current set of discovered services.
  Set<DiscoveredService> get services => Set.unmodifiable(_services);

  ServiceDiscovery() {
    _discovery = BonsoirDiscovery(type: serviceType);
  }

  /// Starts discovering services on the local network.
  Future<void> start() async {
    if(_discovery.isStopped) {
      _discovery = BonsoirDiscovery(type: serviceType);
    }
    await _discovery.initialize();
    await _discovery.start();
    if (_discovery.isReady) {
      debugPrint('ServiceDiscovery: Ready');
    }

    _subscription = _discovery.eventStream?.listen((event) {
      switch (event) {
        case BonsoirDiscoveryStartedEvent():
          debugPrint('ServiceDiscovery: Started');
          break;
        case BonsoirDiscoveryStoppedEvent():
          debugPrint('ServiceDiscovery: Stopped');
          break;
        case BonsoirDiscoveryServiceFoundEvent():
          debugPrint('ServiceDiscovery: Found ${event.service.name}');
          event.service.resolve(_discovery.serviceResolver);
          break;
        case BonsoirDiscoveryServiceResolvedEvent():
          debugPrint('ServiceDiscovery: Resolved ${event.service.toJson()}');
          _resolveService(event.service);
          break;
        case BonsoirDiscoveryServiceLostEvent():
          debugPrint('ServiceDiscovery: Lost ${event.service.name}');
          _removeServiceByName(event.service.name);
          break;
        default:
          debugPrint('ServiceDiscovery: Unknown event $event');
      }
    });
  }

  Future<void> _resolveService(BonsoirService service) async {
    final host = service.host;
    if (host == null) return;

    try {
      final addresses = await InternetAddress.lookup(host, type: InternetAddressType.IPv4);
      if (addresses.isNotEmpty) {
        final ip = addresses.first.address.split('%').first;
        final discovered = DiscoveredService(name: service.name, ip: ip, port: service.port);
        _services.add(discovered);
        _servicesController.add(Set.from(_services));
        debugPrint('ServiceDiscovery: Added $discovered');
      }
    } catch (e) {
      debugPrint('ServiceDiscovery: Failed to resolve host $host: $e');
    }
  }

  void _removeServiceByName(String name) {
    _services.removeWhere((s) => s.name == name);
    _servicesController.add(Set.from(_services));
  }

  /// Stops discovering services.
  Future<void> stop() async {
    await _subscription?.cancel();
    _subscription = null;
    await _discovery.stop();
    _services.clear();
  }

  /// Disposes of resources.
  Future<void> dispose() async {
    await stop();
    await _servicesController.close();
  }
}
