# Torus Environment Profiles

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines the Environment Profiles used throughout the Torus benchmarking framework.

An Environment Profile describes the benchmark topology rather than the underlying hardware.

While a Hardware Profile specifies *where* benchmarks are executed, an Environment Profile specifies *how* the benchmark components are connected.

Benchmark reports reference Environment Profiles to ensure experiments remain reproducible.

---

# 2. Naming Convention

Environment Profiles use the following format:

E001

E002

E003

...

Identifiers are permanent.

Identifiers are never reused.

---

# 3. Profile Template

Every Environment Profile should contain the following information.

---

## General

Environment ID

Name

Status

Purpose

Description

---

## Topology

Describe the benchmark topology.

Example

Benchmark Client

↓

Proxy

↓

Backend

Topology diagrams are encouraged.

---

## Machines

List every participating machine.

For each machine specify:

- Hardware Profile
- Role
- Operating System

---

## Network

Network Type

Loopback

LAN

WAN

Cloud VCN

Internet

---

Network Characteristics

Bandwidth

Latency

Packet Loss

MTU

NIC Speed

---

## Benchmark Roles

Client

Proxy

Backend

Metrics Collector

Monitoring

Logging

---

## Synchronization

Time Synchronization

Clock Source

NTP

Required for distributed benchmarks.

---

## Suitability

Suitable For

Throughput

Latency

Comparative

Scalability

Long-running

Reliability

---

Limitations

---

## Notes

Additional observations.

---

# 4. Active Environment Profiles

---

# E001

## Name

Local Loopback

---

## Purpose

Development

Feature Benchmarking

Regression Benchmarking

---

## Topology

Benchmark Client

↓

Torus

↓

Backend

All running on the same machine.

---

## Machines

Machine 1

Hardware Profile

H001

Role

Client

Proxy

Backend

---

## Network

Type

Loopback

Bandwidth

Operating System Limited

Latency

Near Zero

Packet Loss

None

---

## Monitoring

Local

---

## Suitability

✓ Feature Benchmarks

✓ Regression

✓ Microbenchmarks

✓ Optimization Studies

---

Limitations

Not representative of production networking.

No real network latency.

No NIC bottleneck.

No inter-machine scheduling.

---

# E002

## Name

Single VM

---

Purpose

Cloud Feature Benchmarking

---

Topology

Benchmark Client

↓

Proxy

↓

Backend

All running on the same Oracle VM.

---

Machines

Machine 1

Hardware Profile

H002

Role

Client

Proxy

Backend

---

Network

Loopback

Cloud VM

---

Suitability

✓ Cloud Regression

✓ Feature Benchmarking

---

Limitations

Still localhost networking.

---

# E003

## Name

Two-Machine Deployment

---

Purpose

Real Network Benchmarking

---

Topology

Benchmark Client

↓

Proxy

↓

Backend

Proxy and backend run on separate machines.

---

Machines

Machine 1

Hardware

H002

Role

Proxy

---

Machine 2

Hardware

H002

Role

Backend

---

Network

Cloud Private Network

---

Suitability

✓ Throughput

✓ Latency

✓ Reliability

✓ Production-like networking

---

Limitations

Limited scale.

---

# E004

## Name

Three-Machine Deployment

---

Purpose

Production-style benchmarking.

---

Topology

Benchmark Client

↓

Proxy

↓

Backend Cluster

---

Machines

Machine 1

Role

Benchmark Client

---

Machine 2

Role

Torus

---

Machine 3

Role

Backend Cluster

---

Suitability

✓ Comparative Benchmarks

✓ Performance Reports

✓ Scalability

✓ Reliability

---

# E005

## Name

Comparative Benchmark Environment

---

Purpose

Fair comparison against:

NGINX

HAProxy

Envoy

Caddy

Traefik

---

Topology

Benchmark Client

↓

Proxy Under Test

↓

Backend Cluster

---

Rules

Every proxy must use:

- identical hardware
- identical topology
- identical backend
- identical benchmark client
- identical benchmark tool
- identical benchmark duration
- equivalent configuration

---

Suitability

✓ Comparative Benchmark Reports

---

# 5. Environment Evolution

Environment Profiles are immutable.

If topology changes significantly:

Create a new Environment Profile.

Do not modify existing profiles.

---

# 6. Referencing Environment Profiles

Every benchmark report references:

Hardware Profile

Environment Profile

Software Baseline

Methodology Version

Example

Hardware

H002

Environment

E004

Software Baseline

S001

Methodology

v1.0

---

# 7. Future Environments

Future profiles may include

Multi-region

Cross-cloud

IPv6

HTTP/3

Service Mesh

Kubernetes

Multi-AZ

NUMA

Bare Metal

---

# 8. Scope

Environment Profiles describe benchmark topology only.

Hardware characteristics belong in Hardware Profiles.

Software configuration belongs in Software Baselines.

Statistical methodology belongs in the Methodology document.
