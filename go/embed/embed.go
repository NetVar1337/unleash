// Package embed provides go:embed access to patches and contrib files.
//
// go:embed cannot reference paths above the module root, so a build step
// must copy ../../patches/ and ../../contrib/ into this directory before
// building. Use the copy-assets script (or Makefile) to prepare:
//
//	go/embed/patches/*.json   ← from patches/*.json
//	go/embed/contrib/**       ← from contrib/**
package embed

import "embed"

// EmbeddedPatches holds all patch JSON files.
// The build step must copy patches/*.json into go/embed/patches/ first.
//
//go:embed patches/*.json
var EmbeddedPatches embed.FS

// EmbeddedContrib holds all contrib files (rules, wrappers, preload, etc.).
// The build step must copy contrib/ into go/embed/contrib/ first.
//
//go:embed all:contrib
var EmbeddedContrib embed.FS
