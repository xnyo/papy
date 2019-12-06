package flags

import "strings"

// ArchiveFlags represents the flags associated to a BSA file
type ArchiveFlags uint32

const (
	// IncludeDirectoryNames ...
	IncludeDirectoryNames ArchiveFlags = 1 << iota

	// IncludeFileNames ...
	IncludeFileNames

	// Compressed ...
	Compressed

	// RetainDirectoryNames ...
	RetainDirectoryNames

	// RetainFileNames ...
	RetainFileNames

	// RetainFileNamesOffsets ...
	RetainFileNamesOffsets

	// Xbox360 ...
	Xbox360

	// RetainStringsDuringStartup ...
	RetainStringsDuringStartup

	// EmbedFileNames ...
	EmbedFileNames

	// XMem ...
	XMem

	// DefaultArchiveFlags are the flags set in official BSA archives
	DefaultArchiveFlags = IncludeDirectoryNames | IncludeFileNames | RetainStringsDuringStartup
)

// FileFlags represents the file type flags of a BSA archive
type FileFlags uint32

const (
	// None ...
	None = 1 << iota

	// Meshes ...
	Meshes

	// Textures ...
	Textures

	// Menus ...
	Menus

	// Sounds ...
	Sounds

	// Voices ...
	Voices

	// Shaders ...
	Shaders

	// Trees ...
	Trees

	// Fonts ...
	Fonts

	// Miscellaneous ...
	Miscellaneous
)

// ExtToFileFlags returns the FileFlags from a string extension
// the extension string must be in the format '.xyz' (eg: '.nif')
func ExtToFileFlags(ext string) FileFlags {
	extMap := map[string]FileFlags{
		".nif": Meshes,
	}
	v, ok := extMap[strings.ToLower(ext)]
	if !ok {
		return Miscellaneous
	}
	return v
}
