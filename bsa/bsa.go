package bsa

import (
	"path/filepath"
	"strings"
)

// SanitizePath sanitizes a path for tes hash calculation.
// Basically makes it lowercase and replaces slashes with backslashes.
func SanitizePath(path string) string {
	return strings.ToLower(strings.ReplaceAll(path, "/", "\\"))
}

var tesHashExtMap = map[string]uint64{
	".kf":  0x80,
	".nif": 0x8000,
	".dds": 0x8080,
	".wav": 0x80000000,
}

// TesHash calculates the BSA hash for the specified file name.
// The provided path will be treated as case insensitive, and may contain either
// slashes or backslashes as path separators (will be sanitized)
func TesHash(path string) uint64 {
	path = SanitizePath(path)
	ext := filepath.Ext(path)
	root := filepath.Base(path)
	root = root[:len(root)-len(ext)]

	chars := make([]uint64, len(root))
	for i, x := range root {
		chars[i] = uint64(x)
	}
	hash1 := chars[len(chars)-1]
	if len(chars) > 2 {
		hash1 |= chars[len(chars)-2] << 8
	}
	hash1 |= uint64(len(chars)<<16) | chars[0]<<24 | tesHashExtMap[ext]
	var uintMask, hash2, hash3 uint64 = 0xFFFFFFFF, 0, 0
	for _, char := range chars[1 : len(chars)-2] {
		hash2 = ((hash2 * 0x1003F) + char) & uintMask
	}

	for _, char := range ext {
		hash3 = ((hash3 * 0x1003F) + uint64(char)) & uintMask
	}
	hash2 = (hash2 + hash3) & uintMask
	return (hash2 << 32) + hash1
}

// TesHashable ...
type TesHashable interface {
	TesHash() uint64
}

// Node represents a file or a folder inside a BSA archive
type Node struct {
	Name    string
	tesHash *uint64
}

// TesHash returns a cached version of the TesHash associated to this object
func (n *Node) TesHash() uint64 {
	if n.tesHash == nil {
		*n.tesHash = TesHash(n.Name)
	}
	return *n.tesHash
}

// ByTesHash implements sort.Interface based on TesHash() result
type ByTesHash []TesHashable

func (a ByTesHash) Len() int           { return len(a) }
func (a ByTesHash) Less(i, j int) bool { return a[i].TesHash() < a[j].TesHash() }
func (a ByTesHash) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// Root represents the root of a BSA archive.
// It can only contain subfolders
type Root struct {
	Node
	Subfolders []Folder
}

// Folder represents a folder inside a BSA archive.
// Use NewFolder to instantiate a Folder struct
type Folder struct {
	Node
	Bsa        Bsa
	files      []File
	Subfolders *[]Folder

	sortedFiles *[]File
}

func (f *Folder) addFile(newFile File) {
	f.files = append(f.files, newFile)
	f.sortedFiles = nil
}

// SortedFiles returns all files in the current folder, sorted by their tes hash
// This function's result is cached, meaning that calling it multiple times
// without editing the files slice takes θ(1)
func (f *Folder) SortedFiles() *[]File {
	// a := File{}
	if f.sortedFiles == nil {
		/*a := ByTesHash(f.files)
		f.sortedFiles = sort.Sort(a)*/
	}
	return f.sortedFiles
}

// NewFolder instantiates a new Folder
func NewFolder(path string) Folder {
	return Folder{
		Node: Node{
			Name: SanitizePath(path),
		},
	}
}

// File represents a folder inside a BSA archive.
type File struct {
	Node
	Folder       Folder
	RecordOffset *uint64
}

// Bsa represents a BSA archive
type Bsa struct{}
