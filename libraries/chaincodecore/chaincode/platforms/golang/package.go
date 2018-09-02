
package golang

import (
"errors"
"fmt"
"strings"

"os"
"path/filepath"

flogging "gocoin/libraries/log"
ccutil "gocoin/libraries/chaincodecore/chaincode/platforms/util"
pb "gocoin/libraries/chaincodecore/protos/peer"
)

var includeFileTypes = map[string]bool{
	".c":    true,
	".h":    true,
	".go":   true,
	".yaml": true,
	".json": true,
}

var logger = flogging.New("golang-platform")

func getCodeFromFS(path string) (codegopath string, err error) {
	logger.Debug("getCodeFromFS %s", path)
	gopath, err := getGopath()
	if err != nil {
		return "", err
	}

	tmppath := filepath.Join(gopath, "src", path)
	if err := ccutil.IsCodeExist(tmppath); err != nil {
		return "", fmt.Errorf("code does not exist %s", err)
	}

	return gopath, nil
}

type CodeDescriptor struct {
	Gopath, Pkg string
	Cleanup     func()
}

// collectChaincodeFiles collects chaincode files. If path is a HTTP(s) url it
// downloads the code first.
//
//NOTE: for dev mode, user builds and runs chaincode manually. The name provided
//by the user is equivalent to the path.
func getCode(spec *pb.ChaincodeSpec) (*CodeDescriptor, error) {
	if spec == nil {
		return nil, errors.New("Cannot collect files from nil spec")
	}

	chaincodeID := spec.ChaincodeId
	if chaincodeID == nil || chaincodeID.Path == "" {
		return nil, errors.New("Cannot collect files from empty chaincode path")
	}

	// code root will point to the directory where the code exists
	var gopath string
	gopath, err := getCodeFromFS(chaincodeID.Path)
	if err != nil {
		return nil, fmt.Errorf("Error getting code %s", err)
	}

	return &CodeDescriptor{Gopath: gopath, Pkg: chaincodeID.Path, Cleanup: nil}, nil
}

type SourceDescriptor struct {
	Name, Path string
	Info       os.FileInfo
}
type SourceMap map[string]SourceDescriptor

type Sources []SourceDescriptor

func (s Sources) Len() int {
	return len(s)
}

func (s Sources) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Sources) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}

func findSource(gopath, pkg string) (SourceMap, error) {
	sources := make(SourceMap)
	tld := filepath.Join(gopath, "src", pkg)
	walkFn := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			if path == tld {
				// We dont want to import any directories, but we don't want to stop processing
				// at the TLD either.
				return nil
			}

			// Do not recurse
			logger.Debug("skipping dir: %s", path)
			return filepath.SkipDir
		}

		ext := filepath.Ext(path)
		// we only want 'fileTypes' source files at this point
		if _, ok := includeFileTypes[ext]; ok != true {
			return nil
		}

		name, err := filepath.Rel(gopath, path)
		if err != nil {
			return fmt.Errorf("error obtaining relative path for %s: %s", path, err)
		}

		sources[name] = SourceDescriptor{Name: name, Path: path, Info: info}

		return nil
	}

	if err := filepath.Walk(tld, walkFn); err != nil {
		return nil, fmt.Errorf("Error walking directory: %s", err)
	}

	return sources, nil
}

