package filesystem

import (
	"GoFlix/configs"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size,omitempty"`
	Children []File `json:"children,omitempty"`
}

func (f *File) AddChild(child File) {
	f.Children = append(f.Children, child)
}

func BuildSafePath(baseDir, userPath string) (string, error) {
	// Очищаем путь от множественных слешей и относительных переходов
	cleanPath := filepath.Clean(userPath)
	// Убираем ведущий слеш если есть
	if strings.HasPrefix(cleanPath, "/") && !strings.HasPrefix(cleanPath, baseDir) {
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}

	// Строим полный путь
	fullPath := cleanPath

	if !(strings.HasPrefix(fullPath, baseDir) || strings.HasPrefix(fullPath, strings.TrimPrefix(baseDir, "/"))) {
		fullPath = filepath.Join(baseDir, cleanPath)
	}

	// Получаем абсолютные пути для проверки
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Проверяем, что результирующий путь находится внутри базовой директории
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path traversal detected")
	}

	return fullPath, nil
}

func buildTree(rootPath string) (*File, error) {
	dir, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	// Создаем корневой файл, потом к его children записываем остальные папки и файлы
	file := &File{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Size:     0,
		Children: nil,
	}
	if len(dir) == 0 {
		return file, nil
	}

	var children []File

	for _, entry := range dir {
		if entry.IsDir() {
			tree, err := buildTree(filepath.Join(rootPath, entry.Name()))
			if err != nil {
				return nil, err
			}

			children = append(children, *tree)
		} else {
			info, _ := entry.Info()
			children = append(children, File{
				Name:     entry.Name(),
				Path:     filepath.Join(rootPath, entry.Name()),
				IsDir:    false,
				Size:     info.Size(),
				Children: nil,
			})
		}
	}

	file.Children = children

	return file, nil
}

func getDir(rootPath string) (*File, error) {
	dir, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	file := &File{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Size:     0,
		Children: nil,
	}
	if len(dir) == 0 {
		return file, nil
	}

	for _, entry := range dir {
		if entry.IsDir() {
			file.AddChild(File{
				Name:     entry.Name(),
				Path:     filepath.Join(rootPath, entry.Name()),
				IsDir:    true,
				Size:     0,
				Children: nil,
			})
		} else {
			info, _ := entry.Info()
			file.AddChild(File{
				Name:     entry.Name(),
				Path:     filepath.Join(rootPath, entry.Name()),
				IsDir:    false,
				Size:     info.Size(),
				Children: nil,
			})
		}
	}

	return file, nil
}

// GetFilesTree return files tree without root dir
func GetFilesTree(cfg *configs.Config) ([]File, error) {
	file, err := buildTree(cfg.TorrentsDir)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[filehelpers] %s", err)
	}

	if file.Children != nil {
		return file.Children, nil
	} else {
		return []File{}, nil
	}
}

// GetFiles return files without root dir
func GetFiles(rootPath string) ([]File, error) {
	file, err := getDir(rootPath)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[filehelpers] %s", err)
	}

	if file.Children != nil {
		return file.Children, nil
	} else {
		return []File{}, nil
	}
}
