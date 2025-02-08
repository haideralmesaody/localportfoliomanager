import 'dart:convert';
import 'dart:io';
import 'dart:async';  // Add this import for TimeoutException
import 'package:http/http.dart' as http;
import 'package:flutter/foundation.dart' show kIsWeb;
import 'dart:html' if (dart.library.io) 'dart:io'; // For XMLHttpRequest in web

class ApiService {
  static const String baseUrl = 'http://localhost:8080/api';  // Check this is correct

  final http.Client _client = http.Client();

  // Update headers to include all necessary CORS headers
  Map<String, String> get headers => {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    if (kIsWeb) ...<String, String>{
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS, PUT, DELETE',
      'Access-Control-Allow-Headers': 'Content-Type, Accept',
    },
  };

  // Add this method for transactions
  Future<List<Map<String, dynamic>>> getTransactions(int portfolioId) async {
    try {
      print('Fetching transactions for portfolio: $portfolioId');
      final response = await _client.get(
        Uri.parse('$baseUrl/portfolios/$portfolioId/transactions'),
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          if (kIsWeb) 'Access-Control-Allow-Origin': '*',
        },
      );

      print('Response status: ${response.statusCode}');
      print('Response body: ${response.body}');

      if (response.statusCode == 200) {
        final Map<String, dynamic> decodedResponse = json.decode(response.body);
        print('Decoded response: $decodedResponse'); // Debug the parsed JSON

        // Check if transactions exists and is not null
        if (!decodedResponse.containsKey('transactions') || decodedResponse['transactions'] == null) {
          // Return empty list instead of throwing
          return [];
        }

        final List<dynamic> transactions = decodedResponse['transactions'];
        return transactions.cast<Map<String, dynamic>>();
      } else {
        throw Exception('Failed to load transactions: ${response.statusCode}');
      }
    } catch (e) {
      print('API Error: $e');
      throw Exception('Failed to fetch transactions: $e');
    }
  }

  // Add method to create transaction
  Future<Map<String, dynamic>> createTransaction(int portfolioId, Map<String, dynamic> transaction) async {
    try {
      final url = '$baseUrl/portfolios/$portfolioId/transactions';
      print('Making POST request to: $url');
      print('Request body: ${json.encode(transaction)}');

      // Add preflight check for web
      if (kIsWeb) {
        final preflightResponse = await _client.options(Uri.parse(url));
        if (preflightResponse.statusCode != 200) {
          throw Exception('CORS preflight failed');
        }
      }

      // Add retry logic
      int retries = 3;
      http.Response? response;
      
      while (retries > 0) {
        try {
          response = await _client.post(
            Uri.parse(url),
            headers: headers,
            body: json.encode(transaction),
          ).timeout(const Duration(seconds: 10));
          break;
        } catch (e) {
          print('Attempt failed: $e');
          retries--;
          if (retries == 0) rethrow;
          await Future.delayed(Duration(seconds: 1));
        }
      }

      if (response == null) {
        throw Exception('Failed to connect to server after retries');
      }

      print('Response status: ${response.statusCode}');
      print('Response body: ${response.body}');

      if (response.statusCode == 201 || response.statusCode == 200) {
        return json.decode(response.body) as Map<String, dynamic>;
      }

      throw Exception('Server error: ${response.statusCode}');
    } catch (e) {
      print('Transaction error: $e');
      throw Exception('Failed to create transaction: $e');
    }
  }

  // Add this method
  Future<Map<String, dynamic>> getStockDetails(String ticker) async {
    try {
      print('Fetching stock details for ticker: $ticker');
      final response = await _client.get(
        Uri.parse('$baseUrl/stocks/$ticker/prices'), // Changed from /details to /prices
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          if (kIsWeb) 'Access-Control-Allow-Origin': '*',
        },
      ).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          print('Request timed out');
          throw TimeoutException('Request timed out');
        },
      );

      print('Response status: ${response.statusCode}');
      print('Response headers: ${response.headers}');
      print('Response body: ${response.body}');

      if (response.statusCode == 200) {
        final decodedResponse = json.decode(response.body) as Map<String, dynamic>;
        print('Decoded response: $decodedResponse');
        return decodedResponse;
      } else {
        print('HTTP error: ${response.statusCode}');
        throw Exception('Failed to load stock details: ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      print('API Error: $e');
      throw Exception('Failed to fetch stock details: $e');
    }
  }

  Future<dynamic> get(String endpoint) async {
    try {
      final url = '$baseUrl/$endpoint';
      print('Calling API: $url');  // Debug log
      
      final response = await http.get(Uri.parse(url));
      
      print('Response status: ${response.statusCode}');  // Debug log
      print('Response body: ${response.body}');  // Debug log

      if (response.statusCode == 200) {
        return json.decode(response.body);
      } else {
        throw Exception('API Error: ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      print('API Error: $e');  // Debug log
      rethrow;
    }
  }

  Future<dynamic> post(String endpoint, Map<String, dynamic> body) async {
    try {
      final uri = Uri.parse('$baseUrl/$endpoint');
      print('POST Request to: $uri');
      print('Request body: $body');
      
      final response = await _client.post(
        uri,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          if (kIsWeb) 'Access-Control-Allow-Origin': '*',
        },
        body: json.encode(body),
      ).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          throw TimeoutException('Request timed out');
        },
      );
      
      print('Response status: ${response.statusCode}');
      print('Response body: ${response.body}');
      
      if (response.statusCode == 200 || response.statusCode == 201) {
        return json.decode(response.body);
      } else {
        throw HttpException('Server error: ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      print('API Error: $e');
      rethrow;
    }
  }

  Future<void> delete(String endpoint) async {
    try {
      final uri = Uri.parse('$baseUrl/$endpoint');
      print('DELETE Request to: $uri'); // Debug log
      
      final response = await http.delete(
        uri,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          if (kIsWeb) 'Access-Control-Allow-Origin': '*',
        },
      ).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          print('Request timed out');
          throw TimeoutException('Request timed out');
        },
      );
      
      print('Response status: ${response.statusCode}');
      print('Response body: ${response.body}');
      
      if (response.statusCode != 200) {
        throw HttpException('Server error: ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      print('API Error: $e');
      rethrow;
    }
  }

  Future<dynamic> put(String endpoint, Map<String, dynamic> body) async {
    try {
        final uri = Uri.parse('$baseUrl/$endpoint');
        print('PUT Request to: $uri');
        print('Request body: $body');
        
        final response = await http.put(
            uri,
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
                if (kIsWeb) 'Access-Control-Allow-Origin': '*',
            },
            body: json.encode(body),
        ).timeout(
            const Duration(seconds: 10),
            onTimeout: () {
                print('Request timed out');
                throw TimeoutException('Request timed out');
            },
        );
        
        print('Response status: ${response.statusCode}');
        print('Response body: ${response.body}');
        
        if (response.statusCode == 200) {
            return json.decode(response.body);
        } else {
            throw HttpException('Server error: ${response.statusCode} - ${response.body}');
        }
    } catch (e) {
        print('API Error: $e');
        rethrow;
    }
  }
} 