package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Store 管理单个 JSON 配置文件的读写。所有写操作先改内存再落盘(全量覆盖写)。
type Store struct {
	path string
}

// New 用给定配置文件路径构造 Store。
func New(path string) *Store {
	return &Store{path: path}
}

// Load 读取配置。文件不存在时返回一个已初始化的空 Config(Version=1),不视为错误。
// JSON 损坏时返回错误。
func (s *Store) Load() (Config, error) {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Version: 1, Workspaces: []string{}, Projects: []Project{}}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, nil
}

// save 全量写回,先写临时文件再 rename,保证原子性。
func (s *Store) save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Add 新增项目。若已存在相同 Path(以 IDForPath 判重),则忽略(幂等)。
// 自动为项目分配基于 Path 的 ID。
func (s *Store) Add(p Project) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	p.ID = IDForPath(p.Path)
	for _, existing := range cfg.Projects {
		if existing.ID == p.ID {
			return nil // 已存在,幂等返回
		}
	}
	cfg.Projects = append(cfg.Projects, p)
	return s.save(cfg)
}

// Update 按 ID 覆盖已存在的项目。找不到则返回错误。
func (s *Store) Update(p Project) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	for i := range cfg.Projects {
		if cfg.Projects[i].ID == p.ID {
			cfg.Projects[i] = p
			return s.save(cfg)
		}
	}
	return errors.New("store: project not found: " + p.ID)
}

// Remove 按 ID 删除项目。找不到视为成功(幂等)。
func (s *Store) Remove(id string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	out := cfg.Projects[:0]
	for _, p := range cfg.Projects {
		if p.ID != id {
			out = append(out, p)
		}
	}
	cfg.Projects = out
	return s.save(cfg)
}

// SetWorkspaces 覆盖工作区根目录列表。
func (s *Store) SetWorkspaces(dirs []string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	cfg.Workspaces = dirs
	return s.save(cfg)
}
