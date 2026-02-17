package cdb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"tgp/internal/model"
)

const (
	indexVersion  = 1
	indexFilename = "index.yml"
)

type VersionKind string

const (
	VersionKindTag    VersionKind = "tag"
	VersionKindBranch VersionKind = "branch"
)

type VersionMeta struct {
	Kind        VersionKind `yaml:"kind"`
	Updated     string      `yaml:"updated"`
	ProjectFile string      `yaml:"projectFile"`
}

type ProjectMeta struct {
	Origin     string                 `yaml:"origin"`
	ModulePath string                 `yaml:"modulePath"`
	Versions   map[string]VersionMeta `yaml:"versions"`
}

type Index struct {
	Version  int                    `yaml:"version"`
	Aliases  map[string]string      `yaml:"aliases"`
	Projects map[string]ProjectMeta `yaml:"projects"`
}

func LoadIndex(root string) (idx *Index, err error) {

	p := filepath.Join(root, indexFilename)
	var data []byte
	if data, err = os.ReadFile(p); err != nil {
		if os.IsNotExist(err) {
			return &Index{
				Version:  indexVersion,
				Aliases:  make(map[string]string),
				Projects: make(map[string]ProjectMeta),
			}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	idx = new(Index)
	if err = yaml.Unmarshal(data, idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	if idx.Aliases == nil {
		idx.Aliases = make(map[string]string)
	}
	if idx.Projects == nil {
		idx.Projects = make(map[string]ProjectMeta)
	}
	return
}

func SaveIndex(root string, idx *Index) (err error) {

	if err = os.MkdirAll(root, 0700); err != nil {
		return fmt.Errorf("mkdir root: %w", err)
	}

	var data []byte
	if data, err = yaml.Marshal(idx); err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	p := filepath.Join(root, indexFilename)
	if err = os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return
}

// UpsertProject добавляет или обновляет запись проекта и версии в индексе.
func UpsertProject(root string, idx *Index, projectKey string, origin string, modulePath string, version string, kind VersionKind) (projectFile string, err error) {

	if idx.Projects == nil {
		idx.Projects = make(map[string]ProjectMeta)
	}

	meta, ok := idx.Projects[projectKey]
	if !ok {
		meta = ProjectMeta{
			Origin:     origin,
			ModulePath: modulePath,
			Versions:   make(map[string]VersionMeta),
		}
	}
	if meta.Versions == nil {
		meta.Versions = make(map[string]VersionMeta)
	}

	versionNorm := NormalizeVersionName(version)
	relPath := filepath.Join(projectKey, string(kind), versionNorm+".astg")
	meta.Versions[version] = VersionMeta{
		Kind:        kind,
		Updated:     time.Now().UTC().Format(time.RFC3339),
		ProjectFile: relPath,
	}
	idx.Projects[projectKey] = meta

	return relPath, nil
}

// ResolveRef находит в индексе projectKey (после раскрытия алиаса) и версию; если version пустой — берётся последняя по updated.
func ResolveRef(idx *Index, ref Ref) (projectKey string, projectFile string, err error) {

	projectKey = ResolveAlias(ref.ProjectKey, idx.Aliases)
	meta, ok := idx.Projects[projectKey]
	if !ok {
		return "", "", fmt.Errorf("project not found: %s", ref.ProjectKey)
	}
	if len(meta.Versions) == 0 {
		return projectKey, "", fmt.Errorf("project has no versions: %s", projectKey)
	}

	version := ref.Version
	if version == "" {
		var latest string
		var latestTime time.Time
		for v, vm := range meta.Versions {
			t, _ := time.Parse(time.RFC3339, vm.Updated)
			if t.After(latestTime) {
				latestTime = t
				latest = v
			}
		}
		if latest == "" {
			return projectKey, "", fmt.Errorf("no version for project: %s", projectKey)
		}
		version = latest
	}

	vm, ok := meta.Versions[version]
	if !ok {
		return projectKey, "", fmt.Errorf("version not found: %s@%s", projectKey, version)
	}
	return projectKey, vm.ProjectFile, nil
}

func OriginFromProject(project *model.Project) (origin string, modulePath string, version string, kind VersionKind) {

	if project.Git == nil {
		return "", project.ModulePath, "default", VersionKindBranch
	}
	origin = NormalizeRemoteURLToHostPath(project.Git.RemoteURL)
	modulePath = project.ModulePath
	if project.Git.Tag != "" {
		return origin, modulePath, project.Git.Tag, VersionKindTag
	}
	if project.Git.Branch != "" {
		return origin, modulePath, project.Git.Branch, VersionKindBranch
	}
	return origin, modulePath, "default", VersionKindBranch
}

func ListRefs(idx *Index) (refs []string) {

	refs = make([]string, 0)
	for projectKey, meta := range idx.Projects {
		for version := range meta.Versions {
			refs = append(refs, projectKey+"@"+version)
		}
	}
	sort.Strings(refs)
	return
}
