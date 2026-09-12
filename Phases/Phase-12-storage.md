# Phase 12 — Storage and media

**Status:** IN PROGRESS  
**PRD source:** Sections 18, 30, 49–50, 65, Phase 12  
**Depends on:** Phase 04 and Phase 01  
**Unlocks:** Phase 13 and Phase 17

## Outcome

Add S3-compatible media and document storage through `ObjectStorage`, keeping file bytes out of PostgreSQL and private documents inaccessible by default.

## In scope

- MinIO development adapter and S3-compatible production adapter;
- presigned upload/download flow so large bytes avoid Go where appropriate;
- public product media versus private seller KYC/customer documents;
- content type, size, key ownership, and upload validation;
- image thumbnail/small/medium/large/original variants;
- orphaned temporary-file cleanup;
- access audit for sensitive documents;
- safe replacement/deletion lifecycle.

## Out of scope

Choosing a final cloud provider, public CDN rollout, arbitrary user file sharing, and storing business truth only in object metadata.

## Deliverables

- [ ] object keys are non-guessable and ownership-scoped;
- [ ] private objects require authorized short-lived access;
- [ ] public media is explicitly classified and cacheable only when safe;
- [ ] uploads are validated before becoming catalog media;
- [ ] image variants are generated or queued through bounded work;
- [ ] failed/orphaned uploads have cleanup behavior;
- [ ] PostgreSQL stores object metadata and references, not file bytes.

## Preview

Seller/admin reviewer can upload product media, see the selected variant on a product page, and receive denial for a private KYC object without the required role. Empty-media and failed-upload states are visible.

## Verification

- [ ] adapter contract tests run against a fake/object-storage test seam;
- [ ] MinIO integration tests cover upload, signed access, expiry, replacement, and deletion;
- [ ] private document access is denied to guests, other sellers, and support users without permission;
- [ ] content type/size/path traversal/object-key tests pass;
- [ ] image variant and cleanup tests pass;
- [ ] no raw private object URL appears in public HTML or logs;
- [ ] `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass.

## Gate G12

**PASS when:** media preview works through the storage port, public/private boundaries are tested, and cleanup/variant behavior is bounded and observable.

**BLOCK when:** private files are publicly guessable, uploads stream unbounded data through Go, or object storage becomes the only business record.

## Evidence

Attach storage adapter tests, signed-URL expiry/access output, image preview, private-document denial, and cleanup metrics.
