package store

import (
	"captcha/pkg/model"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ResourceMap maps resource keys (e.g., "active.png", "fixed.png") to Resource
type ResourceMap map[string]*model.Resource

// ResourceStore manages background and template images and fonts
type ResourceStore interface {
	AddResource(captchaType string, res *model.Resource)
	AddTemplate(captchaType string, tag string, template ResourceMap)
	GetRandomResource(captchaType, tag string) (*model.Resource, error)
	GetRandomTemplate(captchaType, tag string) (ResourceMap, error)
	AddResourceDir(dirPath, tag string, captchaTypes ...string) error
}

type MemoryResourceStore struct {
	mu        sync.RWMutex
	resources map[string]map[string][]*model.Resource // type -> tag -> []Resource
	templates map[string]map[string][]ResourceMap    // type -> tag -> []ResourceMap
}

func NewMemoryResourceStore() *MemoryResourceStore {
	return &MemoryResourceStore{
		resources: make(map[string]map[string][]*model.Resource),
		templates: make(map[string]map[string][]ResourceMap),
	}
}

func (s *MemoryResourceStore) AddResource(captchaType string, res *model.Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if res.Tag == "" {
		res.Tag = model.DefaultTag
	}
	if s.resources[captchaType] == nil {
		s.resources[captchaType] = make(map[string][]*model.Resource)
	}
	s.resources[captchaType][res.Tag] = append(s.resources[captchaType][res.Tag], res)
}

func (s *MemoryResourceStore) AddTemplate(captchaType string, tag string, template ResourceMap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tag == "" {
		tag = model.DefaultTag
	}
	if s.templates[captchaType] == nil {
		s.templates[captchaType] = make(map[string][]ResourceMap)
	}
	s.templates[captchaType][tag] = append(s.templates[captchaType][tag], template)
}

func (s *MemoryResourceStore) GetRandomResource(captchaType, tag string) (*model.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byType, ok := s.resources[captchaType]
	if !ok || len(byType) == 0 {
		return nil, fmt.Errorf("no resource found for captcha type: %s", captchaType)
	}
	if tag == "" {
		tag = model.DefaultTag
	}
	list := byType[tag]
	if len(list) == 0 {
		// fallback to any tag
		for _, v := range byType {
			if len(v) > 0 {
				list = v
				break
			}
		}
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no resource found for type %s with tag %s", captchaType, tag)
	}
	return list[rand.Intn(len(list))], nil
}

func (s *MemoryResourceStore) GetRandomTemplate(captchaType, tag string) (ResourceMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byType, ok := s.templates[captchaType]
	if !ok || len(byType) == 0 {
		return nil, fmt.Errorf("no template found for captcha type: %s", captchaType)
	}
	if tag == "" {
		tag = model.DefaultTag
	}
	list := byType[tag]
	if len(list) == 0 {
		// fallback to any tag
		for _, v := range byType {
			if len(v) > 0 {
				list = v
				break
			}
		}
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no template found for type %s with tag %s", captchaType, tag)
	}
	return list[rand.Intn(len(list))], nil
}

// AddResourceDir scans a directory and adds all jpg, jpeg, png files as background images for the given types
func (s *MemoryResourceStore) AddResourceDir(dirPath, tag string, captchaTypes ...string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	if len(captchaTypes) == 0 {
		captchaTypes = []string{model.TypeSlider, model.TypeRotate, model.TypeConcat, model.TypeWordImageClick}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".png") {
			fullPath := filepath.Join(dirPath, entry.Name())
			res := &model.Resource{
				Type: "file",
				Data: fullPath,
				Tag:  tag,
			}
			for _, cType := range captchaTypes {
				s.AddResource(cType, res)
			}
		}
	}
	return nil
}

