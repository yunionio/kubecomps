package ceph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ceph/go-ceph/rados"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type Client struct {
	User     string
	Key      string
	Monitors []string
}

func NewClient(user string, key string, monitors ...string) (*Client, error) {
	cli := &Client{
		User:     user,
		Key:      key,
		Monitors: monitors,
	}
	_, err := cli.withCluster(func(conn *rados.Conn) (i interface{}, err error) {
		return nil, nil
	})
	return cli, err
}

func (c *Client) withCluster(doFunc func(conn *rados.Conn) (interface{}, error)) (interface{}, error) {
	conn, err := rados.NewConnWithUser(c.User)
	if err != nil {
		return nil, errors.Wrap(err, "new rados connection")
	}
	for key, timeout := range map[string]int64{
		// "rados_osd_op_timeout": api.RBD_DEFAULT_OSD_TIMEOUT,
		"rados_osd_op_timeout": 10,
		// "rados_mon_op_timeout": api.RBD_DEFAULT_MON_TIMEOUT,
		"rados_mon_op_timeout": 10,
		// "client_mount_timeout": api.RBD_DEFAULT_MOUNT_TIMEOUT,
		"client_mount_timeout": 10,
	} {
		if err := conn.SetConfigOption(key, fmt.Sprintf("%d", timeout)); err != nil {
			return nil, errors.Wrapf(err, "set timeout option %s=%d", key, timeout)
		}
	}
	if err := conn.SetConfigOption("mon_host", strings.Join(c.Monitors, ",")); err != nil {
		return nil, errors.Wrapf(err, "set monitors %v", c.Monitors)
	}
	if err := conn.SetConfigOption("key", c.Key); err != nil {
		return nil, errors.Wrapf(err, "set key %s", c.Key)
	}
	ti, _ := conn.GetConfigOption("rados_mon_op_timeout")
	log.Errorf("Get config: %s", ti)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	errCh := make(chan error)
	go func() {
		log.Infof("start connecting ceph rados...")
		if err := conn.Connect(); err != nil {
			errCh <- errors.Wrap(err, "connecting error")
		} else {
			errCh <- nil
		}
		return
	}()
	select {
	case <-ctx.Done():
		log.Errorf("connection: %v", ctx.Err())
		err = ctx.Err()
	case nErr := <-errCh:
		log.Errorf("recieve error: %v", nErr)
		err = nErr
	}
	if err != nil {
		return nil, err
	}
	defer conn.Shutdown()
	return doFunc(conn)
}

func (c *Client) ListPools() ([]string, error) {
	ret, err := c.withCluster(func(conn *rados.Conn) (i interface{}, err error) {
		return conn.ListPools()
	})
	if err != nil {
		return nil, err
	}
	return ret.([]string), nil
}

func (c *Client) ListPoolsNoDefault() ([]string, error) {
	pools, err := c.ListPools()
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	for _, p := range pools {
		if strings.Contains(p, ".rgw") {
			continue
		}
		ret = append(ret, p)
	}
	return ret, nil
}
