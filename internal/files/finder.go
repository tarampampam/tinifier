package files

import (
	"context"
	"os"
	"path/filepath"
)

// finderOptions is a set of options for the finder.
type finderOptions struct {
	recursive bool
	filesExt  map[string]struct{}
}

// ExtIsAllowed checks if the given extension is allowed (exist in the extensions map).
func (o *finderOptions) ExtIsAllowed(ext string) (ok bool) {
	if o.filesExt == nil {
		return true
	}

	_, ok = o.filesExt[ext]

	return
}

// FinderOption is a function that can be used to modify the finder options.
type FinderOption func(*finderOptions)

// WithRecursive sets the recursive option to true.
func WithRecursive(recursive bool) FinderOption {
	return func(o *finderOptions) { o.recursive = recursive }
}

// WithFilesExt sets the file extensions (WITHOUT leading dot) to the finder options.
func WithFilesExt(filesExt ...string) FinderOption {
	return func(o *finderOptions) {
		if len(filesExt) == 0 {
			return
		}

		o.filesExt = make(map[string]struct{}, len(filesExt))

		for _, ext := range filesExt { // burn the map
			o.filesExt[ext] = struct{}{}
		}
	}
}

// FindFiles finds all files in the given locations (without duplicates).
// Important note - for the direct files' extension checking will be ignored.
// TODO fn should returns bool (shouldStop)
func FindFiles(ctx context.Context, where []string, fn func(absPath string), opts ...FinderOption) error { //nolint:funlen,gocognit,gocyclo,lll
	if len(where) == 0 { // fast terminator
		return nil
	}

	var options finderOptions

	for _, opt := range opts {
		opt(&options)
	}

	var locations = make(map[string]struct{}, len(where)) // unique location paths

	for i := 0; i < len(where); i++ { // convert relative paths to absolute
		if !filepath.IsAbs(where[i]) {
			if abs, err := filepath.Abs(where[i]); err != nil {
				return err
			} else {
				locations[abs] = struct{}{}
			}
		} else {
			locations[where[i]] = struct{}{}
		}
	}

	var history = make(map[string]struct{})

	for location := range locations {
		if err := ctx.Err(); err != nil {
			return err
		}

		locationStat, statErr := os.Stat(location)
		if statErr != nil {
			return statErr
		}

		switch mode := locationStat.Mode(); {
		case mode.IsRegular(): // regular file (eg.: `./file.png`)
			if _, ok := history[location]; !ok {
				history[location] = struct{}{}

				fn(location)
			}

		case mode.IsDir(): // directory (eg.: `./path/to/images`)
			if options.recursive { //nolint:nestif // deep directory search
				if err := filepath.Walk(location, func(path string, info os.FileInfo, err error) error {
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}

					if err != nil || !info.Mode().IsRegular() {
						return err
					}

					if ext := filepath.Ext(info.Name()); len(ext) > 0 && options.ExtIsAllowed(ext[1:]) {
						if _, ok := history[path]; !ok {
							history[path] = struct{}{}

							fn(path)
						}
					}

					return err
				}); err != nil {
					return err
				}
			} else { // flat directory search
				files, readDirErr := os.ReadDir(location)
				if readDirErr != nil {
					return readDirErr
				}

				for _, file := range files {
					if err := ctx.Err(); err != nil {
						return err
					}

					if file.Type().IsRegular() {
						var path = filepath.Join(location, file.Name())

						if ext := filepath.Ext(file.Name()); len(ext) > 0 && options.ExtIsAllowed(ext[1:]) {
							if _, ok := history[path]; !ok {
								history[path] = struct{}{}

								fn(path)
							}
						}
					}
				}
			}
		}
	}

	return nil
}
