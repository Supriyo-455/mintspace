# Concepts, Philosophy & Future Roadmap

This document explores the philosophical concepts behind MintSpace, scholarly publishing conventions, and the long-term roadmap for decentralized research networks.

---

## 1. The Core Vision: Calm, Cryptographic Publishing

Modern web platforms have evolved into engagement-driven distraction engines filled with bright colors, floating widgets, pop-ups, and fragile database dependencies.

MintSpace is built on three contrarian principles:

1. **Quiet, Paper-First Typography**:
   A reader engaging with a technical manuscript should feel calm and focused. The layout mirrors classical books and mathematical proceedings, with typography tuned for long-form reading (**Constantia** for prose, **Consolas** for code).
2. **Cryptographic Immutability by Default**:
   Documents are identified by their cryptographic hash. If a paper claims to be version `MS-5`, its contents must match `SHA-256(M)`. Citations do not break when servers migrate.
3. **Decentralized Readiness**:
   Because content storage is completely decoupled from relational metadata, documents can be pinned to distributed peer-to-peer networks (IPFS, Arweave, MinIO) without changing the core application logic.

---

## 2. Scholarly Constructs in MintSpace

### A. Threaded Peer Review & Commentary
Instead of flat, social-media-style comment feeds, MintSpace models discussions as a **scholarly peer review tree**:
* Any verified researcher can submit a formal review or critique.
* Authors and reviewers can reply inline to specific points, creating a multi-level thread of critique.
* Community members can endorse both manuscripts and specific commentary notes.
* Authors and administrators retain retraction and deletion rights over their notes.

### B. Dynamic BibTeX Citation Generator
Every manuscript published on MintSpace generates a standardized BibTeX citation:

```bibtex
@article{mintspace_1,
  author  = {Editorial Board},
  title   = {Content-Addressable Storage and Distributed Archiving in Modern Web Platforms},
  journal = {MintSpace Open Research Archive},
  year    = {2026},
  doi     = {10.mintspace/paper.1},
  hash    = {72128b438af655bb2dcb3756fa7a8ca398337637cfc7c0759876929082415fb4}
}
```

Scholars can copy citations with a single click, embedding cryptographic content hashes directly into their LaTeX references.

---

## 3. Product Roadmap

### Phase 1: Storage Layer Evolution
* [x] Pure-Go SQLite WAL relational metadata engine.
* [x] Content-Addressable Storage (CAS) on local filesystem.
* [x] Multi-machine self-healing peer CAS mesh with SHA-256 verification.
* [ ] **IPFS Pinning Adapter**: Automatically publish and pin manuscripts to decentralized IPFS nodes (Pinata, Infura, or native IPFS daemon).
* [ ] **Arweave Permanent Archiving**: One-click permanent manuscript archiving to the Arweave permaweb.
* [ ] **S3 / MinIO Pluggable Backend**: Direct object storage integration for cloud-native enterprise deployments.

### Phase 2: Cryptographic Identity & Provenance
* [ ] **Ed25519 Author Signatures**: Authors cryptographically sign their Markdown manuscripts using local keypairs (or WebAuthn / Passkeys), proving authorship without trusting the server database.
* [ ] **Decentralized Identifiers (DID)**: Support for W3C DIDs allowing cross-platform scholar identity.
* [ ] **ORCID Integration**: Linking scholar accounts to official scientific ORCID registries.

### Phase 3: Collaborative Peer Review Workflows
* [ ] **Preprint & Formal Review Tracks**: Double-blind and open peer review workflows with editorial accept/reject decisions.
* [ ] **Diff & Revision History**: Comparing two revisions of a manuscript via git-like visual diffs based on content hash changes.
* [ ] **PDF Export Engine**: Compiling Markdown, KaTeX, and Mermaid diagrams into camera-ready PDF manuscripts with Typst or LaTeX.
