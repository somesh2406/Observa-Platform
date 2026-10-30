# Increment 0 — Product Contract and Engineering Baseline

This is a historical acceptance record for the initial product and repository baseline. Current
implementation scope is described in [Product Contract](../product-contract.md) and
[Architecture HLD](../architecture/hld.md).

The current source tree contains the Go ingestion service, shared event-contract library,
Envoy gateway configuration, Kafka topic provisioner, local infrastructure, and supporting
tests. Product views, consumer processing, analytical storage, and query APIs are not
implemented; see the product contract's **Out of Scope (Deferred)** section.
