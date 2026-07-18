package parser

import (
	"os"
	"path/filepath"
	"strings"
)

type NoteFile struct {
	Name    string
	Path    string
	Content string
}

func LoadNoteFiles(subjectDir, subject string) ([]NoteFile, error) {
	notesDir := filepath.Join(subjectDir, subject, "notes")
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []NoteFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(notesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		files = append(files, NoteFile{
			Name:    strings.TrimSuffix(entry.Name(), ".md"),
			Path:    path,
			Content: string(data),
		})
	}

	return files, nil
}
