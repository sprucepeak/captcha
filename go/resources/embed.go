package resources

import "embed"

//go:embed resource/* template/*
var BuiltinFS embed.FS

