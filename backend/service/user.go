package service

import (
	"blog/dao"
	"blog/middleware/hotkey"
	"blog/model"
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type user struct {
	cache *hotkey.HotKeyWithCache
}

func init() {
	userService = newUser()
}

var userService *user

func GetUser() *user {
	return userService
}

func newUser() *user {
	userhotkey, err := hotkey.NewHotkey(&hotkey.Option{
		HotKeyCnt:     50000,
		AutoCache:     true,
		LocalCacheCnt: 50000,
		LocalCache:    hotkey.NewLocalCache(50000),
		CacheMs:       1000 * 60,
	})
	if err != nil {
		logrus.Panicf("init hotkey failed:%s", err.Error())
	}
	return &user{
		cache: userhotkey,
	}
}

func (u *user) AutoCreateIfNotExist(ctx context.Context, address string, alias string) (err error) {
	if _, ok := u.cache.Get(address); ok {
		u.cache.Add(address, 1)
		return nil
	}
	userdao := dao.GetUser()
	var user model.User
	defer func() {
		if err == nil {
			u.cache.Add(address, 1)
		}
	}()
	user, err = userdao.FindUserByAddress(ctx, address)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Errorf("find user %v failed: %s", address, err.Error())
			return
		} else {
			err = userdao.CreateUser(&model.User{
				Address: address,
				Alias:   alias,
			})
			if err != nil {
				logrus.Errorf("create user %v failed: %s", address, err.Error())
				return
			}
		}
	}
	u.cache.AddWithValue(address, &user, 1)
	return
}

func (u *user) FindUserByAddress(ctx context.Context, address string) (view *model.User, err error) {
	if userany, ok := u.cache.Get(address); ok {
		view = userany.(*model.User)
		return
	}
	var user model.User
	user, err = dao.GetUser().FindUserByAddress(ctx, address)
	if err != nil {
		logrus.Errorf("find user %v failed: %s", address, err.Error())
	}
	u.cache.AddWithValue(address, &user, 1)
	view = &user
	return
}
