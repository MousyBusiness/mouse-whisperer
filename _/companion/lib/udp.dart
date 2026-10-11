import 'dart:async';
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter/material.dart';

/// A resilient UDP client that automatically reconnects with exponential backoff.
class UdpClient {
  RawDatagramSocket? _socket;
  InternetAddress? _serverAddress;
  int? _serverPort;

  // Connection parameters for reconnection
  String? _serverAddressParam;
  String? _serverHostParam;
  int? _serverPortParam;
  int _localPort = 0;

  // Reconnection state
  bool _shouldReconnect = false;
  int _retryAttempt = 0;
  Timer? _reconnectTimer;
  
  static const Duration _initialBackoff = Duration(milliseconds: 100);
  static const Duration _maxBackoff = Duration(seconds: 5);

  bool get isConnected => _socket != null && _serverAddress != null;

  /// Binds the UDP socket and sets the target server.
  /// Automatically reconnects on failure with exponential backoff.
  /// 
  /// [serverHost] - The server's IP address or hostname.
  /// [serverPort] - The server's port.
  /// [localPort] - Optional local port to bind to (0 = auto-assign).
  Future<void> connect({
    String? serverAddress,
    String? serverHost,
    required int serverPort,
    int localPort = 0,
  }) async {
    assert(serverAddress != null || serverHost != null, 'Either serverAddress or serverHost must be provided');
    
    // Store connection parameters for reconnection
    _serverAddressParam = serverAddress;
    _serverHostParam = serverHost;
    _serverPortParam = serverPort;
    _localPort = localPort;
    _shouldReconnect = true;
    
    await _connect();
  }

  Future<void> _connect() async {
    _cancelReconnectTimer();
    
    try {
      _socket?.close();
      _socket = null;

      if (_serverAddressParam != null) {
        _serverAddress = InternetAddress.tryParse(_serverAddressParam!);
      } else if (_serverHostParam != null) {
        _serverAddress = (await InternetAddress.lookup(_serverHostParam!)).first;
      }
      _serverPort = _serverPortParam;

      _socket = await RawDatagramSocket.bind(InternetAddress.anyIPv4, _localPort);
      
      // Reset retry attempt on successful connection
      _retryAttempt = 0;
      debugPrint('UDP connected to ${_serverAddress?.address}:$_serverPort');
    } catch (e) {
      debugPrint('UDP connection failed: $e');
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (!_shouldReconnect) return;
    
    final backoff = _calculateBackoff();
    debugPrint('Scheduling UDP reconnect in ${backoff.inMilliseconds}ms (attempt ${_retryAttempt + 1})');
    
    _reconnectTimer = Timer(backoff, () {
      _retryAttempt++;
      _connect();
    });
  }

  Duration _calculateBackoff() {
    // Exponential backoff: initial * 2^attempt, capped at max
    final exponentialMs = _initialBackoff.inMilliseconds * pow(2, _retryAttempt);
    final cappedMs = min(exponentialMs, _maxBackoff.inMilliseconds);
    return Duration(milliseconds: cappedMs.toInt());
  }

  void _cancelReconnectTimer() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }

  /// Sends a byte array to the server.
  /// 
  /// Returns the number of bytes sent, or -1 if not connected.
  int send(Uint8List data) {
    if (!isConnected) {
      _scheduleReconnect();
      return -1;
    }
    
    try {
      return _socket!.send(data, _serverAddress!, _serverPort!);
    } catch (e) {
      debugPrint('UDP send failed: $e');
      _scheduleReconnect();
      return -1;
    }
  }

  /// Sends a list of integers as bytes.
  int sendBytes(List<int> bytes) {
    return send(Uint8List.fromList(bytes));
  }

  /// Closes the socket and stops reconnection attempts.
  Future<void> close() async {
    _shouldReconnect = false;
    _cancelReconnectTimer();
    _socket?.close();
    _socket = null;
    _serverAddress = null;
    _serverPort = null;
    _retryAttempt = 0;
  }
}

