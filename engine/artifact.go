package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	"github.com/zenful-ai/arboreal"
)

type ArtifactManifest struct {
	SchemaVersion  int       `json:"schema_version" gorm:"default:1"`
	RuntimeVersion int       `json:"runtime_version" gorm:"default:1"`
	ProjectID      uuid.UUID `json:"project_id"`
	CanvasID       uuid.UUID `json:"canvas_id"`
	Version        uint      `json:"artifact_version"`
	Notes          string    `json:"notes"`
}

type Artifact struct {
	ArtifactManifest
	Data []byte `json:"artifact" gorm:"column:artifact"`
}

type Instance struct {
	Artifact []byte                     `json:"artifact"`
	Snapshot arboreal.Snapshot          `json:"snapshot"`
	History  arboreal.AnnotatedMessages `json:"history"`
}

func RuntimeForArtifact(artifact []byte, options *RuntimeOptions) (*Runtime, error) {
	// Create a new runtime object
	r, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
	if err != nil {
		return nil, err
	}

	var profiles []arboreal.MCPProfile
	var manifest ArtifactManifest
	var code string
	for _, f := range r.File {
		if f.FileInfo().Name() == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(b, &manifest)
			if err != nil {
				return nil, err
			}
		}
		if f.FileInfo().Name() == "main.lua" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			code = string(b)
		}
		if f.FileInfo().Name() == "profiles" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(b, &profiles)
			if err != nil {
				return nil, err
			}
		}
	}

	if options.MCPProfile != nil && options.MCPClient == nil {
		options.MCPClient = arboreal.NewMCPClientMux()

		for _, profile := range profiles {
			if profile.Type == *options.MCPProfile {
				for _, server := range profile.Servers {
					switch server.Type {
					case arboreal.MCPServerTypeSSE:
						options.MCPClient.AddSSEServer(context.Background(), server.Location)
					}
				}
			}

			break
		}
	}

	runtimeVersion := manifest.RuntimeVersion
	if runtimeVersion == 0 {
		runtimeVersion = 1
	}

	return InitializeRuntime(code, runtimeVersion, options)
}
