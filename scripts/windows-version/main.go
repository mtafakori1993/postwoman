// Command windows-version stamps a Windows PE executable with the release
// version metadata (fixed file/product version and StringFileInfo keys).
//
// It exists because the Wails resource generation is not always reflected in the
// produced executable on CI; stamping afterwards guarantees that the app and the
// NSIS installer report the correct version in their file properties.
//
// Usage:
//
//	go run . -file build/bin/Postwoman.exe -version 1.4.1 -product Postwoman
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	file := flag.String("file", "", "path to the Windows executable to stamp")
	rawVersion := flag.String("version", "", "version in MAJOR.MINOR.PATCH form")
	product := flag.String("product", "Postwoman", "product name")
	company := flag.String("company", "Postwoman", "company name")
	copyright := flag.String("copyright", "MIT licensed - Postwoman authors", "legal copyright")
	comments := flag.String("comments", "Modern API development platform", "comments")
	description := flag.String("description", "", "file description (defaults to product name)")
	printOnly := flag.Bool("print", false, "print the current version metadata and exit")
	flag.Parse()

	if *file == "" {
		fail("missing -file")
	}
	parts, err := parseVersion(*rawVersion)
	if err != nil {
		fail(err.Error())
	}
	if *description == "" {
		*description = *product
	}

	before, err := readVersion(*file)
	if err != nil {
		fail(err.Error())
	}
	fmt.Printf("windows-version: before product=%s file=%s\n", before.ProductVersion, before.FileVersion)
	if *printOnly {
		return
	}

	src, err := os.Open(*file)
	if err != nil {
		fail(err.Error())
	}
	resources, err := winres.LoadFromEXE(src)
	if err != nil {
		// A binary compiled without resources (no icon/manifest) has no resource
		// directory yet; start from an empty set and let winres add one.
		if !strings.Contains(err.Error(), "resource directory") {
			src.Close()
			fail(fmt.Sprintf("load resources: %v", err))
		}
		resources = &winres.ResourceSet{}
	}

	info := version.Info{}
	info.FileVersion = parts
	info.ProductVersion = parts
	for key, value := range map[string]string{
		version.ProductVersion:  *rawVersion,
		version.FileVersion:     *rawVersion,
		version.ProductName:     *product,
		version.CompanyName:     *company,
		version.LegalCopyright:  *copyright,
		version.Comments:        *comments,
		version.FileDescription: *description,
	} {
		if err := info.Set(version.LangDefault, key, value); err != nil {
			src.Close()
			fail(fmt.Sprintf("set %s: %v", key, err))
		}
	}
	resources.SetVersionInfo(info)

	temp := *file + ".stamped"
	dst, err := os.Create(temp)
	if err != nil {
		src.Close()
		fail(err.Error())
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		src.Close()
		dst.Close()
		fail(err.Error())
	}
	writeErr := resources.WriteToEXE(dst, src, winres.WithAuthenticode(winres.IgnoreSignature))
	src.Close()
	closeErr := dst.Close()
	if writeErr != nil {
		fail(fmt.Sprintf("write resources: %v", writeErr))
	}
	if closeErr != nil {
		fail(closeErr.Error())
	}
	if err := os.Rename(temp, *file); err != nil {
		// Windows cannot always rename over an existing file.
		if removeErr := os.Remove(*file); removeErr != nil {
			fail(fmt.Sprintf("replace %s: %v", *file, removeErr))
		}
		if err := os.Rename(temp, *file); err != nil {
			fail(fmt.Sprintf("replace %s: %v", *file, err))
		}
	}

	after, err := readVersion(*file)
	if err != nil {
		fail(err.Error())
	}
	fmt.Printf("windows-version: after  product=%s file=%s\n", after.ProductVersion, after.FileVersion)
}

func parseVersion(raw string) ([4]uint16, error) {
	var parts [4]uint16
	chunks := strings.Split(strings.TrimSpace(raw), ".")
	if len(chunks) < 3 || len(chunks) > 4 {
		return parts, fmt.Errorf("version %q must be MAJOR.MINOR.PATCH", raw)
	}
	for index, chunk := range chunks {
		value, err := strconv.ParseUint(chunk, 10, 16)
		if err != nil {
			return parts, fmt.Errorf("version %q has an invalid segment %q", raw, chunk)
		}
		parts[index] = uint16(value)
	}
	return parts, nil
}

func readVersion(file string) (versionStrings, error) {
	handle, err := os.Open(file)
	if err != nil {
		return versionStrings{}, err
	}
	defer handle.Close()
	resources, err := winres.LoadFromEXE(handle)
	if err != nil {
		if strings.Contains(err.Error(), "resource directory") {
			return versionStrings{ProductVersion: "<none>", FileVersion: "<none>"}, nil
		}
		return versionStrings{}, fmt.Errorf("read resources from %s: %w", file, err)
	}
	var found *version.Info
	resources.WalkType(winres.RT_VERSION, func(_ winres.Identifier, _ uint16, data []byte) bool {
		info, parseErr := version.FromBytes(data)
		if parseErr == nil {
			found = info
			return false
		}
		return true
	})
	if found == nil {
		return versionStrings{ProductVersion: "<none>", FileVersion: "<none>"}, nil
	}
	return versionStrings{
		ProductVersion: formatVersion(found.ProductVersion),
		FileVersion:    formatVersion(found.FileVersion),
	}, nil
}

type versionStrings struct {
	ProductVersion string
	FileVersion    string
}

func formatVersion(parts [4]uint16) string {
	return fmt.Sprintf("%d.%d.%d.%d", parts[0], parts[1], parts[2], parts[3])
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "windows-version: "+message)
	os.Exit(1)
}
