package main

import (
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

func (f *File) AddChild(child *File) {
	f.Children = append(f.Children, *child)
}

func main() {
	file, err := buildTree("torrents")
	if err != nil {
		log.Println(err)
	}

	printTree(*file, 0)
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

func printTree(file File, level int) {
	// Символ для директории или файла
	prefix := "📁 "
	if !file.IsDir {
		prefix = "📄 "
	}

	log.Printf("%s%s%s\n", strings.Repeat(" ", level*4), prefix, file.Name)

	for _, child := range file.Children {
		printTree(child, level+1)
	}
}
