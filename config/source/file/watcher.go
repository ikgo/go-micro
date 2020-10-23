//+build !linux

package file

import (
	"os"

	"github.com/asim/go-micro/v3/config/source"
	"github.com/fsnotify/fsnotify"
)

type watcher struct {
	f *file

	fw   *fsnotify.Watcher
	exit chan bool
}

func newWatcher(f *file) (source.Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw.Add(f.path)

	return &watcher{
		f:    f,
		fw:   fw,
		exit: make(chan bool),
	}, nil
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	// is it closed?
	select {
	case <-w.exit:
		return nil, source.ErrWatcherStopped
	default:
	}

	for {
		// try get the event
		select {
		case event, _ := <-w.fw.Events:
			if event.Op == fsnotify.Rename {
				// check existence of file, and add watch again
				_, err := os.Stat(event.Name)
				if err == nil || os.IsExist(err) {
					w.fw.Add(event.Name)
				}
			}

			// ARCH: Darwin Kernel Version 18.7.0
			// ioutil.WriteFile truncates it before writing, but the problem is that
			// you will receive two events(fsnotify.Chmod and fsnotify.Write).
			// We can solve this problem by ignoring fsnotify.Chmod event.
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			c, err := w.f.Read()
			if err != nil {
				return nil, err
			}
			return c, nil
		case err := <-w.fw.Errors:
			return nil, err
		case <-w.exit:
			return nil, source.ErrWatcherStopped
		}
	}
}

func (w *watcher) Stop() error {
	return w.fw.Close()
}
