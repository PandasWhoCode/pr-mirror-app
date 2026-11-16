// SPDX-License-Identifier: Apache-2.0

//go:build unix

package openapi

// The project brandmark lives under docs/assets/ so it can
// be referenced from human-facing docs (README, etc.) without pulling in a
// Go package. `go generate` copies it into this package directory at build
// time so the embed directive in embeds.go can pick it up; the copy is
// gitignored. The Windows companion is generate_windows.go.

//go:generate cp ../../docs/assets/pr-mirror.svg logo.svg
