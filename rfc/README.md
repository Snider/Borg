# Borg RFC Specifications

This directory contains technical specifications (RFCs) for the Borg project.

## Index

| RFC | Title | Status | Description |
|-----|-------|--------|-------------|
| [001](RFC-001-OSS-DRM.md) | Open Source DRM | Proposed | Core DRM system for independent artists |
| [002](RFC-002-SMSG-FORMAT.md) | SMSG Container Format | Draft | Encrypted container format (v1/v2/v3) |
| [003](RFC-003-DATANODE.md) | DataNode | Draft | In-memory filesystem abstraction |
| [004](RFC-004-TIM.md) | Terminal Isolation Matrix | Draft | OCI-compatible container bundle |
| [005](RFC-005-STIM.md) | Encrypted TIM | Draft | ChaCha20-Poly1305 encrypted containers |
| [006](RFC-006-TRIX.md) | TRIX PGP Format | Draft | PGP encryption for archives and accounts |
| [007](RFC-007-LTHN.md) | LTHN Key Derivation | Draft | Rainbow-table resistant rolling keys |
| [008](RFC-008-BORGFILE.md) | Borgfile | Draft | Container compilation syntax |
| [009](RFC-009-STMF.md) | Secure To-Me Form | Draft | Asymmetric form encryption |
| [010](RFC-010-WASM-API.md) | WASM Decryption API | Draft | Browser decryption interface |

## Status Definitions

| Status | Meaning |
|--------|---------|
| **Draft** | Initial specification, subject to change |
| **Proposed** | Ready for review, implementation may begin |
| **Accepted** | Approved, implementation complete |
| **Deprecated** | Superseded by newer specification |

## Contributing

1. Create a new RFC with the next available number
2. Use the template format (see existing RFCs)
3. Start with "Draft" status
4. Update this README index

## Related Documentation

- [CLAUDE.md](../CLAUDE.md) - Developer quick reference
- [docs/](../docs/) - User documentation
- [examples/formats/](../examples/formats/) - Format examples
