# Things to make this better

* More tests!
  - use httptest.Recorder and friends
* Get profiler support with flame graphs
* RBAC UX
* Add OAuth support via Ory Kratos
* Demo Hashicorp plugin architecture
* PoC working from swagger docs to generate
* Otel spans in intercepter
* Set up Otel collector
* Set docker compose (first)
  - Server (N+1)
  - Collector
  - DB
  - Metrics
  - Grafana
* Set up k8s (n+1 grpc servers, otel-collector, postgres)
* Merge in cmd/respawn for nonstop services
* Extend storage layer to include databases
  - SQLite and Postgres
  - sqlc for generating schemas
* Add simple UX to get a CRUD demo against APIs
  - Use HTMX
* Leverage zStd
  - Look at enabling zStd compression in gRPC -- https://pkg.go.dev/google.golang.org/grpc@v1.38.0/encoding#RegisterCompressor
  - zStd compression dictionaries: https://github.com/facebook/zstd
  - Identify training sets
  - Coordinate updating dicts

Dictionary info below:

Zstandard (zstd) offers dictionary compression to significantly improve compression ratios, especially for small files or data blocks with recurring patterns.
Purpose of Zstd Dictionaries:
Enhanced Compression for Small Data: Standard compression algorithms build their dictionary dynamically during a single pass, which is inefficient for small inputs. Zstd dictionaries pre-load common patterns, leading to better compression for such data.
Improved Efficiency on Repetitive Data: When compressing multiple similar files or data blocks (e.g., database blocks, network packets), a dictionary trained on a sample of this data can significantly boost compression performance by providing a pre-existing set of common strings.
How Zstd Dictionaries Work:
Dictionary Training: A dictionary is created by analyzing a representative sample of data that will be compressed. This process identifies frequently occurring patterns and stores them in the dictionary.
Compression with Dictionary: During compression, the zstd algorithm uses the pre-trained dictionary in addition to its dynamic dictionary to find matches and achieve higher compression ratios.
Decompression with Dictionary: The same dictionary used for compression must be provided during decompression to correctly reconstruct the original data.
Creating and Using Zstd Dictionaries:
Training: Dictionaries can be trained using the zstd C library API, the zstd command-line interface, or language-specific wrappers (e.g., python-zstandard's train_dict() function).
Compression: Functions like ZSTD_compress_usingDict (C API) or methods like Encoder::with_dictionary (Rust) are used to compress data with a specified dictionary.
Decompression: Similarly, functions like ZSTD_decompress_usingDict (C API) or methods like Decoder::with_dictionary (Rust) are used for decompression with the corresponding dictionary.
Considerations:
Dictionary Size: The optimal dictionary size depends on the data characteristics.
Performance Impact: While dictionaries improve compression ratios, they can introduce a slight overhead during compression and decompression due to the dictionary lookup.
Dictionary ID: Zstd dictionaries are typically generated with a dictionary ID, which can be used to identify and retrieve the correct dictionary during decompression.

