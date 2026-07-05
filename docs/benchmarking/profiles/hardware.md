# Torus Hardware Profiles

**Version:** 1.0
**Status:** Active

---

# 1. Purpose

This document defines the hardware profiles used throughout the Torus benchmarking framework.

A Hardware Profile represents the physical or virtual machine on which benchmarks are executed.

Benchmark reports reference Hardware Profiles instead of repeating hardware specifications.

Hardware Profiles are immutable once published.

If hardware changes significantly, a new Hardware Profile must be created.

---

# 2. Naming Convention

Hardware Profiles use the following format:

H001

H002

H003

...

Identifiers are permanent.

Identifiers are never reused.

---

# 3. Profile Template

Every Hardware Profile should contain the following information.

---

## General

Profile ID

Machine Name

Machine Type

Owner

Status

Date Created

Date Retired (if applicable)

Purpose

Examples

- Local Development
- Local Benchmarking
- Cloud Benchmarking
- Comparative Benchmarking

---

## CPU

Vendor

Model

Microarchitecture

Architecture

Physical Cores

Logical Threads

Base Frequency

Turbo Frequency

Cache

SIMD Extensions

Examples

SSE4.2

AVX2

AVX512

AES-NI

---

## Memory

Installed RAM

Memory Type

Memory Speed

Memory Channels

Swap Configuration

---

## Storage

Storage Type

Filesystem

Capacity

---

## Operating System

Distribution

Version

Kernel Version

Kernel Scheduler

CPU Governor

Transparent Huge Pages

NUMA

---

## Network

Network Type

NIC

Maximum Link Speed

Loopback Testing

Cloud Network

---

## Virtualization

Bare Metal

VM

Container

Cloud Instance

Hypervisor

---

## Benchmark Suitability

Suitable For

Throughput

Latency

Scalability

Reliability

Comparative

Long-running

Not Suitable For

Examples

Kernel profiling

NUMA

Multi-socket benchmarks

---

## Notes

Anything unique about the system.

Examples

Laptop CPU

Battery Powered

Cloud VM

Shared Host

Thermal Constraints

---

# 4. Active Hardware Profiles

---

# H001

## General

Profile ID

H001

Machine Name

HP 250 G8 Notebook PC

Machine Type

Laptop

Owner

Ashish Barmaiya

Status

Active

Purpose

Local Development

Local Benchmarking

---

## CPU

Vendor

Intel

Model

Core i3-1115G4

Microarchitecture

Tiger Lake

Architecture

x86_64

Physical Cores

2

Logical Threads

4

Base Frequency

3.00 GHz

Turbo Frequency

4.10 GHz

SIMD Extensions

SSE4.2

AVX2

AES-NI

---

## Memory

Installed RAM

8 GB

Memory Type

DDR4

Memory Speed

2666 MT/s

Swap

System Default

---

## Storage

Type

SSD

Filesystem

ext4

---

## Operating System

Distribution

Ubuntu

Version

26.04 LTS

CPU Governor

performance (recommended during benchmarking)

Transparent Huge Pages

Default

NUMA

No

---

## Network

Loopback

Primary Benchmark Network

---

## Virtualization

Bare Metal

---

## Benchmark Suitability

Suitable

✓ Feature Benchmarks

✓ Microbenchmarks

✓ Optimization Studies

✓ Local Regression Testing

✓ Development

Limitations

Small core count

Limited memory

Not suitable for production-scale comparisons

---

## Notes

Primary development machine.

Results obtained on this profile should be interpreted as local engineering benchmarks rather than production proxy benchmarks.

---

# 5. Future Hardware Profiles

Reserved

---

H002

Oracle Cloud AMD VM

---

H003

Oracle Cloud Ampere ARM VM

---

H004

Dedicated Bare Metal Benchmark Machine (Future)

---

# 6. Profile Evolution

Hardware Profiles are immutable.

If hardware changes significantly:

Do NOT modify an existing profile.

Create a new Hardware Profile.

Examples

Correct

H001

↓

H002

Incorrect

Edit H001 CPU model.

---

# 7. Referencing Hardware Profiles

Every benchmark report references the Hardware Profile.

Example

Hardware Profile

H001

Environment Profile

E001

Methodology

v1.0

This uniquely identifies the benchmark environment.

---

# 8. Future Extensions

Future Hardware Profiles may include

Power Consumption

CPU Package Temperature

Energy Measurements

NUMA Topology

Cache Hierarchy

PCIe Topology

Hardware Performance Counters

These are outside the scope of Version 1.0.
