package config

import (
	"context"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type WatchHandler interface {
	OnCreate(ctx context.Context, pathName string)
	OnUpdate(ctx context.Context, pathName string)
	OnDelete(ctx context.Context, pathName string)
	OnError(ctx context.Context, err error)
}

func StartWatcher(ctx context.Context, configFile string, handler WatchHandler) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "fsnotify.NewWatcher")
	}
	defer watcher.Close()

	cfBasename := filepath.Base(configFile)
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Errorf("Get events from watcher not ok")
					return
				}

				curBasename := filepath.Base(event.Name)
				if cfBasename != curBasename {
					log.Debugf("ignore event %s", event)
					continue
				}

				if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					// not care about CHMOD
					continue
				}

				log.Infof("Watcher event happened: %s", event)

				if event.Op&fsnotify.Create == fsnotify.Create {
					handler.OnCreate(ctx, event.Name)
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					handler.OnUpdate(ctx, event.Name)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					handler.OnDelete(ctx, event.Name)
				} else {
					log.Warningf("Unhandle watcher event: %s", event)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Errorf("Get errors from watcher not ok")
					return
				}
				handler.OnError(ctx, err)
			}
		}
	}()

	dirname := filepath.Dir(configFile)
	if err := watcher.Add(dirname); err != nil {
		return errors.Wrapf(err, "watch config file dir %s", dirname)
	}
	<-done
	return nil
}
