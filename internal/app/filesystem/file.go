package filesystem

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type File struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size,omitempty"`
	Children []File `json:"children,omitempty"`
}

func (f *File) AddChild(child File) {
	f.Children = append(f.Children, child) // но тогда Children должен быть []*File
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

func getFilesFromDir(rootPath string) (*File, error) {
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
func GetFilesTree() ([]File, error) {
	file, err := buildTree("torrents")
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[file] %s", err)
	}

	if file.Children != nil {
		return file.Children, nil
	} else {
		return make([]File, 0), nil
	}
}

// GetFiles return files without root dir
func GetFiles(rootPath string) ([]File, error) {
	file, err := getFilesFromDir(rootPath)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[file] %s", err)
	}

	if file.Children != nil {
		return file.Children, nil
	} else {
		return make([]File, 0), nil
	}
}
