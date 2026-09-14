package assets

import "embed"

// FS contains the checked-in CSS, JavaScript, and local preview imagery.
//
//go:embed css js images fonts
var FS embed.FS
