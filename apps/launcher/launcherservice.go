package main

import "ffresources/internal/shared/launcher"

// LauncherService expõe a leitura do manifest.json ao frontend do launcher.
type LauncherService struct{}

func (LauncherService) Load(path string) (*launcher.Manifest, error) {
	return launcher.LoadManifest(path)
}
