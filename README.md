# Lightning-Fast Database Tests

[![Lint](https://github.com/hallabro/lightning-fast-database-tests/actions/workflows/lint.yaml/badge.svg)](https://github.com/hallabro/lightning-fast-database-tests/actions/workflows/lint.yaml)

This repository contains the demonstration code and test suite used in the "Lightning-Fast Database Tests" 
presentation  at Gophercon 2024. It showcases the performance improvements achieved using [pgtestdb](https://github.com/peterldowns/pgtestdb),
a tool for creating ephemeral Postgres databases for testing.

The test suite creates and lists 1000 products under various configurations: sequentially, in parallel
using `pgtestdb`, and with additional optimizations like disabling fsync and using tmpfs.

By running these tests, you can observe the speed improvements possible in database testing, potentially leading to
faster development cycles and more efficient CI/CD pipelines. To get started, clone this repo, ensure you have Go,
Docker and Postgres installed, and run `go test ./...` to see the comparisons in action.