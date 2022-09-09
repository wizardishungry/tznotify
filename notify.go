// build unix
package tznotify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

const PathSeparator = string(os.PathSeparator)
const globalLocalTime = "/etc/localtime"

type Watcher struct {
	c         chan notify.EventInfo
	locations chan *time.Location
	errors    chan error
}

func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) Locations() <-chan *time.Location {
	return w.locations
}

func (w *Watcher) Close() {
	notify.Stop(w.c)
	close(w.c)
}

func New() (*Watcher, error) {
	return NewFromPath(globalLocalTime)
}

func Locations(w *Watcher) <-chan *time.Location {
	go func() {
		for range w.Errors() {
		}
	}()
	return w.Locations()
}

func NewFromPath(localTimeSymlink string) (w *Watcher, finalErr error) {

	suffix := filepath.Base(localTimeSymlink)
	dir, err := filepath.EvalSymlinks(filepath.Dir(localTimeSymlink))
	if err != nil {
		return nil, err
	}
	localTimeSymlink = filepath.Join([]string{dir, suffix}...)

	c := make(chan notify.EventInfo, 1)
	const events = notify.Create | notify.Rename | notify.Remove
	if err := notify.Watch(filepath.Dir(localTimeSymlink), c, events); err != nil {
		return nil, err
	}

	locations := make(chan *time.Location)
	errors := make(chan error)

	go func() {
		defer close(errors)
		defer close(locations)
		for {
			select {
			case ei, ok := <-c:
				if !ok {
					return
				}
				if ei.Path() != localTimeSymlink {
					continue
				}
				loc, err := ParseSymlink(localTimeSymlink)
				if err != nil {
					errors <- err
					continue
				}
				locations <- loc
			}
		}
	}()
	return &Watcher{
		c:         c,
		locations: locations,
		errors:    errors,
	}, nil
}

func ParseSymlink(path string) (*time.Location, error) {
	// See initLocal in time package & zoneinfo_* for other platforms
	link, err := os.Readlink(path)
	if err != nil {
		return nil, err
	}
	l := strings.Split(link, PathSeparator)
	if len(l) < 2 {
		return nil, fmt.Errorf("bad link: %s", link)
	}
	l = l[len(l)-2:]
	var sb strings.Builder // TODO not reused
	sb.WriteString(l[0])
	sb.WriteRune(os.PathSeparator)
	sb.WriteString(l[1])
	z := sb.String()
	sb.Reset()
	return time.LoadLocation(z)
}
